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

// --- NEW STRUCTURES (ISS-004) ---

// NotificationContext wraps a message for fire-and-forget notification handlers.
// It explicitly lacks Reply() or NoReply() to prevent protocol violations.
type NotificationContext struct {
	// Msg is the original incoming message.
	// Constraint: Must not be mutated by the handler.
	Msg *message.Message
}

// CallContext wraps a message for request-reply handlers.
// It provides the necessary methods to complete the synchronized call safely.
type CallContext struct {
	// Msg is the original incoming message.
	Msg   *message.Message
	actor *Actor
}

// Reply sends a response back to the caller.
// Goal: Completes the Call lifecycle.
// Unknowns: Verify if Msg.Reply() generation is needed here or by the caller.
func (c *CallContext) Reply(replyMsg *message.Message) {
	if !c.Msg.ShouldBeRepliedTo {
		c.actor.Logger.WithFields(logrus.Fields{
			"message":      c.Msg.String(),
			"message_type": reflect.TypeOf(c.Msg.TargetMethod).String()}).Warn("Reply called on a message that should not be replied to")
		return
	}
	c.Msg.HasBeenReplied = true
	if c.Msg.ReplyChan != nil {
		c.Msg.ReplyChan <- replyMsg
	}
}

// NoReply acknowledges the call but sends no payload.
// Goal: Completes the Call lifecycle without returning data.
func (c *CallContext) NoReply() {
	if !c.Msg.ShouldBeRepliedTo {
		c.actor.Logger.WithFields(logrus.Fields{
			"message":      c.Msg.String(),
			"message_type": reflect.TypeOf(c.Msg.TargetMethod).String()}).Warn("Reply called on a message that should not be replied to")
	} else {
		c.Msg.HasBeenReplied = true
		if c.Msg.ReplyChan != nil {
			c.Msg.ReplyChan <- c.Msg
		}
	}
}

// DeferReply marks the message as intentionally delayed.
// Goal: Allows the handler to exit without triggering an "unhandled call" crash,
// so the actor can reply later (e.g., after receiving a callback from another actor).
// Context Compilation: Steps: 1. Mark Msg.HasBeenReplied = true. 2. Store ctx in Actor state.
func (c *CallContext) DeferReply() {
	c.Msg.HasBeenReplied = true
}

// ReplyContext wraps a message received as a reply.
// It lacks Reply methods.
type ReplyContext struct {
	// Msg is the replied message from another actor.
	Msg *message.Message
}

// --- END NEW STRUCTURES ---

type notificationHandlerImpl struct {
	handler   func(ctx NotificationContext)
	validator func(msg *message.Message) []error
}

type callHandlerImpl struct {
	handler   func(ctx CallContext)
	validator func(msg *message.Message) []error
}

type replyHandlerImpl struct {
	handler   func(ctx ReplyContext)
	validator func(msg *message.Message) []error
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

	// New structural maps
	notificationHandlers map[reflect.Type]notificationHandlerImpl
	callHandlers         map[reflect.Type]callHandlerImpl
	replyHandlers        map[reflect.Type]replyHandlerImpl

	// CrashOnUnhandled dictates behavior when a message finds no handler.
	// Constraints: Default is true. Unhandled Calls always crash/error (cannot skip).
	CrashOnUnhandled bool

	NotifyStart bool

	// NOTE: dedicated triggers, should be mostly unused or to provide more information in logs.
	NewMessageReceived func(act *Actor, msg *message.Message)
	NewReplyReceived   func(act *Actor, msg *message.Message)
}

// New
func New(name string) *Actor {
	act := &Actor{
		actorName:            name,
		queue:                *messagequeue.New(name),
		CallbackChan:         make(chan *message.Message),
		stopper:              make(chan bool),
		methods:              make(map[reflect.Type]ActorMethod),
		replies:              make(map[reflect.Type]ActorMethod),
		notificationHandlers: make(map[reflect.Type]notificationHandlerImpl),
		callHandlers:         make(map[reflect.Type]callHandlerImpl),
		replyHandlers:        make(map[reflect.Type]replyHandlerImpl),
		CrashOnUnhandled:     true,
		NotifyStart:          false,
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

// AddNotificationHandler registers a fire-and-forget handler.
// Goal: Register a method type to a specific handler function.
// Context Compilation:
// 1. Verify type doesn't already exist.
// 2. Add to internal notification map.
func (a *Actor) AddNotificationHandler(method interface{}, handler func(ctx NotificationContext), validator func(msg *message.Message) []error) {
	t := reflect.TypeOf(method)
	if _, ok := a.notificationHandlers[t]; ok {
		a.Logger.WithFields(logrus.Fields{"method": method}).Warn("Notification Handler already registered")
		return
	}
	a.notificationHandlers[t] = notificationHandlerImpl{handler: handler, validator: validator}
}

// AddCallHandler registers a synchronized request-reply handler.
func (a *Actor) AddCallHandler(method interface{}, handler func(ctx CallContext), validator func(msg *message.Message) []error) {
	t := reflect.TypeOf(method)
	if _, ok := a.callHandlers[t]; ok {
		a.Logger.WithFields(logrus.Fields{"method": method}).Warn("Call Handler already registered")
		return
	}
	a.callHandlers[t] = callHandlerImpl{handler: handler, validator: validator}
}

// AddReplyHandler registers a handler for an incoming reply.
func (a *Actor) AddReplyHandler(method interface{}, handler func(ctx ReplyContext), validator func(msg *message.Message) []error) {
	t := reflect.TypeOf(method)
	if _, ok := a.replyHandlers[t]; ok {
		a.Logger.WithFields(logrus.Fields{"reply": method}).Warn("Reply Handler already registered")
		return
	}
	a.replyHandlers[t] = replyHandlerImpl{handler: handler, validator: validator}
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

// Legacy Reply / NoReply deleted per ISS-004 to enforce Context usage.
// They are now owned by CallContext.

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

	// Try new typed reply handler first
	if rh, found := a.replyHandlers[typ]; found {
		if rh.validator != nil && len(rh.validator(msg)) > 0 {
			a.RequestLogger.Warn("Reply validation failed")
			// Cannot "reply" to a reply. Just drop.
			return
		}
		replyCtx := ReplyContext{Msg: msg}
		rh.handler(replyCtx)
		return
	}

	v, found := a.replies[typ]
	if !found {
		a.RequestLogger.Warn("Unexpected reply")
		if a.CrashOnUnhandled {
			panic(fmt.Sprintf("Unhandled reply msg type %v", typ))
		}
		return
	}

	if v.Validator(msg) != nil {
		a.RequestLogger.Warn("Reply validation failed")
		return // we can't reply to a reply anyway
	}

	if v.Handler(msg) {
		return
	} else {
		a.RequestLogger.Warn("Unhandled reply")
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

	// Try Call handler
	if ch, found := a.callHandlers[typ]; found {
		if !msg.ShouldBeRepliedTo {
			a.RequestLogger.Error("Message arrived as notification but is mapped as a call handler")
			if a.CrashOnUnhandled {
				panic(fmt.Sprintf("Protocol violation: %v dispatched as notification to CallHandler", typ))
			}
			return
		}

		callCtx := CallContext{Msg: msg, actor: a}

		if ch.validator != nil && len(ch.validator(msg)) > 0 {
			a.RequestLogger.Warn("Call validation failed")
			rpl := msg.ReplyWithError("Call validation failed", "actor.call.validation")
			callCtx.Reply(rpl)
			return
		}

		ch.handler(callCtx)

		if !msg.HasBeenReplied {
			a.RequestLogger.Error("Message should be replied to but has not been (missing DeferReply/Reply/NoReply)")
			rpl := msg.ReplyWithError("Message should be replied to but has not been", "actor.reply.missing")
			callCtx.Reply(rpl)
		}
		a.queue.GetExecutorReplyChan() <- msg
		return
	}

	// Try Notification handler
	if nh, found := a.notificationHandlers[typ]; found {
		if msg.ShouldBeRepliedTo {
			a.RequestLogger.Error("Message arrived as call but is mapped as a notification handler")
			if a.CrashOnUnhandled {
				panic(fmt.Sprintf("Protocol violation: %v dispatched as call to NotificationHandler", typ))
			}
			rpl := msg.ReplyWithError("Message is a notification but sent as a call", "actor.protocol.violation")
			// Reply safely via internal mechanism
			msg.HasBeenReplied = true
			a.queue.GetExecutorReplyChan() <- rpl
			return
		}

		if nh.validator != nil && len(nh.validator(msg)) > 0 {
			a.RequestLogger.Warn("Notification validation failed")
			return
		}

		notifCtx := NotificationContext{Msg: msg}
		nh.handler(notifCtx)
		msg.HasBeenReplied = true
		a.queue.GetExecutorReplyChan() <- msg
		return
	}

	// Legacy method logic or unhandled mechanism
	v, found := a.methods[typ]
	if !found {
		// Ignore internal notifications mapping if they aren't explicitly caught
		if typ == reflect.TypeOf(ActorStarted{}) || typ == reflect.TypeOf(ActorAboutToStop{}) {
			a.queue.GetExecutorReplyChan() <- msg
			return
		}

		a.RequestLogger.Warn("Unexpected message")
		if msg.ShouldBeRepliedTo {
			a.RequestLogger.Error("Unhandled call message")
			if a.CrashOnUnhandled {
				panic(fmt.Sprintf("Unhandled call: no handler registered for type %v", typ))
			}
			msg.HasBeenReplied = true
			a.queue.GetExecutorReplyChan() <- msg.ReplyWithError("Unhandled message", "actor.message.unhandled")
		} else {
			a.RequestLogger.Error("Unhandled notification message")
			if a.CrashOnUnhandled {
				panic(fmt.Sprintf("Unhandled notification: no handler registered for type %v", typ))
			}
			msg.HasBeenReplied = true
			a.queue.GetExecutorReplyChan() <- msg
		}
		return
	}

	if v.Validator(msg) != nil {
		a.RequestLogger.Warn("Reply validation failed")
		if msg.ShouldBeRepliedTo {
			rpl := msg.ReplyWithError("Reply validation failed", "actor.reply.validation")
			msg.HasBeenReplied = true
			a.queue.GetExecutorReplyChan() <- rpl
		}
		return
	}

	if v.Handler(msg) {
		if msg.ShouldBeRepliedTo {
			if !msg.HasBeenReplied {
				a.RequestLogger.Error("Message should be replied to but has not been")
				rpl := msg.ReplyWithError("Message should be replied to but has not been", "actor.reply.missing")
				msg.HasBeenReplied = true
				a.queue.GetExecutorReplyChan() <- rpl
			}
		} else {
			// Acknowledge notification
			msg.HasBeenReplied = true
			a.queue.GetExecutorReplyChan() <- msg
		}
		return
	} else {
		a.RequestLogger.Warn("Unhandled message")
		if msg.ShouldBeRepliedTo {
			msg.HasBeenReplied = true
			a.queue.GetExecutorReplyChan() <- msg.ReplyWithError("Unhandled message", "actor.message.unhandled")
		}
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
	if a.NotifyStart {
		a.NotifyActor(message.Create(nil, ActorStarted{}, nil))
	}
}

// String
func (a *Actor) String() string {
	return fmt.Sprintf("[A %s]", a.actorName)
}

func (a *Actor) GetQueue() *messagequeue.MessageQueue {
	return &a.queue
}

func (a *Actor) SendActor(msg *message.Message, res chan *message.Message) {
	msg.ReplyChan = res
	a.Logger.WithFields(logrus.Fields{
		"message":      msg.String(),
		"message_type": reflect.TypeOf(msg.TargetMethod).String()}).Debug("Sending message")
	a.queue.Send(msg)
}

func (a *Actor) NotifyActor(msg *message.Message) {
	a.Logger.WithFields(logrus.Fields{
		"message":      msg.String(),
		"message_type": reflect.TypeOf(msg.TargetMethod).String()}).Debug("Notifying message")
	msg.ShouldBeRepliedTo = false
	a.queue.Send(msg)
}

// GetCallbackChan
func (a *Actor) GetCallbackChan() chan *message.Message {
	return a.CallbackChan
}
