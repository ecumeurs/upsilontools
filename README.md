# Upsilon Tools

Shared utility library for Upsilon projects.

## Modules

### [`tools/`](tools/README.md)

| Package | Purpose |
|---|---|
| [`tools`](tools/tools.go) | Math primitives: `IntRange`, random, distance (2D/3D), linear interpolation, min/max |
| [`tools/actor`](tools/actor/README.md) | Actor model — single-goroutine resource ownership via message passing |
| [`tools/messagequeue`](tools/messagequeue/README.md) | Serial message queue backing the actor; guarantees one-message-at-a-time execution |
| [`tools/messagequeue/message`](tools/messagequeue/message/message.go) | `Message` struct: typed payload envelope with request/reply lifecycle tracking |

### [`logger/`](logger/)

Logrus-based structured logger shared across Upsilon projects.

---

## Core Design Philosophy

> **Share memory by communicating, not by locking.**

Instead of protecting shared state with mutexes, each stateful component wraps itself in an `Actor`. All access to its internal data goes through a serialized message queue. This eliminates the need for `sync.Mutex` or `sync.RWMutex` on hot data paths.

### Message Flow

```
Caller                    MessageQueue              Actor goroutine
  |                            |                         |
  |--- Send(msg, callback) --->|                         |
  |                            |-- executorChan <--------|
  |                            |                    processMessage(msg)
  |                            |                    handler(msg)
  |                            |                    Reply(msg, rpl)
  |                            |<-- executorReplyChan ---|
  |<-- callback <- reply ------|                         |
```

### Key Properties

- **Serial execution**: each actor processes one message at a time.
- **No shared state**: internal data is only ever touched by the actor's own goroutine.
- **Non-blocking callers**: `NotifyActor` (fire-and-forget) or `SendActor` (with reply channel) — callers are never blocked by the actor's internal work.
- **Typed dispatch via reflection**: method handlers are registered by struct type and dispatched via `reflect.TypeOf`.

---

## Known Issues

See [`/issues/`](../issues/) at the workspace root for tracked issues and risks.
