---
id: mech_actor_lifecycle
human_name: "Actor Lifecycle Management"
type: MECHANIC
layer: IMPLEMENTATION
version: 1.0
status: DRAFT
priority: 3
tags: [actor, lifecycle, state]
parents:
  - [[mech_actor_pattern]]
dependents: []
---

# Actor Lifecycle Management

## INTENT
Manage the transition of an Actor from creation to safe termination, ensuring all pending work is cleared and resources are released.

## THE RULE / LOGIC
- **Startup**: `Start()` must initialize the underlying `MessageQueue` and spawn the dispatcher. If `NotifyStart` is set, an `ActorStarted` notification is injected.
- **Graceful Shutdown**:
  - `PrepareToStop()`: Prevents the arrival of new messages but allows the queue to drain.
  - **Draining**: The Actor remains "Alive" until the queue reports empty via the `doneChan`.
- **Hard Shutdown**: `Stop()` causes the actor to immediately terminate the dispatch loop regardless of pending messages.
- **Signal Interaction**: Special `ActorStop` messages should trigger a deferred call to `Stop()`.

## TECHNICAL INTERFACE (The Bridge)
- **Code Tag**: `@spec-link [[mech_actor_lifecycle]]`
- **Primary Methods**:
  - `Start()`
  - `Stop()`
  - `PrepareToStop()`
  - `ActorStop` (Signal type)

## EXPECTATION (For Testing)
- **Persistence**: Messages sent before `PrepareToStop` but after it is called are STILL processed.
- **Finality**: Once the `doneChan` from `PrepareToStop` closes, no further code in the actor's handlers should be executing.
