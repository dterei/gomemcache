package main

import (
	"sync"
)

// Cache represents a cache / hashmap with an finite storage limit and an LRU
// eviction policy of key-value pairs beyond that limit.
//
// Safe to use with from multiple Go routines. We adopt a simple sinle Mutex
// strategy to protect the shared hashmap. It's tempting to think we could use a
// RWMutex as a really easy improvement, but `gets` also perform a write due to
// updating the shared LRU.
//
// Memcached improves this situation by having multiple LRU's (one per
// slab-class) and using a shareded-lock to protect the hashmap too.
type Cache struct {
	maxBytes uint64
	curBytes uint64
	hashmap  map[string]*Item
	version  uint64
	lru      LRU
	sync.Mutex
}

// NewCache creates a new cache with specified storage limit.
func NewCache(maxBytes uint64) *Cache {
	return &Cache{
		maxBytes: maxBytes,
		hashmap:  make(map[string]*Item),
	}
}

// Item represents a value stored in the cache.
type Item struct {
	flags   [4]byte
	key     string
	value   []byte
	version uint64
	lru     LRUElem
}

// NewItem creates a new item for storage in the cache.
func NewItem(key string, value, flags []byte, cas uint64) *Item {
	item := &Item{
		key:     key,
		value:   value,
		version: cas,
	}
	copy(item.flags[:], flags)
	return item
}

// Size returns the total size in bytes of the item. We ignore the space
// required by the 'version' and 'lru' fields for some constant overhead.
func (item *Item) Size() uint64 {
	return uint64(len(item.flags) + len(item.key) + len(item.value))
}

// Get retrieves the specified key from the cache.
func (cache *Cache) Get(key []byte) *Item {
	cache.Lock()
	defer cache.Unlock()

	i, ok := cache.hashmap[string(key)]
	if ok {
		cache.lru.Erase(i)
		cache.lru.PushBack(i)
	}
	return i
}

// Set stores the specified key in the cache.
func (cache *Cache) Set(key, value, flags []byte, cas uint64) (uint64, Status) {
	cache.Lock()
	defer cache.Unlock()

	keyS := string(key)

	i, ok := cache.hashmap[keyS]
	if ok {
		if cas > 0 && i.version != cas {
			return 0, STATUS_KEY_EXISTS
		}
		cache.lru.Erase(i)
		cache.curBytes -= i.Size()
	} else if cas > 0 {
		return 0, STATUS_KEY_NOT_FOUND
	}

	cache.version++
	cas = cache.version
	item := NewItem(keyS, value, flags, cas)
	cache.curBytes += item.Size()
	cache.lru.PushBack(item)
	cache.hashmap[keyS] = item
	cache.evictOverflow()

	return cas, STATUS_OK
}

// Delete removes the specified key from the cache.
func (cache *Cache) Delete(key []byte, cas uint64) Status {
	cache.Lock()
	defer cache.Unlock()

	keyS := string(key)
	i, ok := cache.hashmap[keyS]
	if !ok {
		return STATUS_KEY_NOT_FOUND
	} else if cas > 0 && i.version != cas {
		return STATUS_KEY_EXISTS
	}
	cache.curBytes -= i.Size()
	cache.lru.Erase(i)
	delete(cache.hashmap, keyS)
	return STATUS_OK
}

// evictOverflow evicts key-value pairs in LRU order until the cache is within
// the specified resource constraints.
//
// The caller of this method should hold the write lock on Cache.
func (cache *Cache) evictOverflow() {
	for cache.curBytes > cache.maxBytes {
		i := cache.lru.PopFront()
		cache.curBytes -= i.Size()
		delete(cache.hashmap, i.key)
	}
}
