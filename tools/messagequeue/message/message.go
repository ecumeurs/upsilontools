package message

import (
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
)

type MessageType int

const (
	Request MessageType = 1
	Reply   MessageType = 2
)

// @spec-link [[mechanic_message_queue]]
// @spec-link [[mechanic_message_queue_management]]

// Message represents a single unit of communication within the engine's message queue.
// It encapsulates the request ID, type, content, and routing information needed
// for asynchronous processing and replying.
type Message struct {
	RequestId uuid.UUID
	Type      MessageType

	Content interface{}

	TargetMethod   interface{}
	CallbackMethod interface{}

	EmitedAt          time.Time
	HasError          bool
	ErrorMessage      string
	ErrorKey          string
	HasBeenReplied    bool
	ShouldBeRepliedTo bool
	ReplyChan         chan *Message
}

// New creates a new Message instance with a unique request ID and default settings.
// It initializes the emitted timestamp and sets the initial state to a request
// that should be replied to, with no errors and a "not yet replied" status.
// This function ensures that every message starts with a valid UUID for tracking.
// It is the standard way to instantiate a message before sending it to a queue.
func New() *Message {
	return &Message{
		RequestId:         uuid.New(),
		Type:              Request,
		EmitedAt:          time.Now(),
		HasBeenReplied:    false,
		ShouldBeRepliedTo: true,
		HasError:          false,
	}
}

// Create generates a new Message populated with the provided content and methods.
// It uses the New() function for base initialization and then assigns the
// specific payload (Content), the TargetMethod to execute, and the CallbackMethod
// to be notified upon completion or error.
func Create(Content, TargetMethod, CallbackMethod interface{}) *Message {
	msg := New()
	msg.Content = Content
	msg.TargetMethod = TargetMethod
	msg.CallbackMethod = CallbackMethod

	return msg
}

// Reply creates a new Message instance as a response to the receiver request message.
// It swaps the target and callback methods and sets the type to Reply.
// The new message inherits the original RequestId to maintain traceability.
func (request *Message) Reply() *Message {
	return &Message{
		RequestId:         request.RequestId,
		Type:              Reply,
		EmitedAt:          time.Now(),
		TargetMethod:      request.CallbackMethod,
		CallbackMethod:    request.TargetMethod,
		ShouldBeRepliedTo: false,
		HasBeenReplied:    false,
	}
}

// ReplyWithError creates a reply message that encapsulates a specific error message and key.
// It is used to signal failure during the processing of a request message.
// The returned message will have the HasError flag set to true.
func (m Message) ReplyWithError(err string, errKey string) *Message {
	res := m.Reply()
	res.HasError = true
	res.ErrorMessage = err
	res.ErrorKey = errKey
	return res
}

// String returns a short string representation of the message including its unique RequestId prefix.
func (m *Message) String() string {
	return fmt.Sprintf("[R %s]", m.RequestId.String()[0:8])
}

// TargetString returns the reflect-based type name of the TargetMethod for debugging and logging.
func (m *Message) TargetString() string {
	if m.TargetMethod == nil {
		return "<nil>"
	}
	return reflect.TypeOf(m.TargetMethod).String()
}

// ContentString returns the reflect-based type name of the message Content payload.
func (m *Message) ContentString() string {
	if m.TargetMethod == nil {
		return "<nil>"
	}
	return reflect.TypeOf(m.Content).String()
}

// CallbackString returns the reflect-based type name of the CallbackMethod for debugging.
func (m *Message) CallbackString() string {
	if m.TargetMethod == nil {
		return "<nil>"
	}
	return reflect.TypeOf(m.CallbackMethod).String()
}
