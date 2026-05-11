package actor

import (
	"testing"
	"time"

	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
)

// TestActor_StopIdempotency verifies that calling Stop() multiple times 
// does not cause a panic (close of closed channel) or a hang.
// This is critical for robust lifecycle management during arena destruction.
func TestActor_StopIdempotency(t *testing.T) {
	a := New("StopTestActor")
	a.Start()

	// 1. First Stop
	a.Stop()

	// 2. Second Stop (Should not panic or hang)
	done := make(chan bool)
	go func() {
		a.Stop()
		done <- true
	}()

	select {
	case <-done:
		// Success: second stop handled
	case <-time.After(2 * time.Second):
		t.Errorf("Actor.Stop() hung on second call (likely blocked on MessageQueue.Stop())")
	}
}

// TestActor_StopByMessageIdempotency verifies that receiving multiple ActorStop messages
// is handled correctly without panicking.
func TestActor_StopByMessageIdempotency(t *testing.T) {
	a := New("MessageStopActor")
	a.Start()

	// Send multiple Stop messages
	a.NotifyActor(message.Create(nil, ActorStop{}, nil))
	a.NotifyActor(message.Create(nil, ActorStop{}, nil))

	// Wait a bit to ensure they are processed
	time.Sleep(100 * time.Millisecond)
	
	// If it didn't panic, it's a win for the actor (though it might still hang if MQ is not fixed)
}
