// SPDX-License-Identifier: AGPL-3.0-only

package cache

// Len returns the number of entries held, expired ones included until a
// lookup or an eviction drops them.
func (c *Cache[V]) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}

// lookup returns a copy of the live entry for key and marks it as the most
// recently used. An expired entry is dropped.
func (c *Cache[V]) lookup(key string) (entry[V], bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return entry[V]{}, false
	}
	e := el.Value.(*entry[V])
	if !c.now().Before(e.expires) {
		c.order.Remove(el)
		delete(c.items, key)
		return entry[V]{}, false
	}
	c.order.MoveToFront(el)
	return *e, true
}

// store caches a value for Config.TTL, an error accepted by Config.Negative
// for Config.NegativeTTL, and nothing else. Beyond Config.MaxEntries it
// evicts the least recently used entry.
func (c *Cache[V]) store(key string, v V, err error) {
	ttl := c.cfg.TTL
	if err != nil {
		if c.cfg.Negative == nil || !c.cfg.Negative(err) {
			return
		}
		ttl = c.cfg.NegativeTTL
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e := &entry[V]{key: key, value: v, err: err, expires: c.now().Add(ttl)}
	if el, ok := c.items[key]; ok {
		el.Value = e
		c.order.MoveToFront(el)
		return
	}
	c.items[key] = c.order.PushFront(e)
	if c.order.Len() > c.cfg.MaxEntries {
		oldest := c.order.Back()
		c.order.Remove(oldest)
		delete(c.items, oldest.Value.(*entry[V]).key)
	}
}
