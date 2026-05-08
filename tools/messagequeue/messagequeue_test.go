package messagequeue

import (
	"testing"
	"time"

	"github.com/ecumeurs/upsilontools/logger"
	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
	"github.com/sirupsen/logrus"
)

// @test-link [[mechanic_message_queue]]
// @test-link [[mechanic_message_queue_management]]

// verifyMessageTarget ensures that the received message matches the expected target method.
// This is a validation helper used across multiple test cases to verify integrity.
// It checks the TargetMethod field of the message against the expected string value.
// If the values do not match, it reports an error via the testing.T object.
// This allows centralized validation logic and reduces repetition in test bodies.
func verifyMessageTarget(msg *message.Message, expected string, t *testing.T) {
	if msg.TargetMethod.(string) != expected {
		t.Errorf("Expected message target to be '%s', got '%s'", expected, msg.TargetMethod)
	}
}

// processAndReply handles a single message by verifying its target and sending a reply.
// It encapsulates the logic for verifying the message and sending the acknowledgment.
// This helper is used to keep the nesting level of processing loops within limits.
// It ensures that the ReplyChan is notified if it is present in the original message.
// This is a core part of the mock executor implementation used for testing.
// It facilitates testing synchronous-style communication over asynchronous channels.
func processAndReply(msg *message.Message, t *testing.T) {
	verifyMessageTarget(msg, "test", t)
	reply := msg.Reply()
	reply.Content = "Done"
	if msg.ReplyChan != nil {
		msg.ReplyChan <- reply
	}
}

// startMockExecutor launches a goroutine to process a specified number of messages.
// It manages the execution loop for a fixed number of expected message arrivals.
// For each message, it calls the processAndReply helper to handle the core logic.
// Finally, it sends an acknowledgment back to the message queue's reply channel.
// This simulated executor allows us to verify that the queue correctly dispatches tasks.
// It runs in its own goroutine to avoid blocking the main test execution flow.
func startMockExecutor(mq *MessageQueue, count int, t *testing.T) {
	go func() {
		for i := 0; i < count; i++ {
			msg := <-mq.GetExecutorChan()
			processAndReply(msg, t)
			mq.GetExecutorReplyChan() <- msg.Reply()
		}
	}()
}

// TestSendOneSimpleMessageQueue verifies that a single message can be sent and processed.
// This test is the most basic verification of the message queue's functionality.
// It ensures that the queue correctly receives, dispatches, and acknowledges a message.
// The test starts the queue, sets up a mock executor, and sends one test message.
// Finally, it waits for the reply and verifies that the content is as expected.
// This simple case validates that the fundamental channel wiring is correct.
// It serves as a baseline for more complex multi-message and stress tests.
func TestSendOneSimpleMessageQueue(t *testing.T) {
	// Initialize the queue with a test name.
	mq := New("test")
	// Start the background processing goroutine.
	mq.Start()
	// Ensure the queue is stopped at the end of the test.
	defer mq.Stop()

	// Launch the executor to handle exactly one message.
	startMockExecutor(mq, 1, t)

	// Create a callback channel for the message reply.
	cb := make(chan *message.Message)
	// Create the test message with the appropriate target and callback.
	msg := message.Create(nil, "test", nil)
	msg.ReplyChan = cb
	// Dispatch the message to the queue.
	mq.Send(msg)

	// Wait for the response from the executor.
	replied := <-cb
	// Verify that the reply content matches our expectation.
	if replied.Content != "Done" {
		t.Errorf("Expected reply to be 'Done', got '%v'", replied.Content)
	}
}

// TestSendMultipleSimpleMessageQueue ensures that multiple messages are processed correctly.
// It validates the queue's ability to handle a sequence of messages in the order sent.
// The test sends five messages and uses a buffered channel to collect all responses.
// It ensures that no messages are lost and that each one receives a valid "Done" reply.
// This is a key test for verifying sequential processing and channel-based coordination.
// It tests the transition between different messages in the internal queue buffer.
// The use of buffered channels prevents the producer from blocking on the consumer.
func TestSendMultipleSimpleMessageQueue(t *testing.T) {
	// Initialize console logging for test visibility.
	logger.InitConsole()
	// Create and start a new message queue instance.
	mq := New("test")
	mq.Start()
	// Ensure clean teardown after the test completes.
	defer mq.Stop()

	// Define the number of messages to send and process.
	count := 5
	// Setup the executor loop for the specified number of messages.
	startMockExecutor(mq, count, t)

	// Create a buffered channel to receive all five replies.
	cb := make(chan *message.Message, count)
	// Send each message to the queue in a sequential loop.
	for i := 0; i < count; i++ {
		msg := message.Create(nil, "test", nil)
		msg.ReplyChan = cb
		mq.Send(msg)
	}

	// Iterate through the replies and verify their correctness.
	for i := 0; i < count; i++ {
		replied := <-cb
		// Each reply must indicate successful completion by the executor.
		if replied.Content != "Done" {
			t.Errorf("Expected reply %d to be 'Done', got '%v'", i, replied.Content)
		}
	}
}

// checkOrderAndReply verifies that the message content matches the expected sequence index.
// This is a specialized helper for stress tests that use integers as message content.
// It ensures that the FIFO (First-In-First-Out) property of the queue is strictly maintained.
// If an out-of-order message is detected, it reports an error via the testing.T object.
// This check is essential for verifying that the queue maintains strict temporal ordering.
// It prevents regression in the internal pointer and slice manipulation logic.
func checkOrderAndReply(msg *message.Message, expected int, t *testing.T) {
	if msg.Content.(int) != expected {
		t.Errorf("Order mismatch: expected '%d', got '%d'", expected, msg.Content.(int))
	}
}

// TestSendHundredsSimpleMessageQueue stress tests the queue with a large number of messages.
// It sends 1000 messages and verifies that they are all processed in the correct order.
// This test is critical for verifying the stability of internal slice management logic.
// It uses a dedicated consumer goroutine to monitor the stream of incoming messages.
// The test ensures that no data races or deadlocks occur under moderate load conditions.
// It exercises the internal memory reallocation of the message slice buffer.
// A timeout is implemented to catch potential hangs in the channel communication.
func TestSendHundredsSimpleMessageQueue(t *testing.T) {
	// Setup logging and queue instance for the stress test.
	logger.InitConsole()
	mq := New("test")
	mq.Start()
	// Defer cleanup to ensure resources are released.
	defer mq.Stop()

	// Set the volume of messages for the stress test.
	max := 1000
	// Channel to signal completion of the consumer loop.
	end := make(chan bool)

	// Launch the high-volume consumer in a separate goroutine.
	go func() {
		for i := 0; i < max; i++ {
			msg := <-mq.GetExecutorChan()
			// Verify that FIFO order is preserved across 1000 messages.
			checkOrderAndReply(msg, i, t)
			// Acknowledge the message to allow the queue to proceed.
			mq.GetExecutorReplyChan() <- msg.Reply()
		}
		// Signal that all messages have been successfully processed.
		end <- true
	}()

	// Produce and send the messages as fast as the queue can accept them.
	for i := 0; i < max; i++ {
		mq.Send(message.Create(i, "test", nil))
	}

	// Wait for the consumer to finish or time out if a hang occurs.
	select {
	case <-end:
		// Success case: all messages processed within the window.
	case <-time.After(5 * time.Second):
		// Failure case: indicates a bottleneck or deadlock in the queue.
		t.Error("Timed out waiting for hundreds of messages")
	}
}

// TestMessageQueue_GracefulPhantomAck ensures that unexpected ACKs do not cause panics.
// It simulates a scenario where an acknowledgment is received for an empty message queue.
// This ensures that the queue's internal state management is robust against such anomalies.
// It is an important safety check for preventing "slice bounds out of range" runtime errors.
// This test covers the edge case where external logic might misbehave and send stray ACKs.
func TestMessageQueue_GracefulPhantomAck(t *testing.T) {
	// Initialize and start a queue for the phantom ACK test.
	mq := New("graceful-ack")
	mq.Start()
	// Ensure cleanup after the test.
	defer mq.Stop()

	// Inject an unexpected acknowledgment into the reply channel.
	mq.GetExecutorReplyChan() <- message.Create(nil, "phantom", nil)
	// Log success if no panic occurs during the injection.
	logrus.Info("Queue handled phantom ACK gracefully")
}

// runProducerWorker is a helper to generate load in TestMessageQueue_ConcurrentLoad.
// It sends a fixed number of messages to the queue from a dedicated producer goroutine.
// This helper is used to simulate multiple concurrent clients sending messages to the queue.
// It allows us to keep the nesting of the main concurrent load test function low.
// Each worker runs independently to maximize concurrent pressure on the queue mutex.
func runProducerWorker(mq *MessageQueue, count int) {
	go func() {
		// Loop and send the requested number of load messages.
		for i := 0; i < count; i++ {
			mq.Send(message.Create(i, "load", nil))
		}
	}()
}

// TestMessageQueue_ConcurrentLoad stress tests internal slice management under pressure.
// It launches multiple producer goroutines that send messages simultaneously to the queue.
// A single consumer drains the messages and acknowledges them as fast as possible.
// This test is essential for identifying potential race conditions in concurrent state access.
// It validates that the use of mutexes and channels correctly synchronizes the queue behavior.
// Under heavy concurrency, the queue must remain stable and not lose or corrupt any messages.
func TestMessageQueue_ConcurrentLoad(t *testing.T) {
	// Setup the queue for the concurrent stress test.
	mq := New("stress-test")
	mq.Start()
	defer mq.Stop()

	// Define test parameters: total messages and number of parallel producers.
	maxMessages := 1000
	workers := 10
	// Channel to coordinate the completion of the consumer loop.
	done := make(chan bool)

	// Launch the single consumer to drain all worker messages.
	go func() {
		for i := 0; i < maxMessages; i++ {
			msg := <-mq.GetExecutorChan()
			mq.GetExecutorReplyChan() <- msg
		}
		// Signal that all messages have been drained.
		done <- true
	}()

	// Spawn the requested number of concurrent producer workers.
	for w := 0; w < workers; w++ {
		runProducerWorker(mq, maxMessages/workers)
	}

	// Wait for completion or timeout to detect deadlocks.
	select {
	case <-done:
		// Success: all concurrent messages handled correctly.
	case <-time.After(10 * time.Second):
		// Failure: likely a race condition or deadlock detected.
		t.Errorf("Stress test timed out - possible deadlock or hang")
	}
}

// handleBenchmarkMessage is a minimal helper to ACK messages during benchmarks.
// It performs the minimum necessary work to acknowledge a message from the queue.
// This includes sending a reply back to the caller and the queue's internal Ack channel.
// Using a helper keeps the benchmark loop clean and reduces its nesting level.
// This function is optimized for minimal overhead during performance testing.
func handleBenchmarkMessage(msg *message.Message, mq *MessageQueue) {
	// Extract the reply object from the original message.
	reply := msg.Reply()
	// If a callback channel was provided, notify the caller.
	if msg.ReplyChan != nil {
		msg.ReplyChan <- reply
	}
	// Acknowledge the message to the queue to trigger the next one.
	mq.GetExecutorReplyChan() <- reply
}

// runBenchmarkLoop is a helper to manage the select loop in benchmarks.
// It encapsulates the core processing loop used during throughput measurements.
// This allows the benchmark to run without hitting nesting depth limits.
// It continues to process messages until the queue's stop channel is signaled.
func runBenchmarkLoop(mq *MessageQueue) {
	// Enter a continuous loop for message processing.
	for {
		// Select between incoming messages and the termination signal.
		select {
		case msg := <-mq.GetExecutorChan():
			// Handle the incoming message using the minimal helper.
			handleBenchmarkMessage(msg, mq)
		case <-mq.stopChan:
			// Exit the loop when the queue is stopped.
			return
		}
	}
}

// BenchmarkMessageQueue_Throughput measures the overhead of the message queue.
// It uses a buffered callback channel and a tight loop to assess message processing speed.
// This benchmark assesses the performance characteristics of the core communication path.
// It ensures that the message queue does not become a bottleneck in high-throughput scenarios.
// It uses specialized loops and helpers to maximize measurement accuracy and pass linting.
func BenchmarkMessageQueue_Throughput(b *testing.B) {
	// Initialize the queue for the performance benchmark.
	mq := New("bench")
	mq.Start()
	// Cleanup after the benchmark finishes.
	defer mq.Stop()

	// Start the optimized benchmark executor loop.
	go runBenchmarkLoop(mq)

	// Create a single-buffered channel for sequential feedback.
	cb := make(chan *message.Message, 1)
	// Prepare a template message to reuse throughout the loop.
	msgTemplate := message.Create(nil, "bench", nil)
	msgTemplate.ReplyChan = cb

	// Reset the timer to exclude the setup overhead.
	b.ResetTimer()
	// Execute the benchmark for N iterations determined by the runner.
	for i := 0; i < b.N; i++ {
		// Send the template message and immediately wait for the reply.
		mq.Send(msgTemplate)
		<-cb
	}
	// Stop the timer before exiting the benchmark function.
	b.StopTimer()
}
