package condqueue_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"hermannm.dev/condqueue"
)

type testMessage struct {
	Type string
}

var testMessages = []testMessage{
	{Type: "success"},
	{Type: "error"},
	{Type: "timeout"},
	{Type: "error"},
	{Type: "success"},
	{Type: "timeout"},
	{Type: "timeout"},
	{Type: "error"},
}

func TestSingleProducerSingleConsumer(t *testing.T) {
	queue := condqueue.New[testMessage]()

	ctx, cleanup := context.WithTimeout(context.Background(), time.Second)
	defer cleanup()

	var wg sync.WaitGroup
	wg.Add(2)

	expectedMsg := testMessage{Type: "test"}

	// Producer
	go func() {
		t.Logf("[producer] adding %v", expectedMsg)
		queue.Add(expectedMsg)
		wg.Done()
	}()

	// Consumer
	go func() {
		t.Logf("[consumer] waiting for %v", expectedMsg)

		msg, err := queue.AwaitMatchingItem(
			ctx, func(candidate testMessage) bool {
				return candidate.Type == expectedMsg.Type
			},
		)

		if err != nil {
			t.Errorf("[consumer] [ERROR] timed out waiting for %v", expectedMsg)
		} else if msg.Type != expectedMsg.Type {
			t.Errorf("[consumer] [ERROR] expected %v, got %v", expectedMsg, msg)
		} else {
			t.Logf("[consumer] received %v", msg)
		}

		wg.Done()
	}()

	wg.Wait()
}

func TestSingleProducerMultipleConsumers(t *testing.T) {
	queue := condqueue.New[testMessage]()

	ctx, cleanup := context.WithTimeout(context.Background(), time.Second)
	defer cleanup()

	var wg sync.WaitGroup
	wg.Add(len(testMessages) + 1)

	// Producer
	go func() {
		for _, msg := range testMessages {
			t.Logf("[producer] adding %v", msg)
			queue.Add(msg)
		}

		wg.Done()
	}()

	// Consumers
	for i, expectedMsg := range testMessages {
		go func() {
			t.Logf("[consumer %d] waiting for %v", i, expectedMsg)

			msg, err := queue.AwaitMatchingItem(
				ctx, func(candidate testMessage) bool {
					return candidate.Type == expectedMsg.Type
				},
			)

			if err != nil {
				t.Errorf("[consumer %d] [ERROR] timed out waiting for %v", i, expectedMsg)
			} else if msg.Type != expectedMsg.Type {
				t.Errorf("[consumer %d] [ERROR] expected %v, got %v", i, expectedMsg, msg)
			} else {
				t.Logf("[consumer %d] received %v", i, msg)
			}

			wg.Done()
		}()
	}

	wg.Wait()
}

func TestMultipleProducersMultipleConsumers(t *testing.T) {
	queue := condqueue.New[testMessage]()

	ctx, cleanup := context.WithTimeout(context.Background(), time.Second)
	defer cleanup()

	var wg sync.WaitGroup
	wg.Add(len(testMessages) * 2)

	// Producers
	for i, msg := range testMessages {
		go func() {
			t.Logf("[producer %d] adding %v", i, msg)
			queue.Add(msg)
			wg.Done()
		}()
	}

	// Consumers
	for i, expectedMsg := range testMessages {
		go func() {
			t.Logf("[consumer %d] waiting for %v", i, expectedMsg)

			msg, err := queue.AwaitMatchingItem(
				ctx, func(candidate testMessage) bool {
					return candidate.Type == expectedMsg.Type
				},
			)

			if err != nil {
				t.Errorf("[consumer %d] [ERROR] timed out waiting for %v", i, expectedMsg)
			} else if msg.Type != expectedMsg.Type {
				t.Errorf("[consumer %d] [ERROR] expected %v, got %v", i, expectedMsg, msg)
			} else {
				t.Logf("[consumer %d] received %v", i, msg)
			}

			wg.Done()
		}()
	}

	wg.Wait()
}

func TestTimeout(t *testing.T) {
	queue := condqueue.New[testMessage]()

	ctx, cleanup := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cleanup()

	_, err := queue.AwaitMatchingItem(
		ctx, func(testMessage) bool {
			return true
		},
	)

	if err == nil {
		t.Error("expected timeout error from AwaitMatchingItem")
	}
}

func TestCancel(t *testing.T) {
	queue := condqueue.New[testMessage]()

	ctx, cancel := context.WithCancelCause(context.Background())
	cancelErr := errors.New("something went wrong")
	cancel(cancelErr)

	_, err := queue.AwaitMatchingItem(
		ctx, func(testMessage) bool {
			return true
		},
	)

	if err == nil {
		t.Fatal("expected error from AwaitMatchingItem with canceled context")
	}

	//nolint:errorlint  // We want to test that the error is returned unwrapped
	if err != cancelErr {
		t.Fatalf("expected error to match the one given to cancel function, but got: %v", err)
	}
}

func TestClear(t *testing.T) {
	queue := condqueue.New[testMessage]()

	const msgType = "test"

	queue.Add(testMessage{Type: msgType})
	queue.Clear()

	ctx, cleanup := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cleanup()

	_, err := queue.AwaitMatchingItem(
		ctx, func(candidate testMessage) bool {
			return candidate.Type == msgType
		},
	)

	if err == nil {
		t.Error("expected timeout error from AwaitMatchingItem")
	}
}

func (msg testMessage) String() string {
	return fmt.Sprintf("'%s' message", msg.Type)
}
