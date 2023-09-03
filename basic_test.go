package memory_cache

import (
	"encoding/json"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var (
	loads int64 = 0
)

type ComparableS struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func GetFuncName(f interface{}) string {
	funcValue := reflect.ValueOf(f)
	funcType := funcValue.Type()

	if funcType.Kind() != reflect.Func {
		panic("传入的参数不是函数")
	}

	funcName := runtime.FuncForPC(funcValue.Pointer()).Name()
	return funcName
}

func TestSerialization(t *testing.T) {
	c := NewCache[[]byte](time.Second)
	v := ComparableS{Id: 1, Name: "yhm"}
	b, _ := json.Marshal(v)
	c.LoadOrStore("yhm", b)
	bs, ok := c.Get("yhm")
	if !ok {
		t.Error("Get from cache failed")
		return
	}
	var value ComparableS
	if err := json.Unmarshal(bs, &value); err != nil {
		t.Error(err.Error())
		return
	}
	if v != value {
		t.Error("Value not equal")
		return
	}
	t.Logf("%s test success struct after unmarshal %#v\n",
		GetFuncName(TestSerialization), value)
}

func TestGetAfterExpired(t *testing.T) {
	c := NewCache[int](2 * time.Second)
	c.LoadOrStore("yhm", 1)
	time.Sleep(3 * time.Second)
	num, ok := c.Get("yhm")
	if ok {
		t.Errorf("Get from cache success after expired,but it should not, num=%d", num)
		return
	}
	t.Logf("%s test success \n", GetFuncName(TestGetAfterExpired))
}

func TestOverwrite(t *testing.T) {
	c := NewCache[int](time.Second)
	c.LoadOrStore("yhm", 11)
	num, ok := c.Get("yhm")
	if !ok {
		t.Error("Get from cache failed")
		return
	}
	if num != 11 {
		t.Error("Value not equal after LoadOrStore")
		return
	}
	c.Set("yhm", 12, time.Second)
	num, ok = c.Get("yhm")
	if !ok {
		t.Error("Get from cache failed")
		return
	}
	if num != 12 {
		t.Error("Value not equal after set")
		return
	}
	t.Logf("%s test success \n", GetFuncName(TestOverwrite))
}

func TestIncrement64(t *testing.T) {
	c := NewCache[*int64](time.Second)
	current := c.IncrInt("yhm", int64(1))
	if *current != 1 {
		t.Errorf("Value not equal after IncrInt without initialize, current=%d", current)
		return
	}
	current = c.IncrInt("yhm", int64(11))
	if *current != 12 {
		t.Errorf("Value not equal after IncrInt with initialize, current=%d", current)
		return
	}
	current, _ = c.Get("yhm")
	if *current != 12 {
		t.Errorf("Value not equal after get, current=%d", current)
	}
	t.Logf("%s test success current=%d\n", GetFuncName(TestIncrement64), *current)
}

func TestIncrement32(t *testing.T) {
	c := NewCache[*int32](time.Second)
	current := c.IncrInt("yhm", int32(1))
	if *current != 1 {
		t.Errorf("Value not equal after IncrInt without initialize, current=%d", current)
		return
	}
	current = c.IncrInt("yhm", int32(11))
	if *current != 12 {
		t.Errorf("Value not equal after IncrInt with initialize, current=%d", current)
		return
	}
	current, _ = c.Get("yhm")
	if *current != 12 {
		t.Errorf("Value not equal after get, current=%d", current)
	}
	t.Logf("%s test success current=%d\n", GetFuncName(TestIncrement32), *current)
}

func TestIncrement64WithSharpRace(t *testing.T) {
	c := NewCache[*int64](5 * time.Second)
	for i := 0; i < 10000; i++ {
		go c.IncrInt("yhm", int64(1))
	}
	current, exist := c.Get("yhm")
	if !exist {
		t.Error("Get from cache failed")
		return
	}
	time.Sleep(1 * time.Millisecond)
	if *current != 10000 {
		t.Errorf("Value not equal after IncrInt 10000, current=%d", *current)
		return
	}
}

func TestIncrement32WithSharpRace(t *testing.T) {
	c := NewCache[*int32](5 * time.Second)
	for i := 0; i < 10000; i++ {
		go c.IncrInt("yhm", int32(1))
	}
	current, exist := c.Get("yhm")
	if !exist {
		t.Error("Get from cache failed")
		return
	}
	time.Sleep(1 * time.Millisecond)
	if *current != 10000 {
		t.Errorf("Value not equal after IncrInt 10000, current=%d", *current)
		return
	}
}

func TestDecrement64(t *testing.T) {
	c := NewCache[*int64](time.Second)
	current := c.DecrInt("yhm", int64(1))
	if *current != -1 {
		t.Errorf("Value not equal after DecrInt without initialize, current=%d", current)
		return
	}
	current = c.DecrInt("yhm", int64(11))
	if *current != -12 {
		t.Errorf("Value not equal after DecrInt with initialize, current=%d", current)
		return
	}
	current, _ = c.Get("yhm")
	if *current != -12 {
		t.Errorf("Value not equal after get, current=%d", current)
	}
	t.Logf("%s test success current=%d\n", GetFuncName(TestDecrement64), *current)
}

func TestDecrement32(t *testing.T) {
	c := NewCache[*int32](time.Second)
	current := c.DecrInt("yhm", int32(1))
	if *current != -1 {
		t.Errorf("Value not equal after DecrInt without initialize, current=%d", current)
		return
	}
	current = c.DecrInt("yhm", int32(11))
	if *current != -12 {
		t.Errorf("Value not equal after DecrInt with initialize, current=%d", current)
		return
	}
	current, _ = c.Get("yhm")
	if *current != -12 {
		t.Errorf("Value not equal after get, current=%d", current)
	}
	t.Logf("%s test success current=%d\n", GetFuncName(TestDecrement32), *current)
}

func TestDecrement64WithSharpRace(t *testing.T) {
	c := NewCache[*int64](5 * time.Second)
	for i := 0; i < 10000; i++ {
		go c.DecrInt("yhm", int64(1))
	}
	current, exist := c.Get("yhm")
	if !exist {
		t.Error("Get from cache failed")
		return
	}
	time.Sleep(1 * time.Millisecond)
	if *current != -10000 {
		t.Errorf("Value not equal after DecrInt -10000, current=%d", *current)
		return
	}
}

func TestDecrement32WithSharpRace(t *testing.T) {
	c := NewCache[*int32](5 * time.Second)
	for i := 0; i < 10000; i++ {
		go c.DecrInt("yhm", int32(1))
	}
	current, exist := c.Get("yhm")
	if !exist {
		t.Error("Get from cache failed")
		return
	}
	time.Sleep(1 * time.Millisecond)
	if *current != -10000 {
		t.Errorf("Value not equal after DecrInt -10000, current=%d", *current)
		return
	}
}

func TestInsertAndGetPointer(t *testing.T) {
	c := NewCache[*ComparableS](time.Second)
	v := ComparableS{Id: 1, Name: "yhm"}
	c.LoadOrStore("yhm", &v)
	value, ok := c.Get("yhm")
	if !ok {
		t.Error("Get from cache failed")
		return
	}
	if &v != value {
		t.Error("Value not equal")
		return
	}
	t.Logf("%s test success \n", GetFuncName(TestInsertAndGetPointer))
}

func TestInsertAndGetStruct(t *testing.T) {
	c := NewCache[ComparableS](time.Second)
	v := ComparableS{Id: 1}
	c.LoadOrStore("yhm", v)
	value, ok := c.Get("yhm")
	if !ok {
		t.Error("Get from cache failed")
		return
	}
	if v != value {
		t.Error("Value not equal")
		return
	}
	t.Logf("%s test success \n", GetFuncName(TestInsertAndGetStruct))
}

func TestClear(t *testing.T) {
	c := NewCache[int](20 * time.Second)
	c.LoadOrStore("yhm", 1)
	value, ok := c.Get("yhm")
	if !ok {
		t.Error("Get from cache failed")
		return
	}
	if value != 1 {
		t.Error("Value not equal")
		return
	}
	c.Clear()
	num, ok := c.Get("yhm")
	if ok {
		t.Errorf("Get from cache success after flush,but it should not, num=%d", num)
		return
	}
	t.Logf("%s test success \n", GetFuncName(TestClear))
}

func TestEviction(t *testing.T) {
	success := make(chan bool, 1)
	c := NewCacheWithEviction[int](1*time.Second, func(key string, value int) {
		t.Logf("trigger eviction")
		success <- true
	})
	c.LoadOrStore("yhm", 1)
	value, ok := c.Get("yhm")
	if !ok {
		t.Error("Get from cache failed")
		return
	}
	if value != 1 {
		t.Error("Value not equal")
		return
	}
	c.Delete("yhm")
	time.Sleep(10 * time.Millisecond)
	select {
	case <-success:
		t.Logf("%s test success \n", GetFuncName(TestEviction))
	default:
		t.Errorf("Eviction not triggered")
		return
	}
}

func TestSize(t *testing.T) {
	c := NewCache[int](time.Second)
	c.LoadOrStore("yhm", 1)
	if c.Size() != 1 {
		t.Errorf("Size not equal after LoadOrStore, size=%d", c.Size())
		return
	}
	c.Delete("yhm")
	if c.Size() != 0 {
		t.Errorf("Size not equal after Delete, size=%d", c.Size())
		return
	}
	t.Logf("%s test success \n", GetFuncName(TestSize))
}

func TestConcurrentAccess(t *testing.T) {
	c := NewCache[int64](1 * time.Second)
	c.Set("yhm", 12, 1*time.Second)
	var wg sync.WaitGroup
	wg.Add(1000000)
	for i := 1; i <= 1000000; i++ {
		go func() {
			current, _ := c.Get("yhm")
			if current != 12 {
				t.Errorf("Value not equal after concurrent Get, current=%d", current)
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestConcurrentLoadOrStore(t *testing.T) {
	defer atomic.StoreInt64(&loads, 0)
	c := NewCache[int64](1 * time.Second)
	for i := 1; i < 10000; i++ {
		j := int64(i)
		go func() {
			current, _ := c.LoadOrStore("yhm", j)
			if atomic.LoadInt64(&loads) == 0 {
				atomic.CompareAndSwapInt64(&loads, 0, current)
			}
			if atomic.LoadInt64(&loads) != current {
				t.Errorf("Value not equal after concurrent LoadOrStore, current=%d", current)
			}
		}()
	}
	t.Logf("%s test success loads=%d \n", GetFuncName(TestConcurrentLoadOrStore), loads)
}
