package main

import (
	"container/list"
	"sync"
)

type LRU[K comparable, V any] interface {
	Get(key K) (value V, ok bool)
	Set(key K, value V)
}

type entry[K comparable, V any] struct {
	key   K
	value V
}

type LRUCache[K comparable, V any] struct {
	capacity int
	mu       sync.Mutex
	ll       list.List
	items    map[K]*list.Element
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
	}
}

func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capacity <= 0 {
		var zero V
		return zero, false
	}

	element, exists := c.items[key]

	if !exists {
		var zero V
		return zero, false
	}

	// Успешное чтение обновляет давность использования.
	c.ll.MoveToFront(element)

	item := element.Value.(entry[K, V])

	return item.value, true
}

func (c *LRUCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capacity <= 0 {
		return
	}

	// Обновляем существующий элемент без изменения давности.
	if element, exists := c.items[key]; exists {
		element.Value = entry[K, V]{
			key:   key,
			value: value,
		}
		return
	}

	// Если кэш заполнен, удаляем самый старый элемент.
	if len(c.items) >= c.capacity {
		oldest := c.ll.Back()

		if oldest != nil {
			oldEntry := oldest.Value.(entry[K, V])

			delete(c.items, oldEntry.key)
			c.ll.Remove(oldest)
		}
	}

	// Добавляем новый элемент в начало списка.
	element := c.ll.PushFront(entry[K, V]{
		key:   key,
		value: value,
	})

	c.items[key] = element
}
