package actor

import (
	"fmt"
	"reflect"
	"time"

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

// --- EXTRACTED TO context.go ---

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

// MethodType returns the reflect.Type of the message this method handles.
// This is used by the actor's dispatch loop to route messages to the correct handler.
func (a actorMethodImpl) MethodType() reflect.Type {
	return a.methodType
}

// Handler executes the logic for this method given a raw message.
// It returns true if the message was successfully handled, false otherwise.
func (a actorMethodImpl) Handler(msg *message.Message) (handled bool) {
	if a.handler == nil {
		return false
	}
	return a.handler(msg)
}

// Validator checks if the incoming message is structurally and semantically valid 
// for this specific method. It returns a slice of errors if validation fails.
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

// @spec-link [[mech_actor_pattern]]
type Actor struct {
	actorName     string
	queue         messagequeue.MessageQueue
	CallbackChan  chan *message.Message
	stopper       chan bool
	// Logger provides foundational observability for the actor's operations.
	// @spec-link [[requirement_req_observability_logging]]
	Logger *logrus.Entry
	// RequestLogger is a contextual logger for the currently processing message.
	// @spec-link [[requirement_req_observability_logging]]
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

// New creates and initializes a new Actor instance with the given name.
// This constructor is the primary entry point for creating independent, 
// concurrent processing units in the Upsilon ecosystem. It performs the 
// following critical initializations:
// 1. Allocates a unique name for the actor used in diagnostic logging.
// 2. Initializes a FIFO message queue to handle incoming stimuli sequentially.
// 3. Creates the callback channel for internal stimulus redirection.
// 4. Initializes the various handler maps for modern and legacy protocols.
// The actor is returned in an unstarted state; the caller is responsible 
// for calling Start() to begin the internal dispatch loop. 
// Intent: Standardize the creation of concurrent units within the Upsilon ecosystem
// to ensure thread-safety, clear ownership, and predictable message-driven behavior.
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

// AddMethod registers a legacy-style handler for a specific message type.
// It includes a validator function and a handler function that returns whether the message was handled.
// Deprecated: Use AddCallHandler or AddNotificationHandler for modern gated contexts.
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

// AddActorMethod registers a pre-implemented ActorMethod interface.
// This allows for complex handlers that encapsulate their own validation and logic.
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

// AddReply registers a legacy-style handler for a reply message type.
// Deprecated: Use AddReplyHandler for modern gated contexts.
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

// AddActorReply registers a pre-implemented ActorMethod interface for reply handling.
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

// AddCallHandler registers a synchronized request-reply handler using the CallContext.
// This is the preferred way to handle messages that require a mandatory response.
func (a *Actor) AddCallHandler(method interface{}, handler func(ctx CallContext), validator func(msg *message.Message) []error) {
	t := reflect.TypeOf(method)
	if _, ok := a.callHandlers[t]; ok {
		a.Logger.WithFields(logrus.Fields{"method": method}).Warn("Call Handler already registered")
		return
	}
	a.callHandlers[t] = callHandlerImpl{handler: handler, validator: validator}
}

// AddReplyHandler registers a handler for an incoming reply message using ReplyContext.
// This allows the actor to react to responses from other actors in a type-safe way.
func (a *Actor) AddReplyHandler(method interface{}, handler func(ctx ReplyContext), validator func(msg *message.Message) []error) {
	t := reflect.TypeOf(method)
	if _, ok := a.replyHandlers[t]; ok {
		a.Logger.WithFields(logrus.Fields{"reply": method}).Warn("Reply Handler already registered")
		return
	}
	a.replyHandlers[t] = replyHandlerImpl{handler: handler, validator: validator}
}

// Name returns the human-readable name of the actor.
func (a *Actor) Name() string {
	return a.actorName
}

// Stop terminates the actor's message queue and signals background loops to exit.
// This is the standard way to decommission an actor in the ecosystem.
// @spec-link [[mech_actor_lifecycle]]
func (a *Actor) Stop() {
	a.queue.Stop()
	close(a.stopper)
}

// PrepareToStop will disconnect the actor from the message queue and return a channel that will tell when all pending messages have been processed.
// @spec-link [[mech_actor_lifecycle]]
func (a *Actor) PrepareToStop() chan bool {
	return a.queue.PrepareStop()
}

// Legacy Reply / NoReply deleted per ISS-004 to enforce Context usage.
// They are now owned by CallContext.

// @spec-link [[mech_actor_dispatch_loop]]
// --- EXTRACTED TO dispatch.go ---

// Start begins the actor's internal background loops for message processing.
// It initializes the message queue, starts the callback redirection loop, 
// and begins the main dispatch loop.
// @spec-link [[mech_actor_lifecycle]]
func (a *Actor) Start() {
	a.queue.Start()
	go a.runCallbackRedirectLoop()
	go a.runMainDispatchLoop()

	if a.NotifyStart {
		a.NotifyActor(message.Create(nil, ActorStarted{}, nil))
	}
}

// runCallbackRedirectLoop redirects internal stimuli from CallbackChan to the main queue.
// This prevents deadlocks when an actor's handler sends a message to itself.
func (a *Actor) runCallbackRedirectLoop() {
	for {
		select {
		case msg := <-a.CallbackChan:
			a.queue.Send(msg)
		case <-a.stopper:
			return
		}
	}
}

// runMainDispatchLoop is the primary execution thread for the actor.
// It pulls messages from the queue and routes them to processReply or processMessage.
// @spec-link [[mech_actor_dispatch_loop]]
func (a *Actor) runMainDispatchLoop() {
	a.Logger.Info("Actor started")
	for {
		select {
		case msg := <-a.queue.GetExecutorChan():
			if msg.Type == message.Reply {
				a.Logger.WithFields(logrus.Fields{
					"message":      msg.String(),
					"message_type": msg.TargetString()}).Debug("About to process reply from queue")
				a.processReply(msg)
			} else {
				a.Logger.WithFields(logrus.Fields{
					"message":      msg.String(),
					"message_type": msg.TargetString()}).Debug("About to process message from queue")
				a.processMessage(msg)
			}
		case <-a.stopper:
			a.Logger.Info("Actor Stopped")
			return
		}
	}
}

// String returns a string representation of the actor for logging and debugging.
func (a *Actor) String() string {
	return fmt.Sprintf("[A %s]", a.actorName)
}

// GetQueue returns the underlying message queue of the actor.
func (a *Actor) GetQueue() *messagequeue.MessageQueue {
	return &a.queue
}

// SendActor sends a message to another actor and expects a response on the provided channel.
// Goal: Initiate a request-reply lifecycle.
func (a *Actor) SendActor(msg *message.Message, res chan *message.Message) {
	if msg.CallbackMethod == nil {
		panic(fmt.Sprintf("[%s] Protocol violation: SendActor called with nil CallbackMethod for message %s. Use NotifyActor for fire-and-forget.", a.Name(), msg.TargetString()))
	}
	if res == nil {
		panic(fmt.Sprintf("[%s] Protocol violation: SendActor called with nil response channel for message %s. Provide a channel or use NotifyActor if you don't care about the response.", a.Name(), msg.TargetString()))
	}
	msg.ReplyChan = res
	a.Logger.WithFields(logrus.Fields{
		"message":      msg.String(),
		"message_type": msg.TargetString()}).Debug("Sending message")
	a.queue.Send(msg)
}

// NotifyActor sends a fire-and-forget notification to another actor.
// No response is expected, and the message is marked as such.
func (a *Actor) NotifyActor(msg *message.Message) {
	a.Logger.WithFields(logrus.Fields{
		"message":      msg.String(),
		"message_type": msg.TargetString()}).Debug("Notifying message")
	msg.ShouldBeRepliedTo = false
	a.queue.Send(msg)
}

// SelfNotify sends a notification to the actor's own queue.
// TargetMethod must be a struct registered via AddNotificationHandler.
func (a *Actor) SelfNotify(method interface{}) {
	a.NotifyActor(message.Create(nil, method, nil))
}

// SelfNotifyDelayed schedules a notification to the actor's own queue after a delay.
func (a *Actor) SelfNotifyDelayed(method interface{}, delay time.Duration) {
	time.AfterFunc(delay, func() {
		a.SelfNotify(method)
	})
}

// SelfNotifyMessageDelayed schedules an existing message to the actor's own queue after a delay.
func (a *Actor) SelfNotifyMessageDelayed(msg *message.Message, delay time.Duration) {
	time.AfterFunc(delay, func() {
		a.NotifyActor(msg)
	})
}

// SelfDispatchMessageDelayed schedules an existing message to the actor's own queue after a delay,
// preserving the message's original ShouldBeRepliedTo status.
func (a *Actor) SelfDispatchMessageDelayed(msg *message.Message, delay time.Duration) {
	time.AfterFunc(delay, func() {
		a.queue.Send(msg)
	})
}

// PrintStack logs the current state of the actor's message queue for debugging.
func (a *Actor) PrintStack() {
	a.queue.PrintStack()
}

// GetCallbackChan returns the channel used by the actor to receive callback responses.
func (a *Actor) GetCallbackChan() chan *message.Message {
	return a.CallbackChan
}
