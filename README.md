# MemoryCache
This project is a lock-free, thread-safe, easy-to-use memory cache module for Go that supports generics. 
It aims to provide efficient and concurrent data storage and retrieval.
## Features
* Support caching of any type of data using generics
* Thread-safe and lock-free implementation for high concurrency
* Simple and easy to use with a clean API interface
* Support setting expiration time, and use less goroutines to clean up expired cache data
## Installation
go get gitee.com/MetaphysicCoding/memory-cache

## Compare With go-cache

Source Code See test branch benchmark_test.go

| Benchmark                       |    Iterations | Time/Iteration (ns/op) |
| ------------------------------- | ------------: | ---------------------: |
| GoCache1000KeyReadWriteTest     |             1 |         15,894,992,600 |
| MemoryCache1000KeyReadWriteTest |             1 |          7,810,338,800 |
| GoCache100KeyReadWriteTest      | 1,000,000,000 |                 0.1113 |
| MemoryCache100KeyReadWriteTest  | 1,000,000,000 |                 0.0635 |

## Usage
```go
import (
	"fmt"
	cache "gitee.com/MetaphysicCoding/memory-cache"
	"time"
)

type Foo struct {
	Name string
}

func main() {
	// Create a cache with a default expiration time of 10 Seconds,
	// the generic type of the value is *Foo and key is string
	c := cache.NewCache[*Foo](10 * time.Second)

	// Set the value of the key "foo1" to &Foo{"foo1"}, with the default expiration time
	c.Set("foo1", &Foo{"foo1"})

	// Get the *Foo associated with the key "foo1" from the cache
	foo1, exist := c.Get("foo1")
	if exist {
		fmt.Printf("foo1: %#v\n", foo1)
	}

	// LoadOrStore is atomic and use default expiration time as expiration time
	// LoadOrStore returns the existing value for the key if present.
	// Otherwise, it stores and returns the given value.
	// The loaded result is true if the value was loaded, false if stored.
	foo2, load := c.LoadOrStore("foo2", &Foo{"foo2"})
	if load {
		fmt.Printf("The value of foo2 has been set before, foo2: %#v\n", foo2)
	} else {
		fmt.Printf("The value of foo2 is not set before, foo2: %#v\n", foo2)
	}

	// If generic type of value is *int64、*int32,
	// IncrInt、DecrInt can be used to increase or decrease the value
	// If generic type of value is not *int64、*int32,
	// IncrInt、DecrInt will panic
	// If you want to increase or decrease the value of float,
	// you can multiply a float number by 100 and turn it into an integer to use it.
	numeric := cache.NewCache[*int64](10 * time.Second)

	// If foo3 is not exist, set foo3 to 1 and return 1.
	// Otherwise, atomic increase foo3 by 1 and return the new value.
	// Value must convert to int64
	current := numeric.IncrInt("foo3", int64(12))

	// Should convert current from a pointer to an integer
	fmt.Printf("foo3: %d\n", *current) // output: foo3: 12
}
```
**Note1: This project includes a dedicated testing branch for running test cases. Make sure to switch to that branch for testing purposes.**

**Note2: I can only reopen a repository due to I can't fix enhance-cache's go get bug :( .**
