package actor

import (
	"reflect"

	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
	"github.com/sirupsen/logrus"
)

// NotificationContext wraps a message for fire-and-forget notification handlers.
// It explicitly lacks Reply() or NoReply() to prevent protocol violations.
// Intent: Provide a read-only context for messages that do not require a response.
// @spec-link [[mech_actor_handler_context]]
type NotificationContext struct {
	// Msg is the original incoming message.
	// Constraint: Must not be mutated by the handler.
	Msg *message.Message
}

// CallContext wraps a message for request-reply handlers.
// It provides the necessary methods to complete the synchronized call safely.
// Intent: Enforce the mandatory reply protocol for synchronous actor communication.
// @spec-link [[mech_actor_handler_context]]
type CallContext struct {
	// Msg is the original incoming message.
	Msg   *message.Message
	actor *Actor
}

// Reply sends a response back to the caller.
// Goal: Completes the Call lifecycle.
// It verifies that the message is actually a request before attempting to reply.
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
// This is used when the caller expects an ACK but no specific return value.
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
// This is critical for preventing deadlocks in complex multi-actor flows.
func (c *CallContext) DeferReply() {
	c.Msg.HasBeenReplied = true
}

// ReplyContext wraps a message received as a reply from another actor.
// It lacks Reply methods to prevent accidental recursive replies.
type ReplyContext struct {
	// Msg is the replied message from another actor.
	Msg *message.Message
}
