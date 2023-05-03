package fake

import (
	"sync"
)

type Cache[Key ~string, Value any] struct {
	lock   sync.Mutex
	values map[Key]Value
	hits   int
	misses int
}

func NewCache[Key ~string, Value any]() *Cache[Key, Value] {
	return &Cache[Key, Value]{values: map[Key]Value{}}
}

func (c *Cache[Key, Value]) Get(k Key) (Value, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	v, ok := c.values[k]
	if ok {
		c.hits++
	} else {
		c.misses++
	}
	return v, ok
}

func (c *Cache[Key, Value]) Set(k Key, v Value) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.values[k] = v
}

func (c *Cache[Key, Value]) Clear() {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.values = map[Key]Value{}
}

func (c *Cache[Key, Value]) Hits() int {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.hits
}

func (c *Cache[Key, Value]) Misses() int {
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.misses
}

func (c *Cache[Key, Value]) Size() int {
	c.lock.Lock()
	defer c.lock.Unlock()

	return len(c.values)
}
