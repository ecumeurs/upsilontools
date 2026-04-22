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
