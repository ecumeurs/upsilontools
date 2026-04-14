package messagequeue

import (
	"fmt"
	"sync"

	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
	"github.com/sirupsen/logrus"
)

// @spec-link [[mech_message_queue]]
type MessageQueue struct {
	mu                    sync.Mutex
	Name                  string
	inputChan             chan *message.Message
	executorReplyChan     chan *message.Message
	executorChan          chan *message.Message
	stopChan              chan bool
	doneChan              chan bool // filled when the queue is empty
	dontAcceptNewMessages bool
	logger                *logrus.Entry

	currentMessage *message.Message

	messages []*message.Message
}

// New
func New(name string) *MessageQueue {
	mq := &MessageQueue{
		Name:                  name,
		inputChan:             make(chan *message.Message),
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
	mq.mu.Lock()
	defer mq.mu.Unlock()
	// format all pending messages as a string
	var pendingMessages string
	for _, msg := range mq.messages {
		pendingMessages += fmt.Sprintf("%s - %s,", msg.String(), msg.TargetString())
	}

	var currentMessage string
	if mq.currentMessage != nil {
		currentMessage = mq.currentMessage.String()
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
				mq.mu.Lock()
				if mq.dontAcceptNewMessages {
					mq.mu.Unlock()
					mq.logger.WithField("message", msg.String()).Debug("Message queue is stopping, ignoring message")
					continue
				}
				mq.logger.WithFields(logrus.Fields{
					"message":      msg.String(),
					"message_type": msg.TargetString()}).Debug("Received message")
				mq.messages = append(mq.messages, msg)
				if mq.currentMessage == nil {
					mq.currentMessage = msg
					go func(msg *message.Message) {
						mq.logger.WithFields(logrus.Fields{
							"message":      msg.String(),
							"message_type": msg.TargetString()}).Debug("Executing message")

						mq.executorChan <- msg
					}(msg)
				}
				mq.mu.Unlock()
			case msg := <-mq.executorReplyChan:
				mq.logger.WithFields(logrus.Fields{
					"message":      msg.String(),
					"message_type": msg.TargetString()}).Debug("Execution Ack Received")

				mq.mu.Lock()
				mq.currentMessage = nil
				if len(mq.messages) == 0 {
					mq.mu.Unlock()
					mq.logger.Warn("Received Execution Ack but message queue is empty (phantom ACK). Ignoring.")
					continue
				}
				mq.messages = mq.messages[1:]

				if len(mq.messages) > 0 {
					mq.currentMessage = mq.messages[0]
					go func(msg *message.Message) {
						mq.logger.WithFields(logrus.Fields{
							"message":      msg.String(),
							"message_type": msg.TargetString()}).Debug("Executing New Message")
						mq.executorChan <- msg
					}(mq.currentMessage)
				} else {
					mq.logger.Debug("Message queue is empty")
					if mq.dontAcceptNewMessages {
						mq.doneChan <- true
					}
				}
				mq.mu.Unlock()
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
	mq.mu.Lock()
	defer mq.mu.Unlock()
	mq.dontAcceptNewMessages = true
	if len(mq.messages) == 0 {
		go func() {
			mq.doneChan <- true
		}()
	} else {
		mq.logger.WithField("length", len(mq.messages)).Debug("Message queue is not empty, waiting for it to be empty")
	}
	return mq.doneChan
}

// Send a message to the message queue
func (mq *MessageQueue) Send(msg *message.Message) {
	mq.mu.Lock()
	dontAccept := mq.dontAcceptNewMessages
	mq.mu.Unlock()
	if !dontAccept {
		mq.inputChan <- msg
	} else {
		mq.logger.WithField("message", msg.String()).Debug("Message queue is stopping, ignoring message")
	}
}

// Send a message to the message queue and forget (wont reply)
func (mq *MessageQueue) SendAndForget(msg *message.Message) {
	mq.mu.Lock()
	dontAccept := mq.dontAcceptNewMessages
	mq.mu.Unlock()
	if !dontAccept {
		mq.inputChan <- msg
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
	mq.mu.Lock()
	defer mq.mu.Unlock()
	return len(mq.messages)
}

// String
func (mq *MessageQueue) String() string {
	return fmt.Sprintf("[MQ %s] Length: %d", mq.Name, mq.Length())
}
