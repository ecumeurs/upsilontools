package messagequeue

import (
	"testing"
	"time"

	"github.com/ecumeurs/upsilontools/logger"
	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
	"github.com/sirupsen/logrus"
)

func TestSendOneSimpleMessageQueue(t *testing.T) {

	mq := New("test")

	mq.Start()

	go func() {
		// expect one request.
		msg := <-mq.GetExecutorChan()
		logrus.Info("Got message: ", msg.String())
		if msg.TargetMethod.(string) != "test" {
			t.Errorf("Expected message to be 'test', got '%s'", msg.String())
		}
		reply := msg.Reply()
		reply.Content = "Done"
		mq.GetExecutorReplyChan() <- reply
	}()

	cb := make(chan *message.Message) // callback
	defer close(cb)
	mq.Send(message.Create(nil, "test", "test"), cb)

	replied := <-cb
	if replied.Content != "Done" {
		t.Errorf("Expected reply to be 'Done', got '%s'", replied.Content)
	}

	mq.Stop()
}

func TestSendMultipleSimpleMessageQueue(t *testing.T) {
	logger.InitConsole()
	mq := New("test")

	mq.Start()

	go func() {
		// expect five request.
		for i := 0; i < 5; i++ {
			msg := <-mq.GetExecutorChan()
			logrus.Info("Got message: ", msg.String())
			if msg.TargetMethod.(string) != "test" {
				t.Errorf("Expected message to be 'test', got '%s'", msg.String())
			}
			reply := msg.Reply()
			reply.Content = "Done"
			mq.GetExecutorReplyChan() <- reply
		}
	}()

	cb := make(chan *message.Message) // callback
	defer close(cb)

	go func() {
		for i := 0; i < 5; i++ {
			replied := <-cb
			if replied.Content != "Done" {
				t.Errorf("Expected reply to be 'Done', got '%s'", replied.Content)
			}
		}
	}()

	mq.Send(message.Create(nil, "test", "test"), cb)
	mq.Send(message.Create(nil, "test", "test"), cb)
	mq.Send(message.Create(nil, "test", "test"), cb)
	mq.Send(message.Create(nil, "test", "test"), cb)
	mq.Send(message.Create(nil, "test", "test"), cb)

	<-time.After(1 * time.Second)

	mq.Stop()
}

func TestSendHundredsSimpleMessageQueue(t *testing.T) {
	logger.InitConsole()
	mq := New("test")

	mq.Start()

	max := 1000

	end := make(chan bool)

	go func() {
		last := -1
		for i := 0; i < max; i++ {
			msg := <-mq.GetExecutorChan()
			logrus.Info("Got message: ", msg.String())
			if msg.TargetMethod.(string) != "test" {
				t.Errorf("Expected message to be 'test', got '%s'", msg.String())
			}
			if msg.Content.(int) != last+1 {
				t.Errorf("Expected message to come in the right order ( expected '%d', got '%d')", last+1, msg.Content.(int))
			}
			last = msg.Content.(int)
			reply := msg.Reply()
			reply.Content = "Done"
			mq.GetExecutorReplyChan() <- reply
		}
		end <- true
	}()

	cb := make(chan *message.Message) // callback
	defer close(cb)

	go func() {
		for i := 0; i < max; i++ {
			replied := <-cb
			if replied.Content != "Done" {
				t.Errorf("Expected reply to be 'Done', got '%s'", replied.Content)
			}
		}
	}()

	for i := 0; i < max; i++ {
		mq.Send(message.Create(i, "test", "test"), cb)
	}

	<-end

	mq.Stop()
}
