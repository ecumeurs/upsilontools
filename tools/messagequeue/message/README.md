# .\tools\messagequeue\message

[Up](../README.md)

# Message

Message module aim to provide a simple and easy to use message for message queue. It allow to identify the message with a unique identifier, store a payload and a callback to be called when the message is processed.

## Target & Callback Methods

These interface{} expect to be replaced by the user with custom structs.
These structs will be used to identify which action the actor should use. Expect the actor to provide available methods.
