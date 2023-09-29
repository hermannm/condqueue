// Package condqueue provides a concurrent queue, on which consumers can wait for an item meeting a
// given condition, and producers can add items to wake consumers.
package condqueue

import (
	"context"
	"slices"
)

// CondQueue is a concurrent queue of items of type T. Consumer goroutines can call
// AwaitMatchingItem to wait for an item matching a given condition to arrive in the queue. Producer
// goroutines can call AddItem, which wakes any waiting consumers.
//
// A CondQueue must be initialized with condqueue.New(), and must never be dereferenced.
type CondQueue[T any] struct {
	items   []T
	lock    chan lock
	waiters []chan wake
}

type lock struct{}
type wake struct{}

func New[T any]() *CondQueue[T] {
	return &CondQueue[T]{items: nil, lock: make(chan lock, 1), waiters: nil}
}

// AddItem adds the given item to the queue, and wakes all goroutines waiting on AwaitMatchingItem,
// so they may see if the new item is a match.
//
// If ctx is canceled before the queue's lock is acquired, ctx.Err() is returned. If the context
// never cancels, e.g. when using [context.Background], the error can safely be ignored.
func (queue *CondQueue[T]) AddItem(ctx context.Context, item T) (cancelErr error) {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case queue.lock <- lock{}:
	}

	queue.items = append(queue.items, item)
	for _, waiter := range queue.waiters {
		select {
		case waiter <- wake{}:
		default:
			// If channel is full: waiter is already awoken, do nothing
		}
	}

	<-queue.lock
	return nil
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
// item - i.e., every call to AddItem corresponds with one returned match from AwaitMatchingItem.
func (queue *CondQueue[T]) AwaitMatchingItem(
	ctx context.Context,
	isMatch func(T) bool,
) (match T, cancelErr error) {
	waiter := make(chan wake, 1)

	select {
	case <-ctx.Done():
		return match, ctx.Err()
	case queue.lock <- lock{}:
	}

	queue.waiters = append(queue.waiters, waiter)

	for {
		// Iterates in reverse, to get the newest item first
		for i := len(queue.items) - 1; i >= 0; i-- {
			item := queue.items[i]

			if isMatch(item) {
				queue.items = slices.Delete(queue.items, i, i+1)
				queue.removeWaiter(waiter)
				<-queue.lock
				return item, nil
			}
		}

		<-queue.lock

		select {
		case <-ctx.Done():
			queue.lock <- lock{}
			queue.removeWaiter(waiter)
			<-queue.lock
			return match, ctx.Err()
		case <-waiter:
			queue.lock <- lock{}
		}
	}
}

func (queue *CondQueue[T]) removeWaiter(waiter chan wake) {
	remainingWaiters := make([]chan wake, 0, cap(queue.waiters))

	for _, previousWaiter := range queue.waiters {
		if previousWaiter != waiter {
			remainingWaiters = append(remainingWaiters, previousWaiter)
		}
	}

	queue.waiters = remainingWaiters
}
