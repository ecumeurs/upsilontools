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
	msg := message.Create(nil, "test", nil)
	msg.ReplyChan = cb
	mq.Send(msg)

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

	for i := 0; i < 5; i++ {
		msg := message.Create(nil, "test", nil)
		msg.ReplyChan = cb
		mq.Send(msg)
	}

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
		msg := message.Create(i, "test", nil)
		msg.ReplyChan = cb
		mq.Send(msg)
	}

	<-end

	mq.Stop()
}

func TestMessageQueue_PanicReproduction(t *testing.T) {
	// Goal: Reproduce "slice bounds out of range [1:0]" panic
	// This happens when an ACK is received for an empty queue.
	mq := New("panic-repro")
	mq.Start()

	// Bypass Send and trigger the Ack channel directly when messages slice is 0
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Successfully captured expected panic: %v", r)
		}
	}()

	logrus.Info("Sending unexpected ACK to empty queue...")
	mq.GetExecutorReplyChan() <- message.Create(nil, "phantom", nil)

	// If we reach here without recovery, the test failed to highlight the weakness (or it didn't panic)
	t.Errorf("Queue did not panic on unexpected ACK. Is it already fixed or handled?")
	mq.Stop()
}

func TestMessageQueue_ConcurrentLoad(t *testing.T) {
	// Goal: Stress test the internal slice management under concurrent pressure.
	// This might highlight the lack of mutex protection for the messages slice header.
	mq := New("stress-test")
	mq.Start()

	maxMessages := 1000
	workers := 10

	done := make(chan bool)

	// Consumer loop
	go func() {
		for i := 0; i < maxMessages; i++ {
			msg := <-mq.GetExecutorChan()
			mq.GetExecutorReplyChan() <- msg
		}
		done <- true
	}()

	// Producer loop (concurrent)
	for w := 0; w < workers; w++ {
		go func(workerID int) {
			for i := 0; i < maxMessages/workers; i++ {
				mq.Send(message.Create(i, "load", nil))
			}
		}(w)
	}

	select {
	case <-done:
		logrus.Info("Stress test completed successfully")
	case <-time.After(10 * time.Second):
		t.Errorf("Stress test timed out - possible deadlock or hang")
	}

	mq.Stop()
}
