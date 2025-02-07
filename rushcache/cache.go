package rushcache

import (
	"rushcache/lru"
	"sync"
)

type Cache struct {
	mu         sync.Mutex // 互斥锁
	lru        *lru.Cache // lru 策略缓存
	cacheBytes int64      // 缓存的大小
}

func (c *Cache) add(key string, value ByteView) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		c.lru = lru.New(c.cacheBytes, nil)
	}
	c.lru.Add(key, value)
}

func (c *Cache) get(key string) (value ByteView, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lru == nil {
		return
	}
	if v, ok := c.lru.Get(key); ok {
		return v.(ByteView), ok
	}
	return
}
