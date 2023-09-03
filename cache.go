package memory_cache

import (
	"sync"
	"sync/atomic"
	"time"
)

var (
	caches = make([]any, 0)
	wakeup chan bool
	lock   sync.Mutex
	once   sync.Once
)

type Cleanable interface {
	Clean() (nextCleanTime time.Time)
}

type EnhanceCache[K string, V any] struct {
	DefaultExpiration time.Duration
	NextScan          time.Time
	Items             sync.Map // just like map[string]*Item
	Eviction          func(key string, value V)
}

type Item struct {
	Object     any
	Status     int32
	Expiration time.Time
}

func NewCache[V any](expired time.Duration) *EnhanceCache[string, V] {
	return NewCacheWithEviction[V](expired, nil)
}

func NewCacheWithEviction[V any](expired time.Duration, eviction func(key string, value V)) *EnhanceCache[string, V] {
	once.Do(func() {
		wakeup = make(chan bool, 1)
		go clearExpired()
	})

	cache := &EnhanceCache[string, V]{
		Items:             sync.Map{},
		NextScan:          time.Now().Add(expired),
		DefaultExpiration: expired,
		Eviction:          eviction,
	}

	lock.Lock()
	defer lock.Unlock()

	caches = append(caches, cache)

	select {
	case wakeup <- true:
	default:
	}

	return cache
}

func (ec *EnhanceCache[K, V]) Get(key string) (v V, exist bool) {
	wrap, find := ec.Items.Load(key)

	if !find {
		return
	}

	item := wrap.(*Item)
	if item.Expiration.After(time.Now()) {
		return item.Object.(V), true
	}

	if atomic.CompareAndSwapInt32(&item.Status, 0, 1) {
		ec.Items.Delete(key)
		atomic.StoreInt32(&item.Status, 2)
		if ec.Eviction != nil {
			go ec.Eviction(key, item.Object.(V))
		}
		return
	}

	// All operations need to wait for the deletion to complete,
	// extreme short time to wait, only trigger in stubbing test.
	// Even if 1000 goroutine are create to execute,
	// the wait will not be triggered.
	for status := atomic.LoadInt32(&item.Status); status != 2; {
		status = atomic.LoadInt32(&item.Status)
	}

	return
}

func (ec *EnhanceCache[K, V]) Delete(key string) {
	value, exist := ec.Get(key)
	if !exist {
		return
	}

	ec.Items.Delete(key)
	if ec.Eviction != nil {
		go ec.Eviction(key, value)
	}
}

func (ec *EnhanceCache[K, V]) Set(key string, value V, expiration time.Duration) {
	ec.Get(key)

	item := &Item{
		Object:     value,
		Expiration: time.Now().Add(expiration),
	}

	ec.Items.Store(key, item)
}

func (ec *EnhanceCache[K, V]) LoadOrStore(key string, value V) (V, bool) {
	if target, exist := ec.Get(key); exist {
		return target, true
	}

	item := &Item{
		Object:     value,
		Expiration: time.Now().Add(ec.DefaultExpiration),
	}

	warp, load := ec.Items.LoadOrStore(key, item)
	item = warp.(*Item)
	return item.Object.(V), load
}

func (ec *EnhanceCache[K, V]) DecrInt(key string, value any) (current V) {
	var decrement any
	switch value.(type) {
	case int32:
		decrement = -(value.(int32))
	case int64:
		decrement = -(value.(int64))
	default:
		// Some code check will force to deal with error.
		// Don't want to deal with error that will never happen,
		// so panic instead of return error
		panic("only allow decrement int64、int32")
	}
	return ec.IncrInt(key, decrement)
}

// IncrInt increases the int value stored under the given key by n,
// create new key if key does not exist.
// Only allow increment *int64、*int32,
// can multiply a float number by 100 and turn it into an integer to use it.
func (ec *EnhanceCache[K, V]) IncrInt(key string, value interface{}) (current V) {
	// A pile of shit
	// Maybe I should write a cache dedicated to saving numbers
	switch value.(type) {
	case int32:
		if _, ok := any(current).(*int32); !ok {
			panic("value not match generic type")
		}
		inc := value.(int32)
		old, load := ec.LoadOrStore(key, any(&inc).(V))
		if !load {
			return old
		}
		tmp := atomic.AddInt32(any(old).(*int32), value.(int32))
		return any(&tmp).(V)
	case int64:
		if _, ok := any(current).(*int64); !ok {
			panic("value not match generic type")
		}
		inc := value.(int64)
		old, load := ec.LoadOrStore(key, any(&inc).(V))
		if !load {
			return old
		}
		tmp := atomic.AddInt64(any(old).(*int64), value.(int64))
		return any(&tmp).(V)
	default:
		// Some code check will force to deal with error.
		// Don't want to deal with error that will never happen,
		// so panic instead of return error
		panic("only allow increment *int64、*int32")
	}
}

func (ec *EnhanceCache[K, V]) Clean() (nextCleanTime time.Time) {
	if ec.NextScan.After(time.Now()) {
		return ec.NextScan
	}
	ec.Items.Range(func(key, value any) bool {
		ec.Get(key.(string))
		return true
	})
	ec.NextScan = time.Now().Add(ec.DefaultExpiration)
	return ec.NextScan
}

func (ec *EnhanceCache[K, V]) Clear() {
	ec.Items = sync.Map{}
}

func (ec *EnhanceCache[K, V]) SetWithEviction(evictionFunc func(key string, value V)) {
	lock.Lock()
	ec.Eviction = evictionFunc
	lock.Unlock()
}

func (ec *EnhanceCache[K, V]) Size() int {
	size := 0
	ec.Items.Range(func(key, value any) bool {
		_, ok := ec.Get(key.(string))
		if ok {
			size += 1
		}
		return true
	})
	return size
}

func clearExpired() {
	for {
		lock.Lock()
		nearest := time.Now().Add(time.Hour)
		for _, cache := range caches {
			clearer, ok := cache.(Cleanable)
			if !ok {
				panic("cache must implement Clearable interface")
			}
			nextScan := clearer.Clean()
			if nextScan.Before(nearest) {
				nearest = nextScan
			}
		}
		// It's unsafe to use defer to unlock in loop
		lock.Unlock()
		// if one goroutine can't clear faster than write,
		// the program may have OOM exception.
		// So just need a goroutine to clear
		if time.Now().After(nearest) {
			continue
		}
		if nearest.Sub(time.Now()) > time.Hour {
			nearest = time.Now().Add(time.Hour)
		}

		timer := time.NewTimer(nearest.Sub(time.Now()))
		select {
		case <-timer.C:
			continue
		case <-wakeup:
			// If create cache during clearExpired,
			// it needs to start clear immediately
			continue
		}
	}
}
