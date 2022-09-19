package messagequeue

import (
	"fmt"

	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
)

type internalMessage struct {
	Message  message.Message
	Callback chan message.Message
}

type MessageQueue struct {
	Name      string
	inputChan chan internalMessage
	actorChan chan message.Message
	stopChan  chan bool

	currentMessage *internalMessage

	messages []internalMessage
}

// New
func New(name string) *MessageQueue {
	return &MessageQueue{
		Name:      name,
		inputChan: make(chan internalMessage),
		actorChan: make(chan message.Message),
		stopChan:  make(chan bool),
	}
}

// Start a thread that will expect input messages to come in, and will store them in the messages slice
// When a message is received, it will be sent to the actorChan if no other message are being processed at the moment.
func (mq *MessageQueue) Start() {
	go func() {
		for {
			select {
			case msg := <-mq.inputChan:
				mq.messages = append(mq.messages, msg)
				if mq.currentMessage == nil {
					mq.currentMessage = &msg
					mq.actorChan <- msg.Message
				}
			case msg := <-mq.actorChan:
				mq.currentMessage.Callback <- msg
				mq.currentMessage = nil
				mq.messages = mq.messages[1:]
				if len(mq.messages) > 0 {
					mq.currentMessage = &mq.messages[0]
					mq.actorChan <- mq.currentMessage.Message
				}
			case <-mq.stopChan:
				return
			}
		}
	}()
}

// Stop the message queue
func (mq *MessageQueue) Stop() {
	mq.stopChan <- true
}

// Send a message to the message queue
func (mq *MessageQueue) Send(msg message.Message, callback chan message.Message) {
	mq.inputChan <- internalMessage{
		Message:  msg,
		Callback: callback,
	}
}

// GetActorChan
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
