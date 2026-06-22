---
id: mech_actor_handler_context
human_name: "Actor Handler Context"
type: MECHANIC
layer: IMPLEMENTATION
version: 1.0
status: DRAFT
priority: 5
tags: [actor, protocol, context]
parents:
  - [[mech_actor_pattern]]
dependents: []
---

# Actor Handler Context

## INTENT
Provide a gated interface for message handlers to ensure strict adherence to the request-reply protocol and prevent queue deadlocks.

## THE RULE / LOGIC
- **Context Separation**:
  - `NotificationContext`: Only exposes the message. Has no reply methods.
  - `CallContext`: Exposes `Reply`, `NoReply`, and `DeferReply` to satisfy the mandatory response requirement of a `Call`.
- **Reply Invariant**: Every `CallContext` must eventually trigger exactly one completion signal (`Reply`, `NoReply`, or `DeferReply`).
- **Deferred Execution**: `DeferReply` allows the handler to exit without completing the protocol immediately, delegating the responsibility of replying to a future event (e.g., a callback).

## TECHNICAL INTERFACE (The Bridge)
- **Code Tag**: `@spec-link [[mech_actor_handler_context]]`
- **Key Structs**: `CallContext`, `NotificationContext`, `ReplyContext`.
- **Key Methods**:
  - `CallContext.Reply(msg)`: Sends response and marks message as replied.
  - `CallContext.NoReply()`: Acknowledges call without payload.
  - `CallContext.DeferReply()`: Opts out of automatic reply-check on handler exit.

## EXPECTATION (For Testing)
- **Protocol Safety**: Handlers using `NotificationContext` cannot accidentally respond to the caller.
- **Deadlock Prevention**: Handlers using `CallContext` are checked for "forgotten" replies at the dispatcher level.
