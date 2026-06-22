---
id: mechanic_message_queue
human_name: Message Queue Infrastructure
type: MECHANIC
layer: IMPLEMENTATION
version: 1.0
status: STABLE
priority: 3
tags: [concurrency, message-queue, actor]
parents:
  - [[mech_actor_pattern]]
dependents: []
---

# Message Queue Infrastructure

## INTENT
Provide the core channel-based data structures and the thread-safe sequential processing loop for the actor system's message queue, ensuring messages are processed strictly in order with graceful shutdown.

## THE RULE / LOGIC
**Structures:**
- **Message:** encapsulates target, method, content, and an optional reply channel.
- **Queue:** manages `inputChan` (incoming messages) and `executorChan` (processing), with an internal non-blocking buffer slice so senders are never blocked.
- **Mutex protection:** `sync.Mutex` guards the internal message slice and state flags across goroutines.

**Sequential Processing:**
- **FIFO Ordering:** messages are dispatched in the exact order received via `inputChan`.
- **One In-Flight:** only one message is dispatched to the executor at a time; the next is held until the previous message's ACK arrives on `executorReplyChan`.
- **Resilience:** an ACK received while the buffer is empty (phantom ACK) must be ignored/logged, never panic.

**Graceful Shutdown:**
- `PrepareStop` flags the queue to reject new inputs and signals completion via `doneChan` once the buffer is drained.

## TECHNICAL INTERFACE
- **Code Tag:** `@spec-link [[mechanic_message_queue]]`
- **Test Names:** `TestMessageQueue`

## EXPECTATION
- Messages are processed in FIFO order, exactly one at a time, each gated on the prior ACK.
- The struct initializes correctly with input and executor channels; senders never block.
- `PrepareStop` rejects new messages and closes `doneChan` once the existing queue drains.
- Unexpected/phantom ACKs never cause a panic.
