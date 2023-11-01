// Package condqueue provides a concurrent queue, on which consumers can wait for an item satisfying
// a given condition, and producers can add items to wake consumers.
package condqueue

import (
	"context"
	"slices"
	"sync"
)

// CondQueue is a concurrent queue of items of type T. Consumer goroutines can call
// [CondQueue.AwaitMatchingItem] to wait for an item matching a given condition to arrive in the queue.
// Producer goroutines can call [CondQueue.Add], which wakes any waiting consumers.
//
// A CondQueue must be initialized with condqueue.New(), and must never be dereferenced.
type CondQueue[T any] struct {
	items   []T
	lock    sync.Mutex
	waiters []chan wake
}

type wake struct{}

func New[T any]() *CondQueue[T] {
	return &CondQueue[T]{items: nil, lock: sync.Mutex{}, waiters: nil}
}

// Add adds the given item to the queue, and wakes all goroutines waiting on [CondQueue.AwaitMatchingItem],
// so they may see if the new item is a match.
func (queue *CondQueue[T]) Add(item T) {
	queue.lock.Lock()
	defer queue.lock.Unlock()

	queue.items = append(queue.items, item)
	for _, waiter := range queue.waiters {
		select {
		case waiter <- wake{}:
		default:
			// If channel is full: waiter is already awoken, do nothing
		}
	}
}

// AwaitMatchingItem goes through the items in the queue, and returns an item where isMatch(item)
// returns true. If no matching item is already in the queue, it waits until a match arrives.
//
// If ctx is canceled before a match is found, ctx.Err() is returned. If the context never cancels,
// e.g. when using [context.Background], the error can safely be ignored. If a matching item is
// never received, and the context never cancels, this may halt the calling goroutine forever. It is
// therefore advised to use [context.WithTimeout] or similar.
//
// If multiple goroutines calling this may match on the same item, only one of them will receive the
// item - i.e., every call to Add corresponds with one returned match from AwaitMatchingItem.
func (queue *CondQueue[T]) AwaitMatchingItem(
	ctx context.Context,
	isMatch func(item T) bool,
) (matchingItem T, cancelErr error) {
	waiter := make(chan wake, 1)
	queue.lock.Lock()
	queue.waiters = append(queue.waiters, waiter)

	for {
		// Iterates in reverse, to get the newest item first
		for i := len(queue.items) - 1; i >= 0; i-- {
			item := queue.items[i]

			if isMatch(item) {
				queue.items = slices.Delete(queue.items, i, i+1)
				queue.removeWaiter(waiter)
				queue.lock.Unlock()
				return item, nil
			}
		}

		queue.lock.Unlock()

		select {
		case <-ctx.Done():
			queue.lock.Lock()
			queue.removeWaiter(waiter)
			queue.lock.Unlock()
			return matchingItem, ctx.Err()
		case <-waiter:
			queue.lock.Lock()
		}
	}
}

// Clear removes all items from the queue.
func (queue *CondQueue[T]) Clear() {
	queue.lock.Lock()
	queue.items = nil
	queue.lock.Unlock()
}

func (queue *CondQueue[T]) removeWaiter(waiter chan wake) {
	for i, candidate := range queue.waiters {
		if waiter == candidate {
			queue.waiters = slices.Delete(queue.waiters, i, i+1)
			return
		}
	}
}
