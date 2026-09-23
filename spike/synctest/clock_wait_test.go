package synctestspike_test

import (
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

// synctest gives a fake clock that advances only when the bubble is durably blocked.
func TestSpike_FakeClockAdvancesOnDurableBlock(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		done := make(chan struct{})
		go func() {
			time.Sleep(5 * time.Second)
			close(done)
		}()
		<-done
		if got := time.Since(start); got != 5*time.Second {
			t.Fatalf("fake clock: want 5s, got %v", got)
		}
	})
}

// Wait returns when every other goroutine is durably blocked (or finished).
func TestSpike_WaitSeesBackgroundCompletion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var mu sync.Mutex
		done := false
		go func() {
			mu.Lock()
			done = true
			mu.Unlock()
		}()
		synctest.Wait()
		mu.Lock()
		ok := done
		mu.Unlock()
		if !ok {
			t.Fatal("Wait returned before background goroutine finished")
		}
	})
}
