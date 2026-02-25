package message

import (
	"fmt"
	"reflect"
	"time"

	"github.com/google/uuid"
)

type MessageType int

const (
	Request int = 1
	Reply   int = 2
)

type Message struct {
	RequestId uuid.UUID

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

// New
func New() *Message {
	return &Message{
		RequestId:         uuid.New(),
		EmitedAt:          time.Now(),
		HasBeenReplied:    false,
		ShouldBeRepliedTo: true,
		HasError:          false,
	}
}

func Create(Content, TargetMethod, CallbackMethod interface{}) *Message {
	msg := New()
	msg.Content = Content
	msg.TargetMethod = TargetMethod
	msg.CallbackMethod = CallbackMethod

	return msg
}

// NewReply
func (request *Message) Reply() *Message {
	return &Message{
		RequestId:         request.RequestId,
		EmitedAt:          time.Now(),
		TargetMethod:      request.CallbackMethod,
		CallbackMethod:    request.TargetMethod,
		ShouldBeRepliedTo: false,
		HasBeenReplied:    false,
	}
}

// ReplyWithError
func (m Message) ReplyWithError(err string, errKey string) *Message {
	res := m.Reply()
	res.HasError = true
	res.ErrorMessage = err
	res.ErrorKey = errKey
	return res
}

// String
func (m *Message) String() string {
	return fmt.Sprintf("[R %s]", m.RequestId.String()[0:8])
}

func (m *Message) TargetString() string {
	if m.TargetMethod == nil {
		return "<nil>"
	}
	return reflect.TypeOf(m.TargetMethod).String()
}

func (m *Message) ContentString() string {
	if m.TargetMethod == nil {
		return "<nil>"
	}
	return reflect.TypeOf(m.Content).String()
}

func (m *Message) CallbackString() string {
	if m.TargetMethod == nil {
		return "<nil>"
	}
	return reflect.TypeOf(m.CallbackMethod).String()
}
