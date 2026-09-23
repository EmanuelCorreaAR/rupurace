package synctestspike_test

import (
	"sync"
	"testing"
	"testing/synctest"
)

// synctest does not expose "run goroutine X next". Across bubbles we may see
// more than one winner, but we cannot request a specific interleaving or
// enumerate the schedule space the way RupuRace Explore does.
func TestSpike_NoScheduleControlAPI(t *testing.T) {
	type outcome struct {
		first string
	}
	seen := map[outcome]int{}

	for i := 0; i < 80; i++ {
		synctest.Test(t, func(t *testing.T) {
			var (
				mu    sync.Mutex
				order []string
				wg    sync.WaitGroup
			)
			claim := func(name string) {
				defer wg.Done()
				mu.Lock()
				defer mu.Unlock()
				order = append(order, name)
			}
			wg.Add(2)
			go claim("A")
			go claim("B")
			wg.Wait()
			seen[outcome{first: order[0]}]++
		})
	}

	t.Logf("observed first-claimer distribution: %#v", seen)
	// Even if both appear, that is noise — not controllable exploration.
	if len(seen) == 0 {
		t.Fatal("no outcomes")
	}
}

// Mutex.Lock is not durably blocking (stdlib contract). A peer stuck on Lock
// prevents Wait from settling; the bubble deadlocks unless we Unlock first.
func TestSpike_MutexPreventsWaitSettlement(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var mu sync.Mutex
		mu.Lock()
		reached := make(chan struct{})
		go func() {
			close(reached)
			mu.Lock()
			mu.Unlock()
		}()
		<-reached // peer is now blocked on Mutex — not durable
		mu.Unlock()
		synctest.Wait() // settles only after Unlock unblocked the peer
	})
}
