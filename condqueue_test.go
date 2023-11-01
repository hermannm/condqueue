package condqueue_test

import (
	"context"
	"sync"
	"sync/atomic"
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

func TestSingleProducerMultipleConsumers(t *testing.T) {
	queue := condqueue.New[testMessage]()

	var errCount atomic.Int32
	ctx, ctxCleanup := context.WithTimeout(context.Background(), time.Second)

	var wg sync.WaitGroup
	wg.Add(len(testMessages) + 1)

	// Producer
	go func() {
		for _, message := range testMessages {
			queue.Add(message)
			t.Logf("[Producer] Added %+v", message)
		}
		wg.Done()
	}()

	// Consumers
	for i, message := range testMessages {
		i, message := i, message // Avoids mutating loop variable

		go func() {
			t.Logf("[Consumer %d] Waiting for %+v", i, message)
			receivedMessage, err := queue.AwaitMatchingItem(ctx, func(candidate testMessage) bool {
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
	queue := condqueue.New[testMessage]()

	var errCount atomic.Int32
	ctx, ctxCleanup := context.WithTimeout(context.Background(), time.Second)

	var wg sync.WaitGroup
	wg.Add(len(testMessages) * 2)

	// Producers
	for i, message := range testMessages {
		i, message := i, message // Avoids mutating loop variable

		go func() {
			queue.Add(message)
			t.Logf("[Producer %d] Added %+v", i, message)
			wg.Done()
		}()
	}

	// Consumers
	for i, message := range testMessages {
		i, message := i, message // Avoids mutating loop variable

		go func() {
			t.Logf("[Consumer %d] Waiting for %+v", i, message)
			receivedMessage, err := queue.AwaitMatchingItem(ctx, func(candidate testMessage) bool {
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
	queue := condqueue.New[testMessage]()

	ctx, cleanup := context.WithTimeout(context.Background(), 10*time.Millisecond)

	_, err := queue.AwaitMatchingItem(ctx, func(testMessage) bool {
		return true
	})
	cleanup()
	if err == nil {
		t.Fatal("expected timeout error from AwaitMatchingItem")
	}
}

func TestClear(t *testing.T) {
	queue := condqueue.New[testMessage]()

	const msgType = "success"

	queue.Add(testMessage{Type: msgType})
	queue.Clear()

	ctx, cleanup := context.WithTimeout(context.Background(), 100*time.Millisecond)
	_, err := queue.AwaitMatchingItem(ctx, func(candidate testMessage) bool {
		return candidate.Type == msgType
	})
	cleanup()
	if err == nil {
		t.Fatal("expected timeout error from Clear")
	}
}
