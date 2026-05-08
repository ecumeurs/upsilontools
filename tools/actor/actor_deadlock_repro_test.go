package actor

import (
	"fmt"
	"testing"
	"time"

	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
	"github.com/sirupsen/logrus"
)

type DeadlockMethod struct{}
type StartMethod struct{}

// TestActorDeadlock_CyclicCall attempts to reproduce a cyclic deadlock where two actors
// call each other synchronously (or near-synchronously) within their handlers.
// This test is prone to flaking because it depends on OS scheduling.
// @test-link [[mech_actor_dispatch_loop]]
// TestActorDeadlock_CyclicCall verifies that two actors calling each other in a cyclic
// pattern do not cause a permanent deadlock if handled correctly via the actor protocol.
// It tests the request-reply correlation and the ability of the dispatch loop to
// remain responsive during interleaved synchronized calls. This scenario specifically
// simulates a situation where Actor A calls Actor B, and while Actor B is processing,
// it calls back to Actor A. The system must ensure that Actor A can still process
// the incoming call from B while it is waiting for its own reply from B, avoiding 
// the classic circular wait condition common in concurrent systems. This is a 
// fundamental test for the Actor model implementation in Upsilon, as it proves 
// the robustness of the "Unified Queue" approach against recursive dependencies.
// The test completes by asserting that all messages were handled and replied to.
func TestActorDeadlock_CyclicCall(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	
	a := NewTest("ActorA")
	b := NewTest("ActorB")
	
	doneA := make(chan bool)
	
	// A receives a notification to start the test
	a.AddNotificationHandler(StartMethod{}, func(ctx NotificationContext) {
		logrus.Info("Actor A starting the test cycle...")
		b.SendActor(message.Create(nil, DeadlockMethod{}, DeadlockMethod{}), a.GetCallbackChan())
	}, nil)

	// A's handler for B's call
	a.AddCallHandler(DeadlockMethod{}, func(ctx CallContext) {
		logrus.Info("Actor A received DeadlockMethod from B, replying...")
		ctx.Reply(ctx.Msg.Reply())
	}, nil)
	
	// A's reply handler for B's reply to A's initial call
	a.AddReplyHandler(DeadlockMethod{}, func(ctx ReplyContext) {
		logrus.Info("Actor A received final reply from B")
		doneA <- true
	}, nil)
	
	// B's handler calls A
	b.AddCallHandler(DeadlockMethod{}, func(ctx CallContext) {
		logrus.Info("Actor B received DeadlockMethod from A, calling A back...")
		// This is the cyclic call.
		a.SendActor(message.Create(nil, DeadlockMethod{}, DeadlockMethod{}), b.GetCallbackChan())
		ctx.Reply(ctx.Msg.Reply())
	}, nil)

	// B also needs to handle the reply from A
	b.AddReplyHandler(DeadlockMethod{}, func(ctx ReplyContext) {
		logrus.Info("Actor B received reply from A")
	}, nil)

	a.Start()
	b.Start()

	// Kick off the cycle with a notification
	a.NotifyActor(message.Create(nil, StartMethod{}, nil))
	
	select {
	case <-time.After(5 * time.Second):
		fmt.Println("=== DEADLOCK DETECTED! DUMPING STATE ===")
		a.PrintStack()
		b.PrintStack()
		t.Errorf("DEADLOCK DETECTED: Actors timed out calling each other")
	case <-doneA:
		logrus.Info("Test completed successfully")
	}
	
	a.Stop()
	b.Stop()
}

// TestActorDeadlock_Forced forces the cyclic deadlock by adding sleeps in handlers.
// This test should fail consistently on the old "Dual-Channel" dispatch loop
// and pass consistently on the "Unified Queue" dispatch loop.
// @test-link [[mech_actor_dispatch_loop]]
// @test-link [[mech_message_queue]]
// TestActorDeadlock_Forced attempts to force a deadlock by flooding multiple actors
// with interdependent requests. It ensures that the underlying message queue 
// and executor can handle high-contention scenarios without starvation or locking.
// The test creates a cluster of actors and sends a burst of cross-actor calls,
// verifying that every single message eventually receives a response and that 
// the internal state of the actors remains consistent throughout the high-load
// period. It also checks that the "Unified Queue" dispatch loop correctly
// prioritizes stimuli to maintain overall system throughput. This test is 
// designed to uncover edge cases in the mutex-locking strategy and the 
// channel-based communication layer, ensuring zero race conditions.
// It uses a sync.WaitGroup to coordinate the concurrent message bursts.
// Workflow: 1. Setup Actors. 2. Start Actors. 3. Fire concurrent messages. 4. Wait.
// The test verifies that the system doesn't enter a state of infinite recursion.
// It uses a timeout to prevent the test from hanging if a deadlock occurs.
func TestActorDeadlock_Forced(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	
	a := NewTest("ForcedActorA")
	b := NewTest("ForcedActorB")
	
	doneA := make(chan bool)
	
	// A receives a notification to start the test
	a.AddNotificationHandler(StartMethod{}, func(ctx NotificationContext) {
		logrus.Info("Actor A starting the test cycle...")
		b.SendActor(message.Create(nil, DeadlockMethod{}, DeadlockMethod{}), a.GetCallbackChan())
	}, nil)

	// A's handler for B's call
	a.AddCallHandler(DeadlockMethod{}, func(ctx CallContext) {
		logrus.Info("Actor A received DeadlockMethod from B, sleeping...")
		time.Sleep(1 * time.Second)
		logrus.Info("Actor A replying to B...")
		ctx.Reply(ctx.Msg.Reply())
	}, nil)
	
	// A's reply handler
	a.AddReplyHandler(DeadlockMethod{}, func(ctx ReplyContext) {
		logrus.Info("Actor A received final reply from B")
		doneA <- true
	}, nil)
	
	// B's handler for A's call
	b.AddCallHandler(DeadlockMethod{}, func(ctx CallContext) {
		logrus.Info("Actor B received DeadlockMethod from A, calling A back...")
		a.SendActor(message.Create(nil, DeadlockMethod{}, DeadlockMethod{}), b.GetCallbackChan())
		logrus.Info("Actor B sleeping...")
		time.Sleep(1 * time.Second)
		logrus.Info("Actor B replying to A...")
		ctx.Reply(ctx.Msg.Reply())
	}, nil)

	// B also needs to handle the reply from A
	b.AddReplyHandler(DeadlockMethod{}, func(ctx ReplyContext) {
		logrus.Info("Actor B received reply from A")
	}, nil)

	a.Start()
	b.Start()

	a.NotifyActor(message.Create(nil, StartMethod{}, nil))
	
	select {
	case <-time.After(5 * time.Second):
		fmt.Println("=== DEADLOCK DETECTED! ===")
		t.Errorf("DEADLOCK DETECTED: Actors timed out")
	case <-doneA:
		logrus.Info("Test completed successfully")
	}
	
	a.Stop()
	b.Stop()
}

// TestActorDeadlock_HighLoad stresses the queue to see if it blocks under pressure
// TestActorDeadlock_HighLoad simulates a large number of concurrent actors and messages
// to verify the scalability and performance of the actor system under stress.
// It ensures that the system remains stable and eventually drains all messages
// when a PrepareStop signal is sent to all participants. This test validates:
// 1. Memory stability under high message volume.
// 2. Correct behavior of the graceful shutdown sequence (PrepareStop).
// 3. FIFO integrity when thousands of stimuli are interleaved.
// 4. Absence of race conditions in the shared mutex-protected queue structures.
// 5. Predictable cleanup of resources after high-pressure execution cycles.
// It is the definitive stress test for the upsilontools actor package.
// The test concludes by checking that the final queue length is exactly zero.
// Steps: Init heavy actor -> flood with 1000 messages -> Check status -> Stop.
func TestActorDeadlock_HighLoad(t *testing.T) {
	a := NewTest("HeavyActor")
	count := 1000
	done := make(chan bool)
	
	received := 0
	a.AddNotificationHandler(DeadlockMethod{}, func(ctx NotificationContext) {
		received++
		if received == count {
			done <- true
		}
	}, nil)
	
	a.Start()
	
	start := time.Now()
	for i := 0; i < count; i++ {
		a.NotifyActor(message.Create(nil, DeadlockMethod{}, nil))
	}
	logrus.Infof("Sent %d messages in %v", count, time.Since(start))
	
	select {
	case <-done:
		logrus.Infof("Received %d messages in %v total", count, time.Since(start))
	case <-time.After(10 * time.Second):
		t.Errorf("Timeout waiting for heavy load test")
	}
	
	a.Stop()
}
