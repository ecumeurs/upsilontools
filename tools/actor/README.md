# .\tools\actor

[Up](../README.md)

# Actor

An actor is a stand alone thread that can only be accessed by using the corresponding Message Queue. 

The actor will expose it's available actions through it and will only respond through it as well.

This ensure data integrity and thread safety.

# Pro & Con & Tips

* Actors protect data integrity and thread safety.
* Go dislike cyclic dependencies, so you should avoid them.
  * Make sure that the struct used to discriminate your TargetMethod is declared outside of the actor's package.
* If you need to access to another Actor you should use the actor.Communication interface.


# Usage

Declare a new struct and compound the Actor struct and set the receiveMessageHandler and replyMessageHandler appropriately.

Implement the actor.Communication interface: allows other entities (Actor or not) to communicate with the actor. This also allow to break cyclic dependencies.



```go
type MyActor struct {
    act actor.Actor
}

type MyMethod struct {}

func New() *MyActor{
    a := &MyActor{}
    a.act = actor.New("MyActor") // sets a name for logging purposes
    a.SetReceiveMessageHandler(a.ReceiveMessage)
    a.SetReplyMessageHandler(a.MessageReplied)
    a.Start() // dont forget to start! 
    return a
}


//implement actor.Communication
func (a *MyActor) NotifyActor(msg message.Message) {
    a.act.Notify(msg)
}

func (a *Actor) SendActor(msg message.Message, callback chan message.Message) {
    a.act.Send(msg, callback)
}



func (a *MyMethod) ReceiveMessage(msg message.Message) bool {
    select msg.TargetMethod.(type) {
        case MyMethod:
            a.MyMethod(msg)
            return true // this message has been handled (error or not)
    }
    return false // this will ensure that unhandled messages are correctly defered.
}

func (a *MyMethod) MessageReplied(msg message.Message) bool{
    select msg.TargetMethod.(type) {
    }
    return false
}

func (a *MyMethod) MyMethod(msg message.Message) {
    // Do something
    a.Reply(msg.Reply()) 
    // always finish with either a.Reply() or a.NoReply() 
    // either are needed to free the MessageQueue loop.
    // don't forget to call on msg.Reply() to get the reply message, eventually fill it with further information in Content.
}
```

Then you can create a new instance of your actor and send it a message.

```go
act := MyActor.New()
req := message.Create(nil /*Content*/, MyActor.MyMethod /*TargetMethod*/, nil /*Callback Struct*/)

resChan := make(chan message.Message)
defer close(resChan)
act.SendActor(req, resChan)

<-resChan
```

Also you may want to keep your method structures in a separate file to avoid cyclic dependencies and only forward the Actor's queue to foreign actors and elements.
