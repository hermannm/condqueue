package condqueue_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hermannm.dev/condqueue"
)

// Interface to test both old CondQueue and new CondQueue2
type CondQueue[T any] interface {
	AddItem(item T)

	AwaitMatchingItem(
		ctx context.Context,
		isMatch func(item T) bool,
	) (matchingItem T, cancelErr error)

	Clear()
}

type TestMessage struct {
	Type string
}

var testMessages = []TestMessage{
	{Type: "success"},
	{Type: "error"},
	{Type: "timeout"},
	{Type: "error"},
	{Type: "success"},
	{Type: "timeout"},
	{Type: "timeout"},
	{Type: "error"},
}

func TestSingleProducerMultipleConsumers(t *testing.T) {
	queue := condqueue.New[TestMessage]()
	testSingleProducerMultipleConsumers(t, queue)
}

func TestCondQueue2SingleProducerMultipleConsumers(t *testing.T) {
	queue := condqueue.New2[TestMessage]()
	testSingleProducerMultipleConsumers(t, queue)
}

func testSingleProducerMultipleConsumers(t *testing.T, queue CondQueue[TestMessage]) {
	t.Helper()

	var errCount atomic.Int32
	ctx, ctxCleanup := context.WithTimeout(context.Background(), time.Second)

	var wg sync.WaitGroup
	wg.Add(len(testMessages) + 1)

	go func() {
		for _, message := range testMessages {
			queue.AddItem(message)
			t.Logf("[Producer] Added %+v", message)
		}
		wg.Done()
	}()

	for i, message := range testMessages {
		i, message := i, message // Avoids mutating loop variable

		go func() {
			t.Logf("[Consumer %d] Waiting for %+v", i, message)
			receivedMessage, err := queue.AwaitMatchingItem(ctx, func(candidate TestMessage) bool {
				return candidate.Type == message.Type
			})
			if err == nil {
				t.Logf("[Consumer %d] Received %+v", i, receivedMessage)
			} else {
				t.Logf("[Consumer %d] Received error: %v", i, err)
				errCount.Add(1)
			}
			wg.Done()
		}()
	}

	wg.Wait()
	ctxCleanup()

	if errCount.Load() != 0 {
		t.Fatal("non-zero error count (run tests with -v to see errors)")
	}
}

func TestMultipleProducersMultipleConsumers(t *testing.T) {
	queue := condqueue.New[TestMessage]()
	testMultipleProducersMultipleConsumers(t, queue)
}

func TestCondQueue2MultipleProducersMultipleConsumers(t *testing.T) {
	queue := condqueue.New2[TestMessage]()
	testMultipleProducersMultipleConsumers(t, queue)
}

func testMultipleProducersMultipleConsumers(t *testing.T, queue CondQueue[TestMessage]) {
	t.Helper()

	var errCount atomic.Int32
	ctx, ctxCleanup := context.WithTimeout(context.Background(), time.Second)

	var wg sync.WaitGroup
	wg.Add(len(testMessages) * 2)

	for i, message := range testMessages {
		i, message := i, message // Avoids mutating loop variable

		go func() {
			queue.AddItem(message)
			t.Logf("[Producer %d] Added %+v", i, message)
			wg.Done()
		}()
	}

	for i, message := range testMessages {
		i, message := i, message // Avoids mutating loop variable

		go func() {
			t.Logf("[Consumer %d] Waiting for %+v", i, message)
			receivedMessage, err := queue.AwaitMatchingItem(ctx, func(candidate TestMessage) bool {
				return candidate.Type == message.Type
			})
			if err == nil {
				t.Logf("[Consumer %d] Received %+v", i, receivedMessage)
			} else {
				t.Logf("[Consumer %d] Received error: %v", i, err)
				errCount.Add(1)
			}
			wg.Done()
		}()
	}

	wg.Wait()
	ctxCleanup()

	if errCount.Load() != 0 {
		t.Fatal("non-zero error count (run tests with -v to see errors)")
	}
}

func TestTimeout(t *testing.T) {
	queue := condqueue.New[TestMessage]()
	testTimeout(t, queue)
}

func TestCondQueue2Timeout(t *testing.T) {
	queue := condqueue.New2[TestMessage]()
	testTimeout(t, queue)
}

func testTimeout(t *testing.T, queue CondQueue[TestMessage]) {
	t.Helper()

	ctx, cleanup := context.WithTimeout(context.Background(), 10*time.Millisecond)

	_, err := queue.AwaitMatchingItem(ctx, func(TestMessage) bool {
		return true
	})
	cleanup()
	if err == nil {
		t.Fatal("expected timeout error from AwaitMatchingItem")
	}
}

func TestClear(t *testing.T) {
	queue := condqueue.New[TestMessage]()
	testClear(t, queue)
}

func TestCondQueue2Clear(t *testing.T) {
	queue := condqueue.New2[TestMessage]()
	testClear(t, queue)
}

func testClear(t *testing.T, queue CondQueue[TestMessage]) {
	t.Helper()

	const msgType = "success"

	queue.AddItem(TestMessage{Type: msgType})
	queue.Clear()

	ctx, cleanup := context.WithTimeout(context.Background(), 100*time.Millisecond)
	_, err := queue.AwaitMatchingItem(ctx, func(candidate TestMessage) bool {
		return candidate.Type == msgType
	})
	cleanup()
	if err == nil {
		t.Fatal("expected timeout error from Clear")
	}
}
