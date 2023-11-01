// Package condqueue provides a concurrent queue, on which consumers can wait for an item satisfying
// a given condition.
package condqueue

import (
	"context"
	"slices"
	"sync"
)

// CondQueue is a concurrent queue of items of type T.
// Consumer goroutines can call [CondQueue.AwaitMatchingItem] to wait for an item matching a given
// condition to arrive in the queue.
// Producer goroutines can call [CondQueue.Add], which passes the item to a matching consumer.
//
// A CondQueue must be initialized with condqueue.New(), and must never be dereferenced.
type CondQueue[T any] struct {
	consumers       []consumer[T]
	unconsumedItems []T
	lock            sync.Mutex
}

type consumer[T any] struct {
	isMatch      func(item T) bool
	matchingItem chan T
	canceled     chan cancel
}

type cancel struct{}

func New[T any]() *CondQueue[T] {
	return &CondQueue[T]{}
}

// Add checks if the given item is a match for any consumers currently waiting on the queue.
//
// If a matching consumer is found, the consumer is woken and given the item, then removed from the
// queue. The item is only given to a single consumer (older consumers prioritized), so any other
// matching consumers must wait for a later item.
//
// If no matching consumer is found, the item is stored so that future consumers may match on it.
func (queue *CondQueue[T]) Add(item T) {
	queue.lock.Lock()
	defer queue.lock.Unlock()

	itemConsumed := false
	// Keeps same backing memory (safe, since we're replacing queue.consumers with this after)
	remainingConsumers := queue.consumers[:0]

	for _, consumer := range queue.consumers {
		if !itemConsumed && consumer.isMatch(item) {
			select {
			case consumer.matchingItem <- item:
				itemConsumed = true
			case <-consumer.canceled:
				// Continues loop without adding to remainingConsumers, so it is removed
				continue
			}
		} else {
			select {
			case <-consumer.canceled:
				continue
			default: // If we get here, the consumer was not canceled, i.e. should remain
				remainingConsumers = append(remainingConsumers, consumer)
			}
		}
	}

	queue.consumers = remainingConsumers

	if !itemConsumed {
		queue.unconsumedItems = append(queue.unconsumedItems, item)
	}
}

// AwaitMatchingItem goes through unconsumed items in the queue, and returns an item where
// isMatch(item) returns true. If no match is found there, it waits until given one from
// [CondQueue.Add].
//
// If ctx is canceled before a match is found, ctx.Err() is returned. If the context never cancels,
// e.g. when using [context.Background], the error can safely be ignored. If a matching item is
// never received, and the context never cancels, this may halt the calling goroutine forever. It is
// therefore advised to use [context.WithTimeout] or similar.
//
// If multiple concurrent consumers may match on the same item, only one of them will receive the
// item - i.e., every call to Add corresponds with one returned match from AwaitMatchingItem.
func (queue *CondQueue[T]) AwaitMatchingItem(
	ctx context.Context,
	isMatch func(item T) bool,
) (matchingItem T, cancelErr error) {
	queue.lock.Lock()

	// First we check if a matching item is among previously unconsumed items
	for i, item := range queue.unconsumedItems {
		if isMatch(item) {
			queue.unconsumedItems = slices.Delete(queue.unconsumedItems, i, i+1)
			queue.lock.Unlock()
			return item, nil
		}
	}

	// Since we found no match among the unconsumed items, we create a consumer to wait for one
	consumer := consumer[T]{
		isMatch:      isMatch,
		matchingItem: make(chan T, 1),
		canceled:     make(chan cancel, 1),
	}

	queue.consumers = append(queue.consumers, consumer)
	queue.lock.Unlock()

	select {
	case matchingItem = <-consumer.matchingItem:
		return matchingItem, nil
	case <-ctx.Done():
		consumer.canceled <- cancel{}
		var zero T
		return zero, ctx.Err()
	}
}

// Clear removes all unconsumed items from the queue.
func (queue *CondQueue[T]) Clear() {
	queue.lock.Lock()
	queue.unconsumedItems = nil
	queue.lock.Unlock()
}
