package messagequeue

import (
	"fmt"
	"reflect"

	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
	"github.com/sirupsen/logrus"
)

type internalMessage struct {
	Message  *message.Message
	Callback chan *message.Message
}

type MessageQueue struct {
	Name                  string
	inputChan             chan internalMessage
	executorReplyChan     chan *message.Message
	executorChan          chan *message.Message
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
		executorReplyChan:     make(chan *message.Message),
		executorChan:          make(chan *message.Message),
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

// printStack debug function
func (mq *MessageQueue) PrintStack() {
	// format all pending messages as a string
	var pendingMessages string
	for _, msg := range mq.messages {
		pendingMessages += fmt.Sprintf("%s - %s,", msg.Message.String(), reflect.TypeOf(msg.Message.TargetMethod).String())
	}

	var currentMessage string
	if mq.currentMessage != nil {
		currentMessage = mq.currentMessage.Message.String()
	} else {
		currentMessage = "None"
	}

	mq.logger.WithFields(logrus.Fields{
		"messages": pendingMessages,
		"current":  currentMessage,
	}).Info("Stack")
}

// Start a thread that will expect input messages to come in, and will store them in the messages slice
// When a message is received, it will be sent to the executorReplyChan if no other message are being processed at the moment.
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
					"message_type": reflect.TypeOf(msg.Message.TargetString())}).Debug("Received message")
				mq.messages = append(mq.messages, msg)
				if mq.currentMessage == nil {
					mq.currentMessage = &msg
					go func(msg *message.Message) {
						mq.logger.WithFields(logrus.Fields{
							"message":      msg.String(),
							"message_type": msg.TargetString()}).Debug("Executing message")

						mq.executorChan <- msg
					}(msg.Message)
				}
			case msg := <-mq.executorReplyChan:
				mq.logger.WithFields(logrus.Fields{
					"message":      msg.String(),
					"message_type": msg.TargetString()}).Debug("Reply Received")

				if mq.currentMessage.Callback != nil {

					go func(repl *message.Message, callback chan *message.Message) {
						mq.logger.WithFields(logrus.Fields{
							"message":      msg.String(),
							"message_type": msg.TargetString()}).Debug("Executing ReplyCallback")

						callback <- repl
					}(msg, mq.currentMessage.Callback)
				}
				mq.currentMessage = nil
				mq.messages = mq.messages[1:]

				if len(mq.messages) > 0 {
					mq.currentMessage = &mq.messages[0]
					go func(msg *message.Message) {
						mq.logger.WithFields(logrus.Fields{
							"message":      msg.String(),
							"message_type": msg.TargetString()}).Debug("Executing New Message")
						mq.executorChan <- msg
					}(mq.currentMessage.Message)
				} else {
					mq.logger.Debug("Message queue is empty")
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
	} else {
		mq.logger.WithField("length", mq.Length()).Debug("Message queue is not empty, waiting for it to be empty")
	}
	return mq.doneChan
}

// Send a message to the message queue
func (mq *MessageQueue) Send(msg *message.Message, callback chan *message.Message) {
	if !mq.dontAcceptNewMessages {
		mq.inputChan <- internalMessage{
			Message:  msg,
			Callback: callback,
		}
	} else {
		mq.logger.WithField("message", msg.String()).Debug("Message queue is stopping, ignoring message")
	}
}

// Send a message to the message queue and forget (wont reply)
func (mq *MessageQueue) SendAndForget(msg *message.Message) {
	if !mq.dontAcceptNewMessages {
		mq.inputChan <- internalMessage{
			Message:  msg,
			Callback: nil,
		}
	} else {
		mq.logger.WithField("message", msg.String()).Debug("Message queue is stopping, ignoring message")
	}
}

func (mq *MessageQueue) GetExecutorChan() chan *message.Message {
	return mq.executorChan
}

func (mq *MessageQueue) GetExecutorReplyChan() chan *message.Message {
	return mq.executorReplyChan
}

// Length
func (mq *MessageQueue) Length() int {
	return len(mq.messages)
}

// String
func (mq *MessageQueue) String() string {
	return fmt.Sprintf("[MQ %s] Length: %d", mq.Name, mq.Length())
}
