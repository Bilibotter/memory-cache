package memory_cache

import (
	cache1 "github.com/patrickmn/go-cache"
	"strconv"
	"testing"
	"time"
)

func BenchmarkGoCache100KeyReadWriteTest(b *testing.B) {
	c := cache1.New(time.Second*10, time.Hour)
	for i := 0; i < 100; i++ {
		c.Set(strconv.Itoa(i), i, time.Hour)
	}
	for i := 0; i < 100; i++ {
		for j := 0; j < 1000; j++ {
			go c.Set(strconv.Itoa(i), i, time.Hour)
			go c.Get(strconv.Itoa(i))
		}
	}
}

func BenchmarkMemoryCache100KeyReadWriteTest(b *testing.B) {
	c := NewCache[int](1 * time.Hour)
	for i := 0; i < 100; i++ {
		c.Set(strconv.Itoa(i), i, time.Hour)
	}
	for i := 0; i < 100; i++ {
		for j := 0; j < 1000; j++ {
			go c.Set(strconv.Itoa(i), i, time.Hour)
			go c.Get(strconv.Itoa(i))
		}
	}
}

func BenchmarkGoCache1000KeyReadWriteTest(b *testing.B) {
	c := cache1.New(time.Second*10, time.Hour)
	for i := 0; i < 1000; i++ {
		c.Set(strconv.Itoa(i), i, time.Hour)
	}
	for i := 0; i < 1000; i++ {
		for j := 0; j < 10000; j++ {
			go c.Set(strconv.Itoa(i), i, time.Hour)
			go c.Get(strconv.Itoa(i))
		}
	}
}

func BenchmarkMemoryCache1000KeyReadWriteTest(b *testing.B) {
	c := NewCache[int](1 * time.Hour)
	for i := 0; i < 1000; i++ {
		c.Set(strconv.Itoa(i), i, time.Hour)
	}
	for i := 0; i < 1000; i++ {
		for j := 0; j < 10000; j++ {
			go c.Set(strconv.Itoa(i), i, time.Hour)
			go c.Get(strconv.Itoa(i))
		}
	}
}

func BenchmarkGoCache10000KeyReadWriteTest(b *testing.B) {
	c := cache1.New(time.Second*10, time.Hour)
	for i := 0; i < 10000; i++ {
		c.Set(strconv.Itoa(i), i, time.Hour)
	}
	for i := 0; i < 10000; i++ {
		for j := 0; j < 1000; j++ {
			go c.Set(strconv.Itoa(i), i, time.Hour)
			go c.Get(strconv.Itoa(i))
		}
	}
}

func BenchmarkMemoryCache10000KeyReadWriteTest(b *testing.B) {
	c := NewCache[int](1 * time.Hour)
	for i := 0; i < 10000; i++ {
		c.Set(strconv.Itoa(i), i, time.Hour)
	}
	for i := 0; i < 10000; i++ {
		for j := 0; j < 1000; j++ {
			go c.Set(strconv.Itoa(i), i, time.Hour)
			go c.Get(strconv.Itoa(i))
		}
	}
}
