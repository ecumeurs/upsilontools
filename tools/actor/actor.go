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
	NotifyActor(msg *message.Message)
	SendActor(msg *message.Message, callback chan *message.Message)
}

// Expect actors to be manageable.
type Manageable interface {
	Communication
	Start()
	Stop()
	PrepareToStop() chan bool
}

// Actor's method template
type ActorMethod interface {
	MethodType() reflect.Type
	Handler(msg *message.Message) (handled bool)
	Validator(msg *message.Message) (errors []error)
}

type actorMethodImpl struct {
	methodType reflect.Type
	handler    func(msg *message.Message) (handled bool)
	validator  func(msg *message.Message) (errors []error)
}

func (a actorMethodImpl) MethodType() reflect.Type {
	return a.methodType
}

func (a actorMethodImpl) Handler(msg *message.Message) (handled bool) {
	if a.handler == nil {
		return false
	}
	return a.handler(msg)
}

func (a actorMethodImpl) Validator(msg *message.Message) (errors []error) {
	if a.validator == nil {
		return nil
	}
	return a.validator(msg)
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
	actorName     string
	queue         messagequeue.MessageQueue
	CallbackChan  chan *message.Message
	stopper       chan bool
	Logger        *logrus.Entry
	RequestLogger *logrus.Entry
	methods       map[reflect.Type]ActorMethod
	replies       map[reflect.Type]ActorMethod

	// NOTE: dedicated triggers, should be mostly unused or to provide more information in logs.
	NewMessageReceived func(act *Actor, msg *message.Message)
	NewReplyReceived   func(act *Actor, msg *message.Message)
}

// New
func New(name string) *Actor {
	act := &Actor{
		actorName:    name,
		queue:        *messagequeue.New(name),
		CallbackChan: make(chan *message.Message),
		stopper:      make(chan bool),
		methods:      make(map[reflect.Type]ActorMethod),
		replies:      make(map[reflect.Type]ActorMethod),
	}

	act.Logger = logrus.WithFields(logrus.Fields{
		"component": "actor",
		"name":      act.actorName,
	})
	return act
}

// Add Method to the actor
func (a *Actor) AddMethod(method interface{}, handler func(msg *message.Message) (handled bool), validator func(msg *message.Message) (errors []error)) {
	a.Logger.WithFields(logrus.Fields{
		"method": method,
	}).Debug("Adding method")
	t := reflect.TypeOf(method)
	if _, ok := a.methods[t]; ok {
		a.Logger.WithFields(logrus.Fields{
			"method": method,
		}).Warn("Method already registered")
		return
	}
	a.methods[t] = actorMethodImpl{
		methodType: t,
		handler:    handler,
		validator:  validator,
	}
}

// Add Method to the actor
func (a *Actor) AddActorMethod(mt ActorMethod) {
	a.Logger.WithFields(logrus.Fields{
		"method": mt.MethodType(),
	}).Debug("Adding method")
	if _, ok := a.methods[mt.MethodType()]; ok {
		a.Logger.WithFields(logrus.Fields{
			"method": mt.MethodType(),
		}).Warn("Method already registered")
		return
	}
	a.methods[mt.MethodType()] = mt
}

// Add Reply handler to the actor
func (a *Actor) AddReply(method interface{}, handler func(msg *message.Message) (handled bool), validator func(msg *message.Message) (errors []error)) {
	a.Logger.WithFields(logrus.Fields{
		"reply": method,
	}).Debug("Adding Reply")
	t := reflect.TypeOf(method)
	if _, ok := a.methods[t]; ok {
		a.Logger.WithFields(logrus.Fields{
			"reply": method,
		}).Warn("Reply already registered")
		return
	}
	a.replies[t] = actorMethodImpl{
		methodType: t,
		handler:    handler,
		validator:  validator,
	}
}

// Add Reply handler to the actor
func (a *Actor) AddActorReply(mt ActorMethod) {
	a.Logger.WithFields(logrus.Fields{
		"reply": mt.MethodType(),
	}).Debug("Adding reply")
	if _, ok := a.methods[mt.MethodType()]; ok {
		a.Logger.WithFields(logrus.Fields{
			"reply": mt.MethodType(),
		}).Warn("Reply already registered")
		return
	}
	a.replies[mt.MethodType()] = mt
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
func (a *Actor) Reply(origin *message.Message, msg *message.Message) {
	if origin == nil {
		a.Logger.Warn("Reply called with nil origin -- Totally unexpected")
		return
	} else {
		if !origin.ShouldBeRepliedTo {
			a.Logger.WithFields(logrus.Fields{
				"message":      origin.String(),
				"message_type": reflect.TypeOf(origin.TargetMethod).String()}).Warn("Reply called on a message that should not be replied to")
			return
		}
	}
	origin.HasBeenReplied = true
	a.queue.GetExecutorReplyChan() <- msg
}

func (a *Actor) NoReply(origin *message.Message) {
	if origin == nil {
		a.Logger.Warn("Reply called with nil origin -- Totally unexpected")
		return
	}
	if !origin.ShouldBeRepliedTo {
		a.Logger.WithFields(logrus.Fields{
			"message":      origin.String(),
			"message_type": reflect.TypeOf(origin.TargetMethod).String()}).Warn("Reply called on a message that should not be replied to")
	} else {
		origin.HasBeenReplied = true
		a.queue.GetExecutorReplyChan() <- origin
	}
}

func (a *Actor) processReply(msg *message.Message) {
	a.RequestLogger = a.Logger.WithFields(logrus.Fields{
		"request_type": "reply",
		"message":      msg.String(),
		"message_type": reflect.TypeOf(msg.TargetMethod).String(),
	})
	if a.NewReplyReceived != nil {
		a.NewReplyReceived(a, msg)
	}

	typ := reflect.TypeOf(msg.TargetMethod)
	// special case ActorStop
	if typ == reflect.TypeOf(ActorStop{}) {
		a.RequestLogger.Debug("ActorStop received")
		msg.TargetMethod = ActorAboutToStop{}
		typ = reflect.TypeOf(msg.TargetMethod)
		defer a.Stop()
	}

	if msg.HasError {
		a.RequestLogger.WithFields(logrus.Fields{
			"error": msg.ErrorMessage,
		}).Error("Error received")
	}

	v, found := a.replies[typ]
	if !found {
		a.RequestLogger.Warn("Unexpected reply")
		return
	}

	if v.Validator(msg) != nil {
		a.RequestLogger.Warn("Reply validation failed")
		rpl := msg.ReplyWithError("Reply validation failed", "actor.reply.validation")
		a.Reply(msg, rpl)
		return
	}

	if v.Handler(msg) {
		return
	} else {
		a.RequestLogger.Warn("Unhandled reply")
		// auto reply ... bad ?
		a.Reply(msg, msg.Reply())
		return
	}
}

func (a *Actor) processMessage(msg *message.Message) {
	a.RequestLogger = a.Logger.WithFields(logrus.Fields{
		"request_type": "message",
		"message":      msg.String(),
		"message_type": reflect.TypeOf(msg.TargetMethod).String(),
	})

	if a.NewMessageReceived != nil {
		a.NewMessageReceived(a, msg)
	}
	typ := reflect.TypeOf(msg.TargetMethod)
	// special case ActorStop
	if typ == reflect.TypeOf(ActorStop{}) {
		a.RequestLogger.Debug("ActorStop received")
		msg.TargetMethod = ActorAboutToStop{}
		typ = reflect.TypeOf(msg.TargetMethod)
		defer a.Stop()
	}

	if msg.HasError {
		a.RequestLogger.WithFields(logrus.Fields{
			"error": msg.ErrorMessage,
		}).Error("Error received")
	}

	v, found := a.methods[typ]
	if !found {
		a.RequestLogger.Warn("Unexpected message")
		// auto reply ... bad ?
		a.Reply(msg, msg.Reply())
		return
	}

	if v.Validator(msg) != nil {
		a.RequestLogger.Warn("Reply validation failed")
		rpl := msg.ReplyWithError("Reply validation failed", "actor.reply.validation")
		a.Reply(msg, rpl)
		return
	}

	if v.Handler(msg) {
		if msg.ShouldBeRepliedTo {
			// Add a delayed reply option ( message should be replied to, but hasn't yet. all while allowing other messages to be processed.)
			if !msg.HasBeenReplied {
				a.RequestLogger.Error("Message should be replied to but has not been")
				rpl := msg.ReplyWithError("Message should be replied to but has not been", "actor.reply.missing")
				a.Reply(msg, rpl)
			}
		}

		return
	} else {
		a.RequestLogger.Warn("Unhandled message")
		// auto reply ... bad ?
		a.Reply(msg, msg.Reply())
		return
	}
}

// Start
func (a *Actor) Start() {
	a.queue.Start()
	go func() {
		a.Logger.Info("Actor started")
		done := false
		for !done {
			select {
			case msg := <-a.queue.GetExecutorChan():
				a.Logger.WithFields(logrus.Fields{
					"message":      msg.String(),
					"message_type": reflect.TypeOf(msg.TargetMethod).String()}).Debug("About to process message")
				a.processMessage(msg)
			case msg := <-a.CallbackChan:
				a.Logger.WithFields(logrus.Fields{
					"message":      msg.String(),
					"message_type": reflect.TypeOf(msg.TargetMethod).String()}).Debug("About to process reply")
				a.processReply(msg)
			case <-a.stopper:
				done = true
			}
		}
		a.Logger.Info("Actor Stopped")
	}()
	a.NotifyActor(message.Create(nil, ActorStarted{}, nil))
}

// String
func (a *Actor) String() string {
	return fmt.Sprintf("[A %s]", a.actorName)
}

func (a *Actor) GetQueue() *messagequeue.MessageQueue {
	return &a.queue
}

func (a *Actor) SendActor(msg *message.Message, res chan *message.Message) {
	a.Logger.WithFields(logrus.Fields{
		"message":      msg.String(),
		"message_type": reflect.TypeOf(msg.TargetMethod).String()}).Debug("Sending message")
	a.queue.Send(msg, res)
}

func (a *Actor) NotifyActor(msg *message.Message) {
	a.Logger.WithFields(logrus.Fields{
		"message":      msg.String(),
		"message_type": reflect.TypeOf(msg.TargetMethod).String()}).Debug("Notifying message")
	msg.ShouldBeRepliedTo = false
	a.queue.Send(msg, nil)
}

// GetCallbackChan
func (a *Actor) GetCallbackChan() chan *message.Message {
	return a.CallbackChan
}
