---
id: mech_actor_pattern
human_name: "Actor Architecture Pattern"
type: MODULE
layer: ARCHITECTURE
version: 2.0
status: STABLE
priority: 5
tags: [actor, encapsulation, messaging, concurrency]
parents:
  - [[upsilonbattle:module_actor_concurrency]]
dependents:
  - [[mech_actor_dispatch_loop]]
  - [[mech_actor_handler_context]]
  - [[mech_actor_lifecycle]]
  - [[mechanic_message_queue]]
---

# Actor Architecture Pattern

## INTENT
Encapsulate internal state and behavioral logic behind a message-driven communication interface to prevent race conditions and simplify concurrency in the Upsilon Engine.

## THE RULE / LOGIC
- **Atomic Operations**: All logic within an Actor is strictly sequential, driven by a FIFO `MessageQueue`.
- **Communication Modes**:
  1. **Notification (Async)**: Fire-and-forget message between actors.
  2. **Call (Sync/Request-Reply)**: Blocking or non-blocking request expecting a correlated response.
  3. **Reply (Callback)**: Handling of a response from a previous `Call` via `AddReplyHandler`.
  4. **Self-Notification**: Safe internal work scheduling that maintains FIFO ordering with external messages.
- **Isolation Sovereignty**: No Actor may directly access the internal state of another; all data flow must occur via `Communication` interfaces.

## TECHNICAL INTERFACE (The Bridge)
- **Code Tag**: `@spec-link [[mech_actor_pattern]]`
- **Primary Interfaces**:
  - `Communication`: `NotifyActor` (Async), `SendActor` (Request/Reply).
  - `Manageable`: `Start`, `Stop`, `PrepareToStop`.
- **New Patterns**:
  - `SelfNotify`: Loop-back notification to own queue.
  - `SelfNotifyDelayed`: Scheduled loop-back with configurable delay.

## EXPECTATION (For Testing)
- **Concurrency Safety**: The system survives a high volume of interleaved messages without deadlocks or state corruption.
- **Protocol Integrity**: No `Call` remains unreplied, and no `Notification` blocks the queue indefinitely.
