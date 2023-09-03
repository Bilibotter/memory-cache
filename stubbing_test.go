package memory_cache

import (
	"testing"
	"time"
)

func TestOnce(t *testing.T) {
	defer RestStubbing()
	for i := 0; i < 1000; i++ {
		NewStubCacheWithEviction[int](1*time.Second, nil)
	}
	RestStubbing()
	time.Sleep(1100 * time.Millisecond)
	param := GetStubbingParam()
	if param.SleepTimes >= 1000 {
		t.Errorf("SleepTimes did not meet expectations, SleepTimes=%d", param.SleepTimes)
	}
	t.Logf("1. %#v\n", param)
}

func TestWakeUp(t *testing.T) {
	defer RestStubbing()
	NewStubCacheWithEviction[int](6*time.Second, nil)
	time.Sleep(100 * time.Millisecond)
	NewStubCacheWithEviction[int](3*time.Second, nil)
	time.Sleep(100 * time.Millisecond)
	NewStubCacheWithEviction[int](1*time.Second, nil)
	param := GetStubbingParam()
	t.Logf("1. %#v\n", param)
	if param.WakeupTimes < 2 {
		t.Errorf("WakeupTimes did not meet expectations, WakeupTimes=%d", param.WakeupTimes)
	}
}

func TestMultipleGetWhenExpired(t *testing.T) {
	defer RestStubbing()
	c := NewStubCacheWithEviction[int](time.Second, nil)
	c.Set("yhm", 1, time.Second)
	t.Logf("1. %#v\n", GetStubbingParam())
	time.Sleep(1100 * time.Millisecond)
	t.Logf("2. %#v\n", GetStubbingParam())
	for i := 0; i < 100; i++ {
		go func() {
			value, exist := c.Get("yhm")
			if exist {
				t.Errorf("Get expierd key success, but expected failed.value=%d", value)
				t.Fail()
			}
		}()
	}
	time.Sleep(3 * time.Second)
	t.Logf("3. %#v\n", GetStubbingParam())
	param := GetStubbingParam()
	if param.FinishWait != 100 {
		t.Errorf("FinishDeletedid not meet expectations, FinishDelete=%d", GetStubbingParam().WaitDelete)
	}
	if param.WaitDelete != 100 {
		t.Errorf("WaitDelete did not meet expectations, WaitDelete=%d", GetStubbingParam().WaitDelete)
	}
	if param.FinishDelete != 1 {
		t.Errorf("FinishDelete did not meet expectations, FinishDelete=%d", GetStubbingParam().FinishDelete)
	}
	if param.CleanCycle < 2 || param.CleanTimes < 2 || param.SleepTimes < 2 {
		t.Errorf("clean goroutine cycle did not meet expectations, CleanCycle=%d, CleanTimes=%d, SleepTimes=%d", param.CleanCycle, param.CleanTimes, param.SleepTimes)
	}
}
