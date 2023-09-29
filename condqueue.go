package condqueue

import (
	"slices"
	"sync"
)

type CondQueue[T any] struct {
	items []T
	cond  sync.Cond
}

func New[T any]() *CondQueue[T] {
	return &CondQueue[T]{items: nil, cond: sync.Cond{L: &sync.Mutex{}}}
}

func (queue *CondQueue[T]) AddItem(item T) {
	queue.cond.L.Lock()
	queue.items = append(queue.items, item)
	queue.cond.L.Unlock()
	queue.cond.Broadcast()
}

func (queue *CondQueue[T]) AwaitMatchingItem(isMatch func(T) bool) (match T) {
	queue.cond.L.Lock()
	for {
		// Iterates in reverse, to get the newest item first
		for i := len(queue.items) - 1; i >= 0; i-- {
			item := queue.items[i]

			if isMatch(item) {
				queue.items = slices.Delete(queue.items, i, i+1)
				queue.cond.L.Unlock()
				return item
			}
		}

		queue.cond.Wait()
	}
}
