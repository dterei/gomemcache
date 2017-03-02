package main

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
)

// we store constant KV-pair sizes for testing to make checking the resource
// limits easier: key + value + flags
const KV_SIZE = 4 + 5 + 4

var value = []byte("value")
var flag = []byte("flag")

func StoreKey(c *Cache, key string, val []byte) {
	c.Set([]byte(key), val, flag, 0)
}

func DeleteKey(c *Cache, key string) Status {
	return c.Delete([]byte(key), 0)
}

func CheckKey(t *testing.T, c *Cache, key string, val []byte) {
	i := c.Get([]byte(key))
	if i == nil {
		t.Error("Couldn't find key in cache\n")
	} else if bytes.Compare(val, i.value) != 0 {
		t.Errorf("Corrupted value in cache: %s vs %s\n", val, i.value)
	}
}

func CheckNoKey(t *testing.T, c *Cache, key string) {
	i := c.Get([]byte(key))
	if i != nil {
		t.Error("Found non-existent key in cache\n")
	}
}

func CheckKeyInRange(t *testing.T, c *Cache, key string, vals [][]byte) {
	i := c.Get([]byte(key))
	if i == nil {
		t.Error("Couldn't find key in cache\n")
	}

	for _, val := range vals {
		if bytes.Compare(val, i.value) == 0 {
			return
		}
	}
	t.Errorf("Corrupted value in cache: %s\n", i.value)
}

func TestCacheSetGet(t *testing.T) {
	cache := NewCache(100000)
	StoreKey(cache, "key", value)
	CheckKey(t, cache, "key", value)
}

func TestCacheGetMiss(t *testing.T) {
	cache := NewCache(100000)
	StoreKey(cache, "key1", value)
	CheckNoKey(t, cache, "key2")
}

func TestCacheSetGetMany(t *testing.T) {
	cache := NewCache(100000)

	StoreKey(cache, "key1", value)
	StoreKey(cache, "key2", value)
	StoreKey(cache, "key3", value)

	CheckKey(t, cache, "key1", value)
	CheckKey(t, cache, "key2", value)
	CheckKey(t, cache, "key3", value)
	CheckNoKey(t, cache, "key4")
}

func TestCacheDelete(t *testing.T) {
	cache := NewCache(100000)
	StoreKey(cache, "key", value)
	CheckKey(t, cache, "key", value)
	b := DeleteKey(cache, "key")
	if b != STATUS_OK {
		t.Error("Couldn't delete key in cache")
	}
	CheckNoKey(t, cache, "key")
}

func TestCacheLRU(t *testing.T) {
	cache := NewCache(1 * KV_SIZE)
	StoreKey(cache, "key1", value)
	StoreKey(cache, "key2", value)
	CheckNoKey(t, cache, "key1")
	CheckKey(t, cache, "key2", value)
}

func PrintLRU(c *Cache) {
	for item := c.lru.tail; item != nil; item = item.lru.next {
		fmt.Printf("%s -> ", item.key)
	}
	fmt.Printf("|\n")
}

func TestCacheLRUMany(t *testing.T) {
	cache := NewCache(3 * KV_SIZE)

	StoreKey(cache, "key1", value)
	StoreKey(cache, "key2", value)
	StoreKey(cache, "key3", value)
	StoreKey(cache, "key4", value)
	StoreKey(cache, "key5", value)

	CheckNoKey(t, cache, "key1")
	CheckNoKey(t, cache, "key2")
	CheckKey(t, cache, "key3", value)
	CheckKey(t, cache, "key4", value)
	CheckKey(t, cache, "key5", value)
	CheckNoKey(t, cache, "key6")
}

func TestCacheLRUUpdates(t *testing.T) {
	cache := NewCache(3 * KV_SIZE)

	StoreKey(cache, "key1", value)
	StoreKey(cache, "key2", value)
	StoreKey(cache, "key3", value)
	CheckKey(t, cache, "key1", value)
	StoreKey(cache, "key4", value)
	StoreKey(cache, "key5", value)

	CheckKey(t, cache, "key1", value)
	CheckNoKey(t, cache, "key2")
	CheckNoKey(t, cache, "key3")
	CheckKey(t, cache, "key4", value)
	CheckKey(t, cache, "key5", value)
}

const (
	N_KV_SIZE      = 12 + 5 + 8 + 12 + 4 // namespace + ":key:" + keyspace + value + flags
	N_WORKERS      = 10
	N_PRIVATE_KEYS = 10000
	N_SHARED_KEYS  = 1000
	N_REPEAT       = 4
	N_CACHE_SIZE   = N_PRIVATE_KEYS*N_WORKERS + N_SHARED_KEYS
)

func ConcurrencyWorker(t *testing.T, cache *Cache, namespaces [][]byte, id uint) {
	var wg sync.WaitGroup
	namespace := namespaces[id]

	// run queries over private keys
	wg.Add(1)
	go func() {
		for j := 0; j < N_REPEAT; j++ {
			for i := 0; i < N_PRIVATE_KEYS; i++ {
				key := fmt.Sprintf("%s:key:%08d", namespace, i)
				StoreKey(cache, key, namespace)
				CheckKey(t, cache, key, namespace)
			}
			for i := 0; i < N_PRIVATE_KEYS; i++ {
				key := fmt.Sprintf("%s:key:%08d", namespace, i)
				CheckKey(t, cache, key, namespace)
			}
		}
		wg.Done()
	}()

	// run queries over shared keys
	wg.Add(1)
	go func() {
		for j := 0; j < N_REPEAT; j++ {
			for i := 0; i < N_SHARED_KEYS; i++ {
				key := fmt.Sprintf("%s:key:%08d", "SharedSpace0", i)
				StoreKey(cache, key, []byte(namespace))
			}
			for i := 0; i < N_SHARED_KEYS; i++ {
				key := fmt.Sprintf("%s:key:%08d", "SharedSpace0", i)
				CheckKeyInRange(t, cache, key, namespaces)
			}
		}
		wg.Done()
	}()

	wg.Wait()
}

func TestConcurrency(t *testing.T) {
	cache := NewCache(N_CACHE_SIZE * N_KV_SIZE)

	var namespaces [][]byte
	for i := 0; i < N_WORKERS; i++ {
		name := []byte(fmt.Sprintf("TestWorker%02d", i))
		namespaces = append(namespaces, name)
	}

	for i := uint(0); i < N_WORKERS; i++ {
		id := i
		t.Run(string(namespaces[id]), func(t *testing.T) {
			t.Parallel()
			ConcurrencyWorker(t, cache, namespaces, id)
		})
	}
}
