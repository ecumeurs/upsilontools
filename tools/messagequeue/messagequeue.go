package messagequeue

import (
	"fmt"
	"sync"

	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
	"github.com/sirupsen/logrus"
)

// @spec-link [[mechanic_message_queue]]
// @spec-link [[mechanic_message_queue_management]]
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

// New creates a new MessageQueue instance with the given name.
// It initializes all necessary internal channels for message input, execution,
// and coordination, as well as the logger with the appropriate context.
// The returned queue is ready to be started via the Start() method.
// This constructor is the standard way to instantiate a FIFO message queue.
// It ensures that all internal maps and channels are properly allocated.
// The queue name is used for logging and observability purposes throughout its lifecycle.
// Callers must call Start() to begin background processing of incoming stimuli.
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

// PrintStack logs the current contents of the message queue, including the message currently
// being processed and all pending messages in the FIFO buffer.
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

// Start begins the background processing loop for the message queue.
// It spawns a goroutine that orchestrates the FIFO delivery of messages to the executor.
// The loop handles incoming stimuli, execution acknowledgments, and shutdown signals.
// @spec-link [[mechanic_message_queue_management]]
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

// Stop immediately terminates the message queue's processing loop.
// Pending messages in the buffer will NOT be processed.
func (mq *MessageQueue) Stop() {
	mq.stopChan <- true
}

// PrepareStop initiates a graceful shutdown. It prevents new messages from being accepted
// and returns a channel that will be closed once all currently queued messages have been processed.
// @spec-link [[mechanic_message_queue_management]]
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

// Send adds a message to the queue for processing.
// If the queue is in the process of stopping, the message is ignored.
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

// SendAndForget is an alias for Send, signifying that the caller does not expect a reply.
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

// GetExecutorChan returns the channel where the dispatcher listens for messages to process.
func (mq *MessageQueue) GetExecutorChan() chan *message.Message {
	return mq.executorChan
}

// GetExecutorReplyChan returns the channel where the dispatcher sends acknowledgments after processing.
func (mq *MessageQueue) GetExecutorReplyChan() chan *message.Message {
	return mq.executorReplyChan
}

// Length returns the number of messages currently waiting in the queue.
func (mq *MessageQueue) Length() int {
	mq.mu.Lock()
	defer mq.mu.Unlock()
	return len(mq.messages)
}

// String returns a human-readable summary of the queue state.
func (mq *MessageQueue) String() string {
	return fmt.Sprintf("[MQ %s] Length: %d", mq.Name, mq.Length())
}
