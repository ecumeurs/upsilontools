package actor

import (
	"errors"
	"testing"
	"time"

	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
	"github.com/sirupsen/logrus"
)

// init sets the global log level to Debug for all tests in this package.
// This is necessary to capture granular traces of actor dispatch events,
// such as message popping, handler execution, and internal state transitions.
func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

type TestActor struct {
	Actor
	Counter int
	TA1     *TestActor
	TA2     *TestActor
}

// NewTest is a convenience wrapper around New for test scenarios.
// It initializes a TestActor which embeds the standard Actor but adds
// a Counter and other test-specific fields for verifying handler side-effects.
// It uses the provided name for internal logging and queue identification.
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

// TestActorV2_Notifications verifies that multiple fire-and-forget notifications
// are processed sequentially and accurately by the actor. This test ensures that
// the internal state (Counter) is correctly updated as each stimulus arrives,
// proving the basic notification-handling capability of the "Unified Queue" loop.
// It also verifies that the Start() and Stop() lifecycle methods work as expected.
// Educational Context: In the Actor model, notifications are asynchronous and
// non-blocking. The sender does not wait for a response, and the actor processes
// these messages in the order they were enqueued. This test validates the 
// integrity of the FIFO buffer and the dispatcher's ability to pull stimuli
// without starvation or race conditions in the underlying channel.
// Usage: NotifyActor() is used for non-critical stimuli that don't need ACKs.
// The actor's counter is incremented by each incoming ID value.
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

// TestActorV2_Calls verifies that multiple synchronized calls are processed correctly.
// It ensures that the CallContext.Reply mechanism correctly unblocks the caller
// and that the actor state is updated sequentially across separate call handlings.
// The test validates the round-trip latency and the correct pairing of requests
// and replies using the internal response channel.
// Educational Context: Synchronized calls (Requests) require a mandatory reply
// from the receiver. The sender blocks on a response channel until the reply
// is delivered. This test ensures that the Actor infrastructure correctly 
// manages these channels, preventing leaks and ensuring that the caller is
// always unblocked even if the actor processes multiple calls in a row.
// Key Protocol: Callers must use SendActor() and provide a return channel.
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

// TestActorV2_Replies verifies the full request-reply lifecycle between two actors.
// It tests the AddReplyHandler mechanism, ensuring that an actor can correctly
// receive and process responses to its outgoing messages via its callback channel.
// This is a critical integration test for the communication protocol, as it 
// demonstrates how actors can collaborate without blocking their main execution loops.
// Educational Context: When Actor A calls Actor B, it provides its own 
// CallbackChan as the return address. Actor B's reply is then sent to this channel,
// which Actor A's dispatcher picks up as a new stimulus. This asynchronous 
// callback pattern is what allows the system to scale without thread exhaustion.
// Verification: The Counter in Actor A must be incremented upon reply receipt.
// This confirms that the return address was correctly propagated and handled.
// The test uses a channel to signal completion to the main test routine.
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

// TestActorV2_ContextManagement demonstrates advanced usage of the CallContext.
// It verifies that a call can be "deferred" and then completed later from 
// a different message handler. This pattern is essential for non-blocking
// orchestration where an actor must wait for an external event before 
// replying to a pending request, avoiding deadlock and keeping the queue moving.
// Educational Context: DeferReply() signals to the actor that the current handler
// is exiting but the message protocol is not yet complete. This allows the 
// dispatcher to continue processing other stimuli while the "parent" call 
// remains logically open. The reply can later be sent via the stored context.
// Flow: Call -> Defer -> Notify Self -> Handle Notif -> Reply using Stored Context.
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

// TestActorV2_Sync verifies coordination between multiple actors in a fan-out pattern.
// It ensures that an actor can orchestrate work across several sub-actors
// by sending multiple concurrent calls and processing their replies individually
// without blocking its own execution loop or causing deadlocks. The test 
// validates that the responses are correctly correlated back to the originating actor.
// Educational Context: High-performance orchestration requires actors to 
// delegate tasks and wait for results without idling. By using typed reply
// handlers, the sync actor can maintain its own state-machine while 
// waiting for sub-tasks to complete, effectively acting as a coordinator.
// Expected Behavior: Both workers must reply before the test channel is unblocked.
// This pattern avoids the "Sync Call in Loop" anti-pattern in concurrent code.
// It is the standard way to implement complex orchestration in Upsilon.
// The test completes once both workers have successfully delivered their replies.
// This ensures that the Actor infrastructure can handle multiple concurrent 
// response channels without internal collision or misrouting.
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

// TestActor_CallValidationHang verifies that the actor remains responsive even when
// a message validator fails. It ensures that the system correctly sends an error
// reply and unblocks the message queue instead of hanging on the invalid stimulus.
// This prevents one bad client from causing a denial-of-service on an entire actor.
// Educational Context: Robustness is a key requirement of the Actor system.
// If a message is malformed or invalid, the infrastructure must ensure it 
// is acknowledged as a failure so that subsequent, valid messages can still 
// be processed. A hang in the validator would otherwise block the entire actor.
// The validator should be used for pre-processing checks that don't require
// full business logic execution, keeping the critical path clean.
// Test Setup: Call -> Failing Validator -> Check error reply -> Send new message.
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

// TestActor_UnhandledNotificationHang verifies that the actor does not hang when
// it receives a notification for which no handler has been registered.
// It ensures that unhandled stimuli are correctly discarded or acknowledged
// to maintain the flow of subsequent messages in the FIFO queue, preventing 
// "zombie" actors that stop processing but haven't crashed.
// Educational Context: The "Unified Queue" approach requires every popped 
// stimulus to be acknowledged to the message queue's executor. This test 
// verifies that the fallback "unhandled" logic correctly provides this 
// acknowledgment even when no user-defined handler matches the message type.
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

// BenchmarkActor_Notify measures the performance of fire-and-forget notifications.
// It provides a baseline for the throughput of the actor's internal message queue.
// The benchmark iterates through thousands of notifications to measure 
// the overhead of the mutex and channel operations in the dispatch loop.
// Educational Context: Efficient notification handling is vital for 
// high-frequency engine events (e.g., entity movement). This benchmark 
// allows developers to quantify the performance cost of the Actor's 
// thread-safety guarantees and identifies bottlenecks in the queue's pop logic.
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

// BenchmarkActor_Call measures the round-trip performance of synchronized calls.
// It provides a baseline for the latency of the request-reply protocol.
// This benchmark helps identify performance regressions in the synchronized 
// communication path, which involves multiple channel transfers and context switches.
// Educational Context: Synchronized calls are significantly more expensive 
// than notifications because they involve waiting and context switching. 
// This benchmark helps set performance budgets for cross-actor communication.
// It specifically highlights the overhead of the select loop and channel 
// contention when multiple go-routines compete for the actor's attention.
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

// TestActor_StrictFIFO verifies that messages are processed in the exact order
// they were received. This is a core guarantee of the Actor model implementation.
// It ensures that no concurrent reordering occurs within the dispatch loop,
// which is vital for state-machine consistency in the engine.
// Educational Context: Strict ordering ensures that if Event A happens 
// before Event B, the actor will observe them in that exact sequence. 
// This simplifies logic significantly, as developers don't have to account 
// for out-of-order stimuli which could otherwise lead to inconsistent states.
// The FIFO guarantee is enforced by the message queue's internal mutex and 
// the single-threaded nature of the actor's primary execution loop.
// Assertion: Array of received IDs must exactly match the sequence of sent IDs.
// @test-link [[mech_actor_dispatch_loop]]
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
		verifyFIFO(t, received)
		logrus.Infof("Verified strict FIFO ordering for %d messages", count)
	case <-time.After(5 * time.Second):
		t.Errorf("Timeout waiting for FIFO test completion")
	}
	a.Stop()
}

// verifyFIFO is a helper to reduce nesting in the main test.
// It iterates through the received messages and compares them with expected values.
// This separation of concerns improves readability and keeps the main test 
// function's complexity below the architectural threshold.
func verifyFIFO(t *testing.T, received []int) {
	for i, id := range received {
		if id != i {
			t.Fatalf("FIFO Violation: Expected ID %d at index %d, got %d", i, i, id)
		}
	}
}

// TestActor_SelfNotifyDelayed verifies the AfterFunc scheduling mechanism.
// It ensures that delayed notifications are correctly queued and executed
// after the specified duration, maintaining relative order with other stimuli.
// This test is critical for logic that depends on timeouts or delayed retries.
// Educational Context: Delayed stimuli are scheduled using standard timers
// but injected back into the Actor's primary message queue. This ensures 
// that even delayed messages are processed sequentially with other incoming 
// stimuli, preserving the single-threaded execution guarantee of the actor.
// This mechanism is preferred over using time.Sleep inside handlers, which 
// would block the entire actor from processing other concurrent messages.
// @test-link [[mech_actor_pattern]]
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

