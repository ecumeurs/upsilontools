package actor

import (
	"fmt"

	"github.com/ecumeurs/upsilontools/tools/messagequeue"
	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
)

type ActorStop struct{}

type Actor struct {
	ActorName             string
	Queue                 messagequeue.MessageQueue
	ReceiveMessageHandler func(msg message.Message)
}

// New
func New(name string) *Actor {
	return &Actor{
		ActorName: name,
		Queue:     *messagequeue.New(name),
	}
}

func (a *Actor) Stop() {
	a.Queue.Stop()
}

// Reply
func (a *Actor) Reply(msg message.Message) {
	a.Queue.GetActorChan() <- msg
}

// Start
func (a *Actor) Start() {
	a.Queue.Start()
	go func() {
		for {
			msg := <-a.Queue.GetActorChan()
			switch msg.TargetMethod.(type) {
			case ActorStop:
				a.Stop()
				return
			default:
				if a.ReceiveMessageHandler != nil {
					a.ReceiveMessageHandler(msg)
				} else {
					fmt.Println("Actor: ReceiveMessage is nil")
				}
			}
		}
	}()
}

// String
func (a *Actor) String() string {
	return fmt.Sprintf("[A %s]", a.ActorName)
}

// SendMessage
func (a *Actor) SendMessage(msg message.Message, res chan message.Message) {
	a.Queue.Send(msg, res)
}
