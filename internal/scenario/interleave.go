package scenario

import "fmt"

// Interleave is a synthetic scenario for measuring combinatorial explosion:
// W workers × S cooperative steps each, no shared bug by default.
//
// Full interleavings = (W*S)! / (S!)^W.
type Interleave struct {
	Workers        int
	StepsPerWorker int
}

// InterleaveState tracks each worker's next step and a global tick.
type InterleaveState struct {
	Next []int
	Tick int
}

// Initial implements Scenario.
func (in Interleave) Initial() State {
	next := make([]int, in.Workers)
	return InterleaveState{Next: next}
}

// Enabled implements Scenario.
func (in Interleave) Enabled(st State) []Transition {
	s := st.(InterleaveState)
	var out []Transition
	for i := 0; i < in.Workers; i++ {
		if s.Next[i] < in.StepsPerWorker {
			out = append(out, Transition{
				Worker: WorkerID(workerName(i)),
				Step:   s.Next[i],
			})
		}
	}
	return out
}

// Apply implements Scenario.
func (in Interleave) Apply(st State, t Transition) State {
	s := st.(InterleaveState)
	next := make([]int, len(s.Next))
	copy(next, s.Next)
	idx := workerIndex(string(t.Worker))
	if idx < 0 || idx >= len(next) {
		panic("scenario: unknown worker " + string(t.Worker))
	}
	if t.Step != next[idx] {
		panic(fmt.Sprintf("scenario: unexpected step %d want %d", t.Step, next[idx]))
	}
	next[idx]++
	return InterleaveState{Next: next, Tick: s.Tick + 1}
}

func workerName(i int) string {
	if i < 26 {
		return string(rune('A' + i))
	}
	return fmt.Sprintf("W%d", i)
}

func workerIndex(name string) int {
	if len(name) == 1 && name[0] >= 'A' && name[0] <= 'Z' {
		return int(name[0] - 'A')
	}
	var n int
	if _, err := fmt.Sscanf(name, "W%d", &n); err == nil {
		return n
	}
	return -1
}
