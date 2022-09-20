package actor

import (
	"fmt"
	"reflect"

	"github.com/ecumeurs/upsilontools/tools/messagequeue"
	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
)

// That's to embed in a method struct. It's useless but it's documentary.
type NoReply struct{}

// Default methods
type ActorStarted struct {
	NoReply
}
type ActorAboutToStop struct {
	NoReply
}

type ActorStop struct{}
type ActorError struct {
	Err error
}

type Actor struct {
	actorName             string
	queue                 messagequeue.MessageQueue
	receiveMessageHandler func(msg message.Message) bool // return false when message hasn't been handled.
	replyMessageHandler   func(msg message.Message) bool // return false when message hasn't been handled
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

func (a *Actor) processMessage(msg message.Message, handler func(msg message.Message) bool) {
	switch msg.TargetMethod.(type) {
	case ActorStop:
		msg.TargetMethod = ActorAboutToStop{}
		if handler != nil {
			if !handler(msg) {
				// message hasn't been handled.
				// auto handle it.
				fmt.Println("Actor[", a.actorName, "]: message not handled", msg.String(), reflect.TypeOf(msg.TargetMethod))
				a.Reply(msg.Reply())
			}
		}
		a.Stop()
	default:
		if handler != nil {
			if !handler(msg) {
				// message hasn't been handled.
				// auto handle it.
				fmt.Println("Actor[", a.actorName, "]: message not handled", msg.String(), reflect.TypeOf(msg.TargetMethod))
				a.Reply(msg.Reply())
			}
		} else {
			fmt.Println("Actor: ReceiveMessage is nil")
		}
	}
}

// Start
func (a *Actor) Start() {
	a.queue.Start()
	go func() {
		fmt.Println("Actor[", a.actorName, "]: Started")
		for {
			select {
			case msg := <-a.queue.GetActorChan():
				fmt.Println("Actor[", a.actorName, "]: Received message", msg.String(), reflect.TypeOf(msg.TargetMethod))
				a.processMessage(msg, a.receiveMessageHandler)
			case msg := <-a.CallbackChan:
				fmt.Println("Actor[", a.actorName, "]: Reply message", msg.String(), reflect.TypeOf(msg.TargetMethod))
				a.processMessage(msg, a.replyMessageHandler)
			}
		}
	}()
	a.SendMessageAndForget(message.Create(nil, ActorStarted{}, nil))
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

func (a *Actor) SetReceiveMessageHandler(handler func(msg message.Message) bool) {
	a.receiveMessageHandler = handler
}

func (a *Actor) SetReplyMessageHandler(handler func(msg message.Message) bool) {
	a.replyMessageHandler = handler
}
