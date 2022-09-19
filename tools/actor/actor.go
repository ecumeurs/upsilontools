package actor

import (
	"fmt"

	"github.com/ecumeurs/upsilontools/tools/messagequeue"
	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
)

type ActorStop struct{}

// That's to embed in a method struct. It's useless but it's documentary.
type NoReply struct{}

type Actor struct {
	actorName             string
	queue                 messagequeue.MessageQueue
	receiveMessageHandler func(msg message.Message)
}

// New
func New(name string) *Actor {
	return &Actor{
		actorName: name,
		queue:     *messagequeue.New(name),
	}
}

func (a *Actor) Stop() {
	a.queue.Stop()
}

// Reply
func (a *Actor) Reply(msg message.Message) {
	a.queue.GetActorChan() <- msg
}

// Start
func (a *Actor) Start() {
	a.queue.Start()
	go func() {
		for {
			msg := <-a.queue.GetActorChan()
			switch msg.TargetMethod.(type) {
			case ActorStop:
				a.Stop()
				return
			default:
				if a.receiveMessageHandler != nil {
					a.receiveMessageHandler(msg)
				} else {
					fmt.Println("Actor: ReceiveMessage is nil")
				}
			}
		}
	}()
}

// String
func (a *Actor) String() string {
	return fmt.Sprintf("[A %s]", a.actorName)
}

// SendMessage
func (a *Actor) SendMessage(msg message.Message, res chan message.Message) {
	a.queue.Send(msg, res)
}

func (a *Actor) SendMessageAndForget(msg message.Message) {
	a.queue.Send(msg, nil)
}

func (a *Actor) SetReceiveMessageHandler(handler func(msg message.Message)) {
	a.receiveMessageHandler = handler
}
