package actor

import (
	"fmt"
	"testing"

	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
	"github.com/sirupsen/logrus"
)

type TestActor struct {
	Actor
}
type TestActorRequest struct{}
type TestActorReply struct{}
type TestActorReplyRequest struct{}
type TestActorRequestError struct{}
type TestActorRequestWithParam struct {
	Param string
}

// NewTest
func NewTest(name string) *TestActor {
	r := &TestActor{
		Actor: *New(name),
	}
	r.AddMethod(TestActorRequest{}, r.testActorRequest, nil)
	r.AddMethod(TestActorRequestError{}, r.testActorRequestError, nil)
	r.AddMethod(TestActorRequestWithParam{}, r.testActorRequestWithParam, nil)

	return r
}

func NewReplierTest(name string) *TestActor {
	r := &TestActor{
		Actor: *New(name),
	}
	r.AddMethod(TestActorReplyRequest{}, r.testActorReplier, nil)

	return r
}

func (a *TestActor) testActorRequest(msg *message.Message) bool {
	fmt.Println("testActorRequest")
	a.Reply(msg, msg.Reply())
	return true
}

func (a *TestActor) testActorRequestError(msg *message.Message) bool {
	fmt.Println("testActorRequestError")
	a.Reply(msg, msg.ReplyWithError("some error", "some error key"))
	return true
}

func (a *TestActor) testActorReplier(msg *message.Message) bool {
	a.Logger.Info("testActorReplier Received message: ", msg.String())
	a.Reply(msg, msg.Reply())
	return true
}

func (a *TestActor) testActorRequestWithParam(msg *message.Message) bool {
	fmt.Println("testActorRequestWithParam")
	return true
}

func TestActorSendMessage(t *testing.T) {
	testActor := NewTest("test")

	testActor.Start()

	resChan := make(chan *message.Message)
	defer close(resChan)
	req := message.New()
	req.TargetMethod = TestActorRequest{}

	testActor.SendActor(req, resChan)

	<-resChan

	testActor.Stop()
}

func TestActorToActorMessaging(t *testing.T) {

	testActor := NewTest("test")
	testActor.Start()

	testActor2 := NewReplierTest("test2")
	testActor2.Start()

	testActor.AddMethod(TestActorReplyRequest{}, func(msg *message.Message) (handled bool) {
		testActor.Logger.Info("testActor received message: TestActorReplyRequest")
		testActor2.SendActor(message.Create(nil, TestActorReplyRequest{}, TestActorReply{}), testActor.GetCallbackChan())
		testActor.Reply(msg, msg.Reply())
		return true
	}, nil)

	replyChan := make(chan *message.Message)
	defer close(replyChan)

	testActor.AddReply(TestActorReply{}, func(msg *message.Message) (handled bool) {
		testActor.Logger.Info("testActor received reply: TestActorReply")
		replyChan <- msg
		// replies don't necessitate a Reply()/NoReply() call
		return true
	}, nil)

	resChan := make(chan *message.Message)
	defer close(resChan)
	req := message.New()
	req.TargetMethod = TestActorReplyRequest{}

	testActor.SendActor(req, resChan)

	logrus.Info("Waiting for request to be received")
	<-resChan
	logrus.Info("Waiting for reply")
	<-replyChan
	logrus.Info("Reply received")

	testActor.Stop()
	testActor2.Stop()
}

func TestBlockingActorToActorMessaging(t *testing.T) {

	testActor := NewTest("test")
	testActor.Start()

	testActor2 := NewReplierTest("test2")
	testActor2.Start()

	testActor.AddMethod(TestActorReplyRequest{}, func(msg *message.Message) (handled bool) {
		testActor.Logger.Info("testActor received message: TestActorReplyRequest")

		localReplyChan := make(chan *message.Message)
		defer close(localReplyChan)

		testActor2.SendActor(message.Create(nil, TestActorReplyRequest{}, TestActorReply{}), localReplyChan)

		testActor.Logger.Info("Waiting for reply")
		<-localReplyChan

		// won't proc a Reply slot in the actor as the reply is handled by the callback `localReplyChan`
		// this method allows for a multistep action to occurs in a single method, but is blocking the whole actor during this step
		// (the actor will still accepts new message as it's handled by the queue thread)
		testActor.Logger.Info("Reply received")

		testActor.Reply(msg, msg.Reply())
		return true
	}, nil)

	resChan := make(chan *message.Message)
	defer close(resChan)
	req := message.New()
	req.TargetMethod = TestActorReplyRequest{}

	testActor.SendActor(req, resChan)

	logrus.Info("Waiting for reply to be received")
	<-resChan
	logrus.Info("Request fully processed")

	testActor.Stop()
	testActor2.Stop()
}
