package condqueue

import (
	"context"
	"slices"
)

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

func (queue *CondQueue[T]) AddItem(ctx context.Context, item T) (cancelErr error) {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case queue.lock <- lock{}:
	}

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

	<-queue.lock
	return nil
}

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
