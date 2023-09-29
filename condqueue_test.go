package condqueue_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hermannm.dev/condqueue"
)

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

	var errCount atomic.Int32
	ctx, ctxCleanup := context.WithTimeout(context.Background(), time.Second)

	var wg sync.WaitGroup
	wg.Add(len(testMessages) + 1)

	go func() {
		for _, message := range testMessages {
			err := queue.AddItem(ctx, message)
			if err == nil {
				t.Logf("[Producer] Added %+v", message)
			} else {
				t.Logf("[Producer] AddItem error: %v", err)
				errCount.Add(1)
			}
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

	var errCount atomic.Int32
	ctx, ctxCleanup := context.WithTimeout(context.Background(), time.Second)

	var wg sync.WaitGroup
	wg.Add(len(testMessages) * 2)

	for i, message := range testMessages {
		i, message := i, message // Avoids mutating loop variable

		go func() {
			err := queue.AddItem(ctx, message)
			if err == nil {
				t.Logf("[Producer %d] Added %+v", i, message)
			} else {
				t.Logf("[Producer %d] AddItem error: %v", i, err)
				errCount.Add(1)
			}
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

func TestClear(t *testing.T) {
	queue := condqueue.New[TestMessage]()

	const msgType = "success"

	ctx := context.Background()
	if err := queue.AddItem(ctx, TestMessage{Type: msgType}); err != nil {
		t.Fatalf("unexpected AddItem error: %v", err)
	}

	if err := queue.Clear(ctx); err != nil {
		t.Fatalf("unexpected Clear error: %v", err)
	}

	ctx, cleanup := context.WithTimeout(ctx, 100*time.Millisecond)
	item, err := queue.AwaitMatchingItem(ctx, func(candidate TestMessage) bool {
		return candidate.Type == msgType
	})
	if err == nil {
		t.Fatalf("expected Clear to error with timeout, but got item instead: %+v", item)
	}

	cleanup()
}
