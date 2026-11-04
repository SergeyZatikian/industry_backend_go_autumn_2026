package main

import "container/list"

type entry[K comparable, V any] struct {
	key   K
	value V
}

type LRUCache[K comparable, V any] struct {
	capacity int
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
	if c.capacity <= 0 {
		var zero V
		return zero, false
	}

	element, exists := c.items[key]

	if !exists {
		var zero V
		return zero, false
	}

	// Успешное чтение делает элемент самым свежим.
	c.ll.MoveToFront(element)

	item := element.Value.(entry[K, V])

	return item.value, true
}

func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}

	// Если ключ уже есть, обновляем только значение.
	if element, exists := c.items[key]; exists {
		element.Value = entry[K, V]{
			key:   key,
			value: value,
		}

		return
	}

	// Если места нет, удаляем самый старый элемент.
	if len(c.items) >= c.capacity {
		oldest := c.ll.Back()

		if oldest != nil {
			oldEntry := oldest.Value.(entry[K, V])

			delete(c.items, oldEntry.key)

			c.ll.Remove(oldest)
		}
	}

	// Новый элемент становится самым свежим.
	element := c.ll.PushFront(entry[K, V]{
		key:   key,
		value: value,
	})

	c.items[key] = element
}

type LRU[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
}
