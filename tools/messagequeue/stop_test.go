package messagequeue

import (
	"testing"
	"time"
)

// TestMessageQueue_StopIdempotency verifies that calling Stop() multiple times 
// does not cause a hang. The second call should return immediately if the queue 
// is already stopped.
func TestMessageQueue_StopIdempotency(t *testing.T) {
	mq := New("StopTestMQ")
	mq.Start()

	// 1. First Stop
	mq.Stop()

	// 2. Second Stop (Currently this is expected to HANG if not fixed)
	done := make(chan bool)
	go func() {
		mq.Stop()
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Errorf("MessageQueue.Stop() hung on second call")
	}
}
