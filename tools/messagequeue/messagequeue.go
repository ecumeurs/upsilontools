package messagequeue

import (
	"fmt"
	"reflect"

	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
	"github.com/sirupsen/logrus"
)

type internalMessage struct {
	Message  message.Message
	Callback chan message.Message
}

type MessageQueue struct {
	Name                  string
	inputChan             chan internalMessage
	actorChan             chan message.Message
	stopChan              chan bool
	doneChan              chan bool // filled when the queue is empty
	dontAcceptNewMessages bool
	logger                *logrus.Entry

	currentMessage *internalMessage

	messages []internalMessage
}

// New
func New(name string) *MessageQueue {
	mq := &MessageQueue{
		Name:                  name,
		inputChan:             make(chan internalMessage),
		actorChan:             make(chan message.Message),
		stopChan:              make(chan bool),
		doneChan:              make(chan bool),
		dontAcceptNewMessages: false,
	}

	mq.logger = logrus.WithFields(logrus.Fields{
		"component": "messagequeue",
		"name":      mq.Name,
	})
	return mq
}

// Start a thread that will expect input messages to come in, and will store them in the messages slice
// When a message is received, it will be sent to the actorChan if no other message are being processed at the moment.
func (mq *MessageQueue) Start() {
	go func() {
		mq.logger.Info("Starting message queue")
		for {
			select {
			case msg := <-mq.inputChan:
				if mq.dontAcceptNewMessages {
					mq.logger.WithField("message", msg.Message.String()).Debug("Message queue is stopping, ignoring message")
					continue
				}
				mq.logger.WithFields(logrus.Fields{
					"message":      msg.Message.String(),
					"message_type": reflect.TypeOf(msg.Message.TargetMethod).String()}).Debug("Received message")
				mq.messages = append(mq.messages, msg)
				if mq.currentMessage == nil {
					mq.currentMessage = &msg
					mq.actorChan <- msg.Message
				}
			case msg := <-mq.actorChan:
				mq.logger.WithFields(logrus.Fields{
					"message":      msg.String(),
					"message_type": reflect.TypeOf(msg.TargetMethod)}).Debug("Reply Received")

				if mq.currentMessage.Callback != nil {
					mq.currentMessage.Callback <- msg
				}
				mq.currentMessage = nil
				mq.messages = mq.messages[1:]

				if len(mq.messages) > 0 {
					mq.currentMessage = &mq.messages[0]

					mq.logger.WithFields(logrus.Fields{
						"message":      mq.currentMessage.Message.String(),
						"message_type": reflect.TypeOf(mq.currentMessage.Message.TargetMethod).String()}).Debug("Reply Received")
					mq.actorChan <- mq.currentMessage.Message
				} else {
					if mq.dontAcceptNewMessages {
						mq.doneChan <- true
					}
				}
			case <-mq.stopChan:
				mq.logger.Info("Stopping message queue")
				return
			}
		}
	}()
}

// Stop the message queue
func (mq *MessageQueue) Stop() {
	mq.stopChan <- true
}

// PrepareStop will prevent new message from being added to the queue, and will return a channel that will be closed when the queue is empty
func (mq *MessageQueue) PrepareStop() chan bool {
	mq.dontAcceptNewMessages = true
	if mq.Length() == 0 {
		go func() {
			mq.doneChan <- true
		}()
	}
	return mq.doneChan
}

// Send a message to the message queue
func (mq *MessageQueue) Send(msg message.Message, callback chan message.Message) {
	if !mq.dontAcceptNewMessages {
		mq.inputChan <- internalMessage{
			Message:  msg,
			Callback: callback,
		}
	}
}

// Send a message to the message queue and forget (wont reply)
func (mq *MessageQueue) SendAndForget(msg message.Message) {
	if !mq.dontAcceptNewMessages {
		mq.inputChan <- internalMessage{
			Message:  msg,
			Callback: nil,
		}
	}
}

// GetActorChan That's the reply channel
func (mq *MessageQueue) GetActorChan() chan message.Message {
	return mq.actorChan
}

// Length
func (mq *MessageQueue) Length() int {
	return len(mq.messages)
}

// String
func (mq *MessageQueue) String() string {
	return fmt.Sprintf("[MQ %s] Length: %d", mq.Name, mq.Length())
}
