package actor

import (
	"fmt"
	"reflect"

	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
	"github.com/sirupsen/logrus"
)

// processReply handles incoming replies to messages sent by this actor.
// It matches the reply type against registered reply handlers or legacy methods.
// @spec-link [[mech_actor_dispatch_loop]]
func (a *Actor) processReply(msg *message.Message) {
	a.RequestLogger = a.Logger.WithFields(logrus.Fields{
		"request_type": "reply",
		"message":      msg.String(),
		"message_type": msg.TargetString(),
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
			// Cannot "reply" to a reply. Just drop but ACK.
			a.queue.GetExecutorReplyChan() <- msg
			return
		}
		replyCtx := ReplyContext{Msg: msg}
		rh.handler(replyCtx)
		a.queue.GetExecutorReplyChan() <- msg
		return
	}

	v, found := a.replies[typ]
	if !found {
		a.RequestLogger.Warn("Unexpected reply")
		if a.CrashOnUnhandled {
			typeName := "<nil>"
			if typ != nil {
				typeName = typ.String()
			}
			panic(fmt.Sprintf("Unhandled reply msg type %s", typeName))
		}
		a.queue.GetExecutorReplyChan() <- msg
		return
	}

	if v.Validator(msg) != nil {
		a.RequestLogger.Warn("Reply validation failed")
		a.queue.GetExecutorReplyChan() <- msg
		return // we can't reply to a reply anyway
	}

	if v.Handler(msg) {
		a.queue.GetExecutorReplyChan() <- msg
		return
	} else {
		a.RequestLogger.Warn("Unhandled reply")
		a.queue.GetExecutorReplyChan() <- msg
		return
	}
}

// processMessage handles incoming requests (Calls and Notifications).
// It orchestrates the validation and execution of the appropriate handler.
// @spec-link [[mech_actor_dispatch_loop]]
func (a *Actor) processMessage(msg *message.Message) {
	a.RequestLogger = a.Logger.WithFields(logrus.Fields{
		"request_type": "message",
		"message":      msg.String(),
		"message_type": msg.TargetString(),
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
			a.queue.GetExecutorReplyChan() <- msg // Send ACK to queue to unblock
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
		} else {
			// Acknowledge unhandled notification to unblock the queue
			msg.HasBeenReplied = true
			a.queue.GetExecutorReplyChan() <- msg
		}
		return
	}
}
