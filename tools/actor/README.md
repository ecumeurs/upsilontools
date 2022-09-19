# .\tools\actor

[Up](../README.md)

# Actor

An actor is a stand alone thread that can only be accessed by using the corresponding Message Queue. 

The actor will expose it's available actions through it and will only respond through it as well.

This ensure data integrity and thread safety.

# Usage

Declare a new struct embedding the Actor struct and set the ReceiveMessageHandler appropriately

```go
type MyActor struct {
    actor.Actor
}

type MyMethod struct {}

func (a *MyActor) Init() {
    a.Start()
    a.ReceiveMessageHandler =  a.ReceiveMessage
}

func (a *MyMethod) ReceiveMessage(msg message.Message) {
    select msg.TargetMethod.(type) {
        case MyMethod:
            a.MyMethod(msg)
    }
}

func (a *MyMethod) MyMethod(msg message.Message) {
    // Do something
    a.Reply(msg.Reply())
}
```

Then you can create a new instance of your actor and send it a message.

```go
act := MyActor{}
act.Init()
req := message.New()
req.TargetMethod = MyMethod{}

resChan := make(chan message.Message)
defer close(resChan)
act.SendMessage(req, resChan)

<-resChan
```

Note you may also add the actor as a compound to the struct instead of embedding it (hides the Actor struct from the outside)

Also you may want to keep your method structures in a separate file to avoid cyclic dependencies and only forward the Actor's queue to foreign actors and elements.
