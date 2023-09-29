package condqueue

import (
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
		foundMatch := false
		remainingItems := make([]T, 0, cap(queue.items))

		for _, item := range queue.items {
			foundMatch = isMatch(item)

			if foundMatch {
				match = item
			} else {
				remainingItems = append(remainingItems, item)
			}
		}

		if foundMatch {
			queue.items = remainingItems
			queue.cond.L.Unlock()
			return match
		}

		queue.cond.Wait()
	}
}
