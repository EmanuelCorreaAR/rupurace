package scheduler

import (
	"fmt"
	"math/big"
	"runtime"
	"time"

	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
	"github.com/EmanuelCorreaAR/rupurace/internal/witness"
)

// Fingerprint returns a deterministic identity for a state.
// Built-in scenarios use %#v; later scenarios may specialize.
func Fingerprint(state scenario.State) string {
	return fmt.Sprintf("%T:%#v", state, state)
}

// Multinomial returns n! / (k1! k2! … km!) where n = sum(ki).
// Used as PossibleSchedules when each worker has a fixed step count.
// Returns 0 if the value does not fit in uint64.
func Multinomial(counts []int) uint64 {
	n := 0
	for _, k := range counts {
		if k < 0 {
			return 0
		}
		n += k
	}
	num := factorial(n)
	den := big.NewInt(1)
	for _, k := range counts {
		den.Mul(den, factorial(k))
	}
	if den.Sign() == 0 {
		return 0
	}
	num.Div(num, den)
	if !num.IsUint64() {
		return 0
	}
	return num.Uint64()
}

func factorial(n int) *big.Int {
	out := big.NewInt(1)
	for i := 2; i <= n; i++ {
		out.Mul(out, big.NewInt(int64(i)))
	}
	return out
}

type exploreMeter struct {
	stats      witness.Stats
	seen       map[string]struct{}
	memBefore  runtime.MemStats
	started    time.Time
	collectMem bool
}

func newMeter(collectMem bool) *exploreMeter {
	m := &exploreMeter{
		seen:       make(map[string]struct{}),
		started:    time.Now(),
		collectMem: collectMem,
	}
	if collectMem {
		runtime.GC()
		runtime.ReadMemStats(&m.memBefore)
	}
	return m
}

func (m *exploreMeter) visit(state scenario.State) {
	m.stats.StatesVisited++
	fp := Fingerprint(state)
	if _, ok := m.seen[fp]; !ok {
		m.seen[fp] = struct{}{}
		m.stats.UniqueStates++
	}
}

func (m *exploreMeter) finish() *witness.Stats {
	m.stats.DurationNanos = time.Since(m.started).Nanoseconds()
	if m.collectMem {
		var after runtime.MemStats
		runtime.ReadMemStats(&after)
		if after.TotalAlloc >= m.memBefore.TotalAlloc {
			m.stats.HeapBytes = after.TotalAlloc - m.memBefore.TotalAlloc
		}
	}
	out := m.stats
	return &out
}
