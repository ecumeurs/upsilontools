# .\tools\messagequeue

[Up](../README.md)

# Message Queue 

Message queue module aim to provide a simple and easy to use message queue for the application. It is based on the [Message Queue](https://en.wikipedia.org/wiki/Message_queue) concept.

## Design

The message queue will have one input channel that is expected to be shared among multiple potential caller.

An internal thread will manage the queue itself and will be responsible to call the registered callback for each message.

The message queue will be able to handle multiple message type. Each message type will have its own callback.

The message queue will have a consummer who will be responsible to process the messages from the queue. The consummer will be a thread that will be started when the message queue is started.
The consummer will have access to another channel the queue will handle to send back the result of the processing.
Only when the result is sent back, the message will be removed from the queue, and replied to, of course. But also the next message will be processed.

## Usage


