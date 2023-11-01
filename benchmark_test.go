package condqueue_test

import (
	"context"
	"math/rand"
	"sync"
	"testing"

	"hermannm.dev/condqueue"
)

func BenchmarkSingleProducerMultipleConsumers(b *testing.B) {
	benchmarkSingleProducerMultipleConsumers(b, func() CondQueue[TestMessage] {
		return condqueue.New[TestMessage]()
	})
}

func BenchmarkCondQueue2SingleProducerMultipleConsumers(b *testing.B) {
	benchmarkSingleProducerMultipleConsumers(b, func() CondQueue[TestMessage] {
		return condqueue.New2[TestMessage]()
	})
}

func benchmarkSingleProducerMultipleConsumers(
	b *testing.B,
	queueConstructor func() CondQueue[TestMessage],
) {
	b.Helper()
	b.StopTimer()

	for n := 0; n < b.N; n++ {
		queue := queueConstructor()
		testMessages := createTestMessages(100)

		var wg sync.WaitGroup
		wg.Add(len(testMessages) + 1)

		b.StartTimer()

		// Consumers
		for _, message := range testMessages {
			message := message // Avoids mutating loop variable

			go func() {
				queue.AwaitMatchingItem(context.Background(), func(candidate TestMessage) bool {
					return candidate.Type == message.Type
				})

				wg.Done()
			}()
		}

		// Producer
		go func() {
			for _, message := range testMessages {
				queue.AddItem(message)
			}
			wg.Done()
		}()

		wg.Wait()

		b.StopTimer()
	}
}

func BenchmarkMultipleProducersMultipleConsumers(b *testing.B) {
	benchmarkMultipleProducersMultipleConsumers(b, func() CondQueue[TestMessage] {
		return condqueue.New[TestMessage]()
	})
}

func BenchmarkCondQueue2MultipleProducersMultipleConsumers(b *testing.B) {
	benchmarkMultipleProducersMultipleConsumers(b, func() CondQueue[TestMessage] {
		return condqueue.New2[TestMessage]()
	})
}

func benchmarkMultipleProducersMultipleConsumers(
	b *testing.B,
	queueConstructor func() CondQueue[TestMessage],
) {
	b.Helper()
	b.StopTimer()

	for n := 0; n < b.N; n++ {
		queue := queueConstructor()
		testMessages := createTestMessages(100)

		var wg sync.WaitGroup
		wg.Add(len(testMessages) * 2)

		b.StartTimer()

		// Consumers
		for _, message := range testMessages {
			message := message // Avoids mutating loop variable

			go func() {
				queue.AwaitMatchingItem(context.Background(), func(candidate TestMessage) bool {
					return candidate.Type == message.Type
				})

				wg.Done()
			}()
		}

		// Producers
		for _, message := range testMessages {
			message := message // Avoids mutating loop variable

			go func() {
				queue.AddItem(message)
				wg.Done()
			}()
		}

		wg.Wait()

		b.StopTimer()
	}
}

func createTestMessages(count int) []TestMessage {
	messages := make([]TestMessage, count)
	messageTypes := []string{"success", "error", "timeout"}

	for i := range messages {
		messages[i] = TestMessage{
			Type: messageTypes[rand.Intn(len(messageTypes))],
		}
	}

	return messages
}
