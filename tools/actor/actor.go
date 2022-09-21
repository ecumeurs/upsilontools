package actor

import (
	"fmt"
	"reflect"

	"github.com/ecumeurs/upsilontools/tools/messagequeue"
	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
	"github.com/sirupsen/logrus"
)

// That's to embed in a method struct. It's useless but it's documentary.
type NoReply struct{}

// Actor communication interface
// Used by other to initiate communication with the actor.
type Communication interface {
	NotifyActor(msg message.Message)
	SendActor(msg message.Message, callback chan message.Message)
}

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
	stopper               chan bool
	logger                *logrus.Entry
}

// New
func New(name string) *Actor {
	act := &Actor{
		actorName:    name,
		queue:        *messagequeue.New(name),
		CallbackChan: make(chan message.Message),
		stopper:      make(chan bool),
	}

	act.logger = logrus.WithFields(logrus.Fields{
		"component": "actor",
		"name":      act.actorName,
	})
	return act
}

func (a Actor) Name() string {
	return a.actorName
}

func (a *Actor) Stop() {
	a.queue.Stop()
}

// PrepareToStop will disconnect the actor from the message queue and return a channel that will tell when all pending messages have been processed.
func (a *Actor) PrepareToStop() chan bool {
	return a.queue.PrepareStop()
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
				a.logger.WithFields(logrus.Fields{
					"message":      msg.String(),
					"message_type": reflect.TypeOf(msg.TargetMethod).String()}).Warn("Unhandled message")
				a.Reply(msg.Reply())
			}
		}
		a.Stop()
	default:
		if handler != nil {
			if !handler(msg) {
				// message hasn't been handled.
				// auto handle it.
				a.logger.WithFields(logrus.Fields{
					"message":      msg.String(),
					"message_type": reflect.TypeOf(msg.TargetMethod).String()}).Warn("Unhandled message")

				a.Reply(msg.Reply())
			}
		} else {
			a.logger.WithFields(logrus.Fields{
				"message":      msg.String(),
				"message_type": reflect.TypeOf(msg.TargetMethod).String()}).Debug("ReceiveMessage is nil")
		}
	}
}

// Start
func (a *Actor) Start() {
	a.queue.Start()
	go func() {
		a.logger.Info("Actor started")
		done := false
		for !done {
			select {
			case msg := <-a.queue.GetActorChan():
				a.logger.WithFields(logrus.Fields{
					"message":      msg.String(),
					"message_type": reflect.TypeOf(msg.TargetMethod).String()}).Debug("About to process message")
				a.processMessage(msg, a.receiveMessageHandler)
			case msg := <-a.CallbackChan:
				a.logger.WithFields(logrus.Fields{
					"message":      msg.String(),
					"message_type": reflect.TypeOf(msg.TargetMethod).String()}).Debug("About to process reply")
				a.processMessage(msg, a.replyMessageHandler)
			case <-a.stopper:
				done = true
			}
		}
		a.logger.Info("Actor Stopped")
	}()
	a.Notify(message.Create(nil, ActorStarted{}, nil))
}

// String
func (a *Actor) String() string {
	return fmt.Sprintf("[A %s]", a.actorName)
}

func (a *Actor) GetQueue() *messagequeue.MessageQueue {
	return &a.queue
}

// SendMessage
func (a *Actor) Send(msg message.Message, res chan message.Message) {
	a.queue.Send(msg, res)
}

func (a *Actor) Notify(msg message.Message) {
	a.queue.Send(msg, nil)
}

func (a *Actor) SetReceiveMessageHandler(handler func(msg message.Message) bool) {
	a.receiveMessageHandler = handler
}

func (a *Actor) SetReplyMessageHandler(handler func(msg message.Message) bool) {
	a.replyMessageHandler = handler
}
