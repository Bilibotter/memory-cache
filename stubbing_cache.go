package memory_cache

import (
	"sync"
	"sync/atomic"
	"time"
)

var (
	stubCaches    = make([]any, 0)
	stubWakeup    chan bool
	stubLock      sync.Mutex
	stubOnce      sync.Once
	deleteTimes   int64 = 0
	finishDelete  int64 = 0 // increment by 1 each time a key is deleted
	cleanCycle    int64 = 0 // increment by 1 each time all caches that need to clean expired keys
	cleanTimes    int64 = 0 // increment by 1 each time a cache complete cleanup of expired keys
	waitDelete    int64 = 0 // increment by 1 each time a goroutine is waiting for a key to be deleted
	finishWait    int64 = 0 // increment by 1 each time a goroutine complete wait for a key to be deleted
	wakeupTimes   int64 = 0 // increment by 1 each time the goroutine that cleans up expired keys is woken up during sleep
	sleepTimes    int64 = 0 // increment by 1 each time the goroutine that cleans up the expired keys completes the sleep without being woken up
	evictionTimes int64 = 0 // increment by 1 each time the deleted key triggers eviction
)

type StubParams struct {
	DeleteTimes   int64
	FinishDelete  int64
	CleanCycle    int64
	CleanTimes    int64
	WaitDelete    int64
	FinishWait    int64
	WakeupTimes   int64
	SleepTimes    int64
	EvictionTimes int64
}

type StubCache[K string, V any] struct {
	EnhanceCache[K, V]
}

func RestStubbing() {
	atomic.StoreInt64(&deleteTimes, 0)
	atomic.StoreInt64(&cleanCycle, 0)
	atomic.StoreInt64(&cleanTimes, 0)
	atomic.StoreInt64(&finishDelete, 0)
	atomic.StoreInt64(&waitDelete, 0)
	atomic.StoreInt64(&finishWait, 0)
	atomic.StoreInt64(&wakeupTimes, 0)
	atomic.StoreInt64(&sleepTimes, 0)
	atomic.StoreInt64(&evictionTimes, 0)
}

func GetStubbingParam() StubParams {
	return StubParams{
		DeleteTimes:   atomic.LoadInt64(&deleteTimes),
		FinishDelete:  atomic.LoadInt64(&finishDelete),
		CleanCycle:    atomic.LoadInt64(&cleanCycle),
		CleanTimes:    atomic.LoadInt64(&cleanTimes),
		WaitDelete:    atomic.LoadInt64(&waitDelete),
		FinishWait:    atomic.LoadInt64(&finishWait),
		WakeupTimes:   atomic.LoadInt64(&wakeupTimes),
		SleepTimes:    atomic.LoadInt64(&sleepTimes),
		EvictionTimes: atomic.LoadInt64(&evictionTimes),
	}
}

func NewStubCacheWithEviction[V any](expired time.Duration, eviction func(key string, value V)) *StubCache[string, V] {
	stubOnce.Do(func() {
		stubWakeup = make(chan bool, 1)
		go clearStubExpired()
	})

	cache := &StubCache[string, V]{
		EnhanceCache: EnhanceCache[string, V]{
			Items:             sync.Map{},
			NextScan:          time.Now().Add(expired),
			DefaultExpiration: expired,
			Eviction:          eviction,
		},
	}

	stubLock.Lock()
	defer stubLock.Unlock()

	stubCaches = append(stubCaches, cache)

	select {
	case stubWakeup <- true:
	default:
	}

	return cache
}

func (ec *StubCache[K, V]) Delete(key string) {
	wrap, find := ec.Items.Load(key)
	if find {
		item := wrap.(*Item)
		if item.Expiration.After(time.Now()) {
			atomic.AddInt64(&deleteTimes, 1)
		}
	}

	ec.EnhanceCache.Delete(key)
}

func (ec *StubCache[K, V]) Get(key string) (v V, exist bool) {
	wrap, find := ec.Items.Load(key)

	if !find {
		return
	}

	item := wrap.(*Item)
	if item.Expiration.After(time.Now()) {
		return item.Object.(V), true
	}

	if atomic.CompareAndSwapInt32(&item.Status, 0, 1) {

		atomic.AddInt64(&deleteTimes, 1)

		time.Sleep(ec.DefaultExpiration)
		ec.Items.Delete(key)

		atomic.AddInt64(&finishDelete, 1)

		atomic.StoreInt32(&item.Status, 2)
		if ec.Eviction != nil {
			atomic.AddInt64(&evictionTimes, 1)
			go ec.Eviction(key, item.Object.(V))
		}
		return
	}

	atomic.AddInt64(&waitDelete, 1)
	// All operations need to wait for the deletion to complete,
	// extreme short time to wait, only trigger in stubbing test.
	// Even if 1000 goroutine are create to execute,
	// the wait will not be triggered.
	for status := atomic.LoadInt32(&item.Status); status != 2; {
		status = atomic.LoadInt32(&item.Status)
		time.Sleep(10 * time.Millisecond)
	}
	atomic.AddInt64(&finishWait, 1)
	return
}

func (ec *StubCache[K, V]) Clean() (nextCleanTime time.Time) {
	if ec.NextScan.After(time.Now()) {
		return ec.NextScan
	}
	ec.Items.Range(func(key, value any) bool {
		ec.Get(key.(string))
		return true
	})
	ec.NextScan = time.Now().Add(ec.DefaultExpiration)
	atomic.AddInt64(&cleanTimes, 1)
	return ec.NextScan
}

func clearStubExpired() {
	for {
		stubLock.Lock()
		nearest := time.Now().Add(time.Hour)
		for _, cache := range stubCaches {
			clearer, ok := cache.(Cleanable)
			if !ok {
				panic("cache must implement Clearable interface")
			}
			nextScan := clearer.Clean()
			if nextScan.Before(nearest) {
				nearest = nextScan
			}
		}
		atomic.AddInt64(&cleanCycle, 1)
		// It's unsafe to use defer to unlock in loop
		stubLock.Unlock()
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
			atomic.AddInt64(&sleepTimes, 1)
			continue
		case <-stubWakeup:
			atomic.AddInt64(&wakeupTimes, 1)
			// If create cache during clearExpired,
			// it needs to start clear immediately
			continue
		}
	}
}
