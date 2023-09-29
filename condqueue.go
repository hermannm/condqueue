package condqueue

import (
	"context"
	"slices"
	"sync"
)

type CondQueue[T any] struct {
	items   []T
	lock    sync.Mutex
	waiters []chan wake
}

type wake struct{}

func New[T any]() *CondQueue[T] {
	return &CondQueue[T]{items: nil, lock: sync.Mutex{}, waiters: nil}
}

func (queue *CondQueue[T]) AddItem(ctx context.Context, item T) (cancelErr error) {
	queue.lock.Lock()

	queue.items = append(queue.items, item)
	for _, waiter := range queue.waiters {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case waiter <- wake{}:
		default:
			// If channel is full: waiter is already awoken, do nothing
		}
	}

	queue.lock.Unlock()
	return nil
}

func (queue *CondQueue[T]) AwaitMatchingItem(
	ctx context.Context,
	isMatch func(T) bool,
) (match T, cancelErr error) {
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
			return match, ctx.Err()
		case <-waiter:
			queue.lock.Lock()
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
