# tools/actor/

[Up](../README.md)

## Overview

An **Actor** is a self-contained goroutine that owns its data exclusively. All access to that data must go through the actor's message queue. Because only one goroutine ever reads or writes the actor's internal state, no mutexes are needed.

This is an implementation of the [Actor Model](https://en.wikipedia.org/wiki/Actor_model) tailored for Go channel idioms.

---

## Architecture

```
                    ┌──────────────────────────────┐
  NotifyActor() ──> │         MessageQueue          │
  SendActor()   ──> │  inputChan  ──> []messages    │
                    │                   │           │
                    │           executorChan         │
                    └───────────────────┼───────────┘
                                        │
                    ┌───────────────────▼───────────┐
                    │         Actor goroutine        │
                    │   select {                     │
                    │     case msg <- executorChan:  │
                    │       processMessage(msg)      │
                    │     case msg <- CallbackChan:  │
                    │       processReply(msg)        │
                    │     case <- stopper:           │
                    │       done = true              │
                    │   }                            │
                    └────────────────────────────────┘
```

**Key invariant**: the actor goroutine is the only reader/writer of the actor's internal fields.

---

## Public Interface

### Types

| Type | Purpose |
|---|---|
| `Actor` | Core struct. Embed or compose in your own type. |
| `Communication` | Interface: `NotifyActor` + `SendActor`. Expose this to callers to break cyclic imports. |
| `Manageable` | Interface: `Communication` + `Start` / `Stop` / `PrepareToStop`. |
| `ActorMethod` | Interface for registering typed handlers. |
| `NoReply` | Embed in method structs that never expect a reply from the actor (documentary only). |

### Lifecycle

```go
// Create
act := actor.New("MyActor")

// Register handlers BEFORE Start()
act.AddMethod(MyRequest{}, myHandler, myValidator)
act.AddReply(MyReply{}, myReplyHandler, nil)

// Start the goroutine
act.Start()

// Stop (immediate, may drop queued messages)
act.Stop()

// Graceful stop: drain queue, then stop
done := act.PrepareToStop()
<-done
act.Stop()
```

### Sending Messages

| Method | Blocks? | Reply expected? | Use when |
|---|---|---|---|
| `NotifyActor(msg)` | No | No (`ShouldBeRepliedTo = false`) | Fire-and-forget events |
| `SendActor(msg, chan)` | No (caller polls chan) | Yes | RPC-style calls |

### Replying from a Handler

Every handler that receives a message via `SendActor` **must** terminate with exactly one of:

```go
a.Reply(msg, replyMsg)   // send a reply payload
a.NoReply(msg)           // acknowledge without payload
```

Failure to call either causes the caller's reply channel to block forever.

---

## Defining an Actor

```go
// 1. Declare method structs in a separate package to avoid cyclic deps.
// e.g. package mymethods
type DoWork struct { Param string }
type DoWorkReply struct { Result int }

// 2. Declare your actor
type MyActor struct {
    *actor.Actor
    // your private state here — safe, only this goroutine touches it
    counter int
}

func New() *MyActor {
    a := &MyActor{
        Actor: actor.New("MyActor"),
    }
    a.AddMethod(mymethods.DoWork{}, a.doWork, a.validateDoWork)
    a.Start()
    return a
}

// 3. Implement the Communication interface for external callers
func (a *MyActor) NotifyActor(msg *message.Message) { a.Actor.NotifyActor(msg) }
func (a *MyActor) SendActor(msg *message.Message, cb chan *message.Message) { a.Actor.SendActor(msg, cb) }

// 4. Implement handlers
func (a *MyActor) doWork(msg *message.Message) bool {
    req := msg.Content.(mymethods.DoWork)
    a.counter++
    reply := msg.Reply()
    reply.Content = mymethods.DoWorkReply{Result: a.counter}
    a.Reply(msg, reply)
    return true
}

func (a *MyActor) validateDoWork(msg *message.Message) []error {
    // return nil for no errors
    return nil
}
```

### Calling an Actor

```go
act := New()

// Fire-and-forget
act.NotifyActor(message.Create(nil, mymethods.DoWork{Param: "x"}, nil))

// With reply (blocking pattern — safe from outside an actor)
resChan := make(chan *message.Message)
defer close(resChan)
act.SendActor(message.Create(nil, mymethods.DoWork{Param: "x"}, nil), resChan)
result := <-resChan
```

---

## Actor-to-Actor Communication

When one actor needs to call another, **do not block on the reply from inside a handler**. The actor goroutine must remain free to process replies on `CallbackChan`.

### ✅ Correct: Async via `CallbackChan`

```go
// Inside Actor A's handler:
func (a *ActorA) handleRequest(msg *message.Message) bool {
    outMsg := message.Create(nil, othermethods.DoThing{}, MyReply{})
    // Send to Actor B; when B replies, it will arrive on A's CallbackChan
    actorB.SendActor(outMsg, a.GetCallbackChan())
    // Reply to our caller immediately (or defer — your choice)
    a.Reply(msg, msg.Reply())
    return true
}

// Actor A must have a reply handler registered for MyReply{}
a.AddReply(MyReply{}, a.handleMyReply, nil)
```

### ⚠️ Risky: Blocking inside a handler (deadlock risk)

```go
// Inside Actor A's handler:
func (a *ActorA) handleRequest(msg *message.Message) bool {
    localChan := make(chan *message.Message)
    defer close(localChan)
    actorB.SendActor(outMsg, localChan)
    <-localChan  // ⚠️ BLOCKS the executor goroutine
    // ...
    a.Reply(msg, msg.Reply())
    return true
}
```

This pattern is **only safe** if Actor B **never** calls back into Actor A to complete its work. If any chain leads A → B → ... → A, it will deadlock. The queue goroutine will still accept new messages, but the executor goroutine cannot process them.

See issue [`20260223_actor_deadlock_risk.md`](../../../../issues/20260223_actor_deadlock_risk.md).

---

## Special Messages

| Type | Direction | Meaning |
|---|---|---|
| `ActorStarted` | Internal notify on `Start()` | Actor is live; good place for init logic |
| `ActorStop` | Trigger `Stop()` via message | Graceful stop from message handler context |
| `ActorAboutToStop` | Internal, converted from `ActorStop` | Hook to run cleanup before goroutine exits |
| `ActorError` | Application-defined | Carry error payloads in replies |

---

## Caveats & Pitfalls

### 1. Deadlock: blocking actor-to-actor call
**Risk: Medium.** If a handler blocks waiting on a reply from another actor that itself needs to call back into this actor, both goroutines deadlock. Use `CallbackChan` (async reply) instead. See the linked issue.

### 2. Data race on `mq.messages` slice
**Risk: Low-Medium.** The internal `[]internalMessage` slice and `dontAcceptNewMessages` bool in `MessageQueue` are accessed from two goroutines: the queue loop and the caller of `PrepareToStop()` → `Length()`. This is a technically correct data race per Go's memory model, even if it rarely manifests visibly. Add a mutex or restructure if this becomes a concern under the race detector.

### 3. Reflection-based dispatch is fragile
**Risk: Low.** Method handlers are keyed by `reflect.TypeOf(method)`. The type must match exactly (value vs. pointer, full package path). Renames, moves, or accidental shadowing will silently fail at runtime with an "Unexpected message" warning rather than a compile error.

### 4. `NoReply` on a `Send` path hangs the caller
**Risk: Medium.** If a handler calls `a.NoReply(msg)` on a message that arrived via `SendActor`, the caller's reply channel will block forever. The flag `msg.ShouldBeRepliedTo` is checked at runtime, but there is no compile-time enforcement. Always trace the call site.

### 5. Method not registered → silent auto-reply
**Risk: Low.** If a message arrives with an unregistered type, `processMessage` logs a warning and auto-replies with an empty message. This can mask bugs. Validators should be provided for all methods.

### 6. `Stop()` is not graceful by default
`a.Stop()` sends to `stopChan` which terminates the queue loop immediately, potentially dropping queued messages. Use `PrepareToStop()` + wait + `Stop()` for graceful shutdown.

---

## Testing Actors (The "Stopper" Pattern)

Because actors process messages asynchronously, testing them in a synchronous test function can be challenging. A common pattern to solve this is using **Stoppers** in your mock or fake actors.

### What is a Stopper?
A "Stopper" is a mechanism that allows a test to block until the actor receives a specific message. It is typically implemented in a test double (like a `FakeController` or `FakeActor`) as a map of channels:
`Stoppers map[string]chan *message.Message`. The key is typically the string reflection of a specific message type.

### How to use Stoppers in Tests
1. **Registration**: In the test, register a stopper channel for each expected message type.
2. **Triggering**: In the mock actor's message handlers, route incoming messages to a `triggerStopper()` helper method. If a stopper channel exists for that message type, the message is pushed onto the channel.
3. **Blocking & Asserting**: The test runner explicitly waits on the stopper channel. This halts test execution until the message arrives, avoiding race conditions and allowing the test to assert on the payload of the message exactly when it occurs.

---

## Tips

- Keep your method structs **outside** the actor's package. This prevents cyclic imports when two actors need to reference each other's method types.
- Expose only `actor.Communication` (not `*Actor`) in struct fields referencing other actors. This keeps the dependency surface minimal.
- Use `NotifyActor` for one-way state updates; use `SendActor` only when you genuinely need confirmation or a return value.
- Validators are optional but recommended: they let you reject malformed messages before the handler runs.
- Log at `DEBUG` in handlers; the actor framework already logs message lifecycle at `DEBUG`.
