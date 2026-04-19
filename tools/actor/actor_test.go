package actor

import (
	"errors"
	"testing"
	"time"

	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
	"github.com/sirupsen/logrus"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

type TestActor struct {
	Actor
	Counter int
	TA1     *TestActor
	TA2     *TestActor
}

func NewTest(name string) *TestActor {
	r := &TestActor{
		Actor:   *New(name),
		Counter: 0,
	}
	// We no longer arbitrarily map here, we set handlers in tests
	return r
}

// --- NEW V2 TESTS (ISS-004 / ISS-001) ---

type TestV2Notification struct{ ID int }
type TestV2Call struct{ ID int }
type TestV2Reply struct{ ID int }

// 1. One notification, then another
// @test-link [[mech_actor_pattern]]
func TestActorV2_Notifications(t *testing.T) {
	a := NewTest("NotifActor")
	done := make(chan bool)
	a.AddNotificationHandler(TestV2Notification{}, func(ctx NotificationContext) {
		msg := ctx.Msg.TargetMethod.(TestV2Notification)
		a.Counter += msg.ID
		if a.Counter == 3 {
			done <- true
		}
	}, nil)

	a.Start()
	a.NotifyActor(message.Create(nil, TestV2Notification{ID: 1}, nil))
	a.NotifyActor(message.Create(nil, TestV2Notification{ID: 2}, nil))
	<-done
	a.Stop()
}

// 2. One call, then another; expecting replies
func TestActorV2_Calls(t *testing.T) {
	a := NewTest("CallActor")
	a.AddCallHandler(TestV2Call{}, func(ctx CallContext) {
		msg := ctx.Msg.TargetMethod.(TestV2Call)
		a.Counter += msg.ID
		ctx.Reply(ctx.Msg.Reply())
	}, nil)

	a.Start()
	resChan := make(chan *message.Message)
	a.SendActor(message.Create(nil, TestV2Call{ID: 5}, TestV2Reply{}), resChan)
	<-resChan
	a.SendActor(message.Create(nil, TestV2Call{ID: 10}, TestV2Reply{}), resChan)
	<-resChan
	if a.Counter != 15 {
		t.Errorf("Expected 15, got %d", a.Counter)
	}
	a.Stop()
}

// 3. Expecting replies (using AddReplyHandler)
func TestActorV2_Replies(t *testing.T) {
	// A calls B. B replies to A. A processes reply.
	actorA := NewTest("ActorA")
	actorB := NewTest("ActorB")

	done := make(chan bool)

	actorB.AddCallHandler(TestV2Call{}, func(ctx CallContext) {
		ctx.Reply(ctx.Msg.Reply())
	}, nil)

	actorA.AddReplyHandler(TestV2Reply{}, func(ctx ReplyContext) {
		actorA.Counter++
		done <- true
	}, nil)

	// dummy method to kick off the call
	actorA.AddNotificationHandler(TestV2Notification{}, func(ctx NotificationContext) {
		actorB.SendActor(message.Create(nil, TestV2Call{ID: 1}, TestV2Reply{}), actorA.GetCallbackChan())
	}, nil)

	actorA.Start()
	actorB.Start()

	actorA.NotifyActor(message.Create(nil, TestV2Notification{}, nil))
	<-done

	if actorA.Counter != 1 {
		t.Errorf("A did not process reply")
	}

	actorA.Stop()
	actorB.Stop()
}

// 4. Context management example
// @test-link [[mech_actor_handler_context]]
func TestActorV2_ContextManagement(t *testing.T) {
	// A receives a Call. Stores ctx. Triggers Notification to itself. Notif handler unstashes and replies.
	a := NewTest("CtxActor")
	var savedCtx CallContext

	a.AddCallHandler(TestV2Call{}, func(ctx CallContext) {
		// Defer the reply and store it
		ctx.DeferReply()
		savedCtx = ctx
		// Notify self to finish it later
		a.NotifyActor(message.Create(nil, TestV2Notification{}, nil))
	}, nil)

	a.AddNotificationHandler(TestV2Notification{}, func(ctx NotificationContext) {
		a.Counter++
		// Use saved context to finish the call
		savedCtx.Reply(savedCtx.Msg.Reply())
	}, nil)

	a.Start()
	resChan := make(chan *message.Message)
	a.SendActor(message.Create(nil, TestV2Call{}, TestV2Reply{}), resChan)
	<-resChan // waiting for the deferred reply

	if a.Counter != 1 {
		t.Errorf("Counter not updated via deferred logic")
	}
	a.Stop()
}

// 5. Sync between multiple actors
// @test-link [[mech_actor_lifecycle]]
func TestActorV2_Sync(t *testing.T) {
	// Multiple actors syncing without blocking executors inline.
	syncActor := NewTest("SyncActor")
	w1 := NewTest("Worker1")
	w2 := NewTest("Worker2")

	w1Done := make(chan bool)
	w2Done := make(chan bool)

	w1.AddCallHandler(TestV2Call{}, func(ctx CallContext) { ctx.Reply(ctx.Msg.Reply()) }, nil)
	w2.AddCallHandler(TestV2Call{}, func(ctx CallContext) { ctx.Reply(ctx.Msg.Reply()) }, nil)

	syncActor.AddNotificationHandler(TestV2Notification{}, func(ctx NotificationContext) {
		// Ask w1 and w2
		w1.SendActor(message.Create(nil, TestV2Call{ID: 1}, TestV2Reply{}), syncActor.GetCallbackChan())
		w2.SendActor(message.Create(nil, TestV2Call{ID: 2}, TestV2Reply{}), syncActor.GetCallbackChan())
	}, nil)

	syncActor.AddReplyHandler(TestV2Reply{}, func(ctx ReplyContext) {
		syncActor.Counter++
		if syncActor.Counter == 1 {
			w1Done <- true
		} else if syncActor.Counter == 2 {
			w2Done <- true
		}
	}, nil)

	syncActor.Start()
	w1.Start()
	w2.Start()

	syncActor.NotifyActor(message.Create(nil, TestV2Notification{}, nil))
	<-w1Done
	<-w2Done

	syncActor.Stop()
	w1.Stop()
	w2.Stop()
}

// @test-link [[mech_actor_pattern]]
func TestActor_CallValidationHang(t *testing.T) {
	// Goal: Highlight the hang when a call validator fails.
	a := NewTest("ValidationActor")
	
	// Register a call handler with a failing validator
	a.AddCallHandler(TestV2Call{}, func(ctx CallContext) {
		t.Errorf("Handler should not be called if validator fails")
	}, func(msg *message.Message) []error {
		return []error{errors.New("validation failed")} // Return a real error
	})

	a.Start()
	
	resChan := make(chan *message.Message, 1) // buffered to avoid blocking
	a.SendActor(message.Create(nil, TestV2Call{}, TestV2Reply{}), resChan)
	
	// The first one should return an error
	select {
	case r := <-resChan:
		logrus.Infof("Msg 1: Received expected error: %s", r.ErrorMessage)
	case <-time.After(2 * time.Second):
		t.Errorf("Timeout waiting for validation error reply")
	}

	// Now send another message. If the queue is hanging, this will never be processed.
	done := make(chan bool)
	a.AddNotificationHandler(TestV2Notification{}, func(ctx NotificationContext) {
		done <- true
	}, nil)

	a.NotifyActor(message.Create(nil, TestV2Notification{}, nil))

	select {
	case <-done:
		logrus.Info("Queue still alive after validation failure (FIXED)")
	case <-time.After(2 * time.Second):
		t.Errorf("Queue is still HANGING after validation failure")
	}

	a.Stop()
}

func TestActor_UnhandledNotificationHang(t *testing.T) {
	// Goal: Highlight the hang when a notification is received but no handler exists.
	a := NewTest("UnhandledActor")
	a.CrashOnUnhandled = false // Don't panic, just watch it hang
	
	a.Start()
	
	// Send unhandled notification
	a.NotifyActor(message.Create(nil, struct{ Unhandled bool }{}, nil))
	
	// Now send a handled message. If the queue is hanging, this will never be processed.
	done := make(chan bool)
	a.AddNotificationHandler(TestV2Notification{}, func(ctx NotificationContext) {
		done <- true
	}, nil)

	a.NotifyActor(message.Create(nil, TestV2Notification{}, nil))

	select {
	case <-done:
		logrus.Info("Queue still alive after unhandled notification (FIXED)")
	case <-time.After(2 * time.Second):
		t.Errorf("Queue is still HANGING after unhandled notification")
	}

	a.Stop()
}

func BenchmarkActor_Notify(b *testing.B) {
	a := New("bench-notif")
	a.AddNotificationHandler(struct{}{}, func(ctx NotificationContext) {}, nil)
	a.Start()
	defer a.Stop()

	msg := message.Create(nil, struct{}{}, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.NotifyActor(msg)
	}
	b.StopTimer()
}

func BenchmarkActor_Call(b *testing.B) {
	a := New("bench-call")
	a.AddCallHandler(struct{}{}, func(ctx CallContext) {
		ctx.NoReply()
	}, nil)
	a.Start()
	defer a.Stop()

	cb := make(chan *message.Message, 1)
	msg := message.Create(nil, struct{}{}, struct{}{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.SendActor(msg, cb)
		<-cb
	}
	b.StopTimer()
}

// 6. Strict FIFO Ordering Verification
// @spec-link [[mech_actor_dispatch_loop]]
func TestActor_StrictFIFO(t *testing.T) {
	a := NewTest("FIFOActor")
	count := 500
	received := make([]int, 0, count)
	done := make(chan bool)

	a.AddNotificationHandler(TestV2Notification{}, func(ctx NotificationContext) {
		msg := ctx.Msg.TargetMethod.(TestV2Notification)
		received = append(received, msg.ID)
		if len(received) == count {
			done <- true
		}
	}, nil)

	a.Start()
	for i := 0; i < count; i++ {
		a.NotifyActor(message.Create(nil, TestV2Notification{ID: i}, nil))
	}

	select {
	case <-done:
		for i, id := range received {
			if id != i {
				t.Fatalf("FIFO Violation: Expected ID %d at index %d, got %d", i, i, id)
			}
		}
		logrus.Infof("Verified strict FIFO ordering for %d messages", count)
	case <-time.After(5 * time.Second):
		t.Errorf("Timeout waiting for FIFO test completion")
	}
	a.Stop()
}

// 7. Delayed Self-Notification Verification
// @spec-link [[mech_actor_pattern]]
func TestActor_SelfNotifyDelayed(t *testing.T) {
	a := NewTest("DelayedActor")
	received := make([]int, 0)
	done := make(chan bool)

	a.AddNotificationHandler(TestV2Notification{}, func(ctx NotificationContext) {
		msg := ctx.Msg.TargetMethod.(TestV2Notification)
		received = append(received, msg.ID)
		if msg.ID == 3 {
			done <- true
		}
	}, nil)

	a.Start()
	
	// 1. Send immediate notification
	a.NotifyActor(message.Create(nil, TestV2Notification{ID: 1}, nil))
	
	// 2. Schedule delayed notification (ID 3)
	// This should arrive after ID 2 because of the delay.
	a.SelfNotifyDelayed(TestV2Notification{ID: 3}, 100*time.Millisecond)
	
	// 3. Send another immediate notification (ID 2)
	a.NotifyActor(message.Create(nil, TestV2Notification{ID: 2}, nil))

	select {
	case <-done:
		if len(received) != 3 {
			t.Fatalf("Expected 3 messages, got %d", len(received))
		}
		// Order must be 1, 2, 3 because 3 was delayed while 1 and 2 were already in or entering the queue.
		if received[0] != 1 || received[1] != 2 || received[2] != 3 {
			t.Errorf("Ordering failure in delayed notification: %v", received)
		}
		logrus.Infof("Verified delayed self-notification ordering: %v", received)
	case <-time.After(1 * time.Second):
		t.Errorf("Timeout waiting for delayed notification")
	}
	a.Stop()
}

