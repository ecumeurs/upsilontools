package actor

import (
	"fmt"
	"testing"

	"github.com/ecumeurs/upsilontools/tools/messagequeue/message"
)

type TestActor struct {
	Actor
}
type TestActorRequest struct{}
type TestActorRequestError struct{}
type TestActorRequestWithParam struct {
	Param string
}

// NewTest
func NewTest(name string) *TestActor {
	r := &TestActor{
		Actor: *New(name),
	}
	r.receiveMessageHandler = r.ReceiveMessage
	return r
}

func (a *TestActor) testActorRequest(msg message.Message) {
	fmt.Println("testActorRequest")
	a.Reply(msg.Reply())
}

func (a *TestActor) testActorRequestError(msg message.Message) {
	fmt.Println("testActorRequestError")
	a.Reply(msg.ReplyWithError("some error", "some error key"))
}

func (a *TestActor) testActorRequestWithParam(msg message.Message) {
	fmt.Println("testActorRequestWithParam")
}

func (a *TestActor) ReceiveMessage(msg message.Message) bool {
	switch msg.TargetMethod.(type) {
	case TestActorRequest:
		a.testActorRequest(msg)
		return true
	case TestActorRequestError:
		a.testActorRequestError(msg)
		return true
	case TestActorRequestWithParam:
		a.testActorRequestWithParam(msg)
		return true
	}
	return false
}

func TestActorSendMessage(t *testing.T) {
	testActor := NewTest("test")

	testActor.Start()

	resChan := make(chan message.Message)
	defer close(resChan)
	req := message.New()
	req.TargetMethod = TestActorRequest{}

	testActor.SendMessage(req, resChan)

	<-resChan

	testActor.Stop()
}
