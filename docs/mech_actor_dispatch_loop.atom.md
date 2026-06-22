---
id: mech_actor_dispatch_loop
human_name: "Actor Dispatch Loop"
type: MECHANIC
layer: IMPLEMENTATION
version: 1.0
status: STABLE
priority: 5
tags: [actor, dispatch, loop, logic]
parents:
  - [[mech_actor_pattern]]
dependents: []
---

# Actor Dispatch Loop

## INTENT
Centrally orchestrate the processing of incoming messages and callbacks, ensuring that the Actor remains responsive to both internal and external stimuli.

## THE RULE / LOGIC
- **Unified Queue Dispatch**: The dispatcher listens exclusively to the `MessageQueue` executor channel. 
- **Stimuli Redirector**: Replies arriving on `CallbackChan` must be redirected into the `MessageQueue` to ensure they are handled as ordered, non-blocking stimuli.
- **Sequential Execution**: All processing (Messages and Replies) must be gated by the `MessageQueue` to ensure exactly one stimulus is processed at a time.
- **Hierarchical Handler Lookup**: For any incoming stimulus, the dispatcher must search in order:
  1. **Reply Types**: If `msg.Type == Reply`, dispatch to `replyHandlers`.
  2. **Typed Handlers**: Modern `callHandlers` or `notificationHandlers` maps.
  3. **Internal Signals**: Special handling for `ActorStarted`, `ActorStop`.
  4. **Legacy Handlers**: The `methods` map for backward compatibility.
- **Acknowledge Stimulus**: Every stimulus processed from the queue (including replies) MUST send an acknowledgment back to the queue to unblock the next item.

## TECHNICAL INTERFACE (The Bridge)
- **Code Tag**: `@spec-link [[mech_actor_dispatch_loop]]`
- **Key Methods**:
  - `Actor.Start()`: Spawns the main `select` loop goroutine.
  - `Actor.processMessage(msg)`: Entry point for logic dispatching.
  - `Actor.processReply(msg)`: Entry point for handling callback responses.

## EXPECTATION (For Testing)
- **Sequentiality**: Multiple messages on different channels must be processed one at a time via the shared logic in `processMessage`.
- **Transparency**: Unhandled messages should be logged or cause a panic based on the `CrashOnUnhandled` flag.
