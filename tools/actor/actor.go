package actor

import (
	"fmt"

	"github.com/ecumeurs/upsilontools/tools/messagequeue"
	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
)

type ActorStop struct{}
type ActorError struct {
	Err error
}

// That's to embed in a method struct. It's useless but it's documentary.
type NoReply struct{}

type Actor struct {
	actorName             string
	queue                 messagequeue.MessageQueue
	receiveMessageHandler func(msg message.Message)
	replyMessageHandler   func(msg message.Message)
	CallbackChan          chan message.Message
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

func (a *Actor) NoReply(msg message.Message) {
	a.queue.GetActorChan() <- msg
}

func (a *Actor) processMessage(msg message.Message, handler func(msg message.Message)) {
	switch msg.TargetMethod.(type) {
	case ActorStop:
		a.Stop()
	default:
		if a.receiveMessageHandler != nil {
			handler(msg)
		} else {
			fmt.Println("Actor: ReceiveMessage is nil")
		}
	}
}

// Start
func (a *Actor) Start() {
	a.queue.Start()
	go func() {
		for {
			select {
			case msg := <-a.queue.GetActorChan():
				a.processMessage(msg, a.receiveMessageHandler)
			case msg := <-a.CallbackChan:
				a.processMessage(msg, a.replyMessageHandler)
			}
		}
	}()
}

// String
func (a *Actor) String() string {
	return fmt.Sprintf("[A %s]", a.actorName)
}

func (a *Actor) GetQueue() *messagequeue.MessageQueue {
	return &a.queue
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

func (a *Actor) SetReplyMessageHandler(handler func(msg message.Message)) {
	a.replyMessageHandler = handler
}
