package main

type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]V
}

func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	return &Cache[K, V]{
		capacity: capacity,
		items:    make(map[K]V),
	}
}

func (c *Cache[K, V]) Get(k K) (v V, ok bool) {
	if c.capacity <= 0 {
		var zero V
		return zero, false
	}

	value, found := c.items[k]
	return value, found
}

func (c *Cache[K, V]) Set(k K, v V) bool {
	if c.capacity <= 0 {
		return false
	}

	// Проверяем, существует ли ключ
	if _, exists := c.items[k]; exists {
		c.items[k] = v
		return true
	}

	// Проверяем свободное место
	if len(c.items) >= c.capacity {
		return false
	}

	// Добавляем новый элемент
	c.items[k] = v
	return true
}
