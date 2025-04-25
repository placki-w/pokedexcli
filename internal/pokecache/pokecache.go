package pokecache

import (
	"sync"
	"time"
)

type cacheEntry struct {
	createdAt time.Time
	val       []byte
}

type Cache struct {
	entries  map[string]cacheEntry
	mutex    sync.Mutex
	interval time.Duration
}

func NewCache(interval time.Duration) *Cache {
	cache := &Cache{
		entries:  make(map[string]cacheEntry),
		interval: interval,
	}

	//start the reaping goroutine
	go cache.reapLoop()

	return cache
}

func (c *Cache) Add(key string, val []byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry := cacheEntry{
		createdAt: time.Now(),
		val:       val,
	}

	c.entries[key] = entry

}

func (c *Cache) Get(key string) ([]byte, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entry, ok := c.entries[key]

	if !ok {
		return nil, false
	}

	return entry.val, true
}

func (c *Cache) reapLoop() {
	ticker := time.NewTicker(c.interval)

	// run forever in this goroutine
	for {
		//waiting for the next tick
		<-ticker.C

		//lock when modifying the map
		c.mutex.Lock()

		//get current time
		now := time.Now()

		//check entries in the map

		for k, entry := range c.entries {
			//if entry is older than interval, delete it
			if now.Sub(entry.createdAt) > c.interval {
				delete(c.entries, k)
			}
		}

		c.mutex.Unlock()

	}
}
