package scenario

// Counter is a tiny abstract concurrent program used by engine tests:
// two actors each increment a shared counter up to their quota.
type Counter struct {
	QuotaA int
	QuotaB int
}

// CounterState is the mutable snapshot of a Counter run.
type CounterState struct {
	ADone int
	BDone int
	Value int
}

// Initial implements Scenario.
func (c Counter) Initial() State {
	return CounterState{}
}

// Enabled implements Scenario.
func (c Counter) Enabled(st State) []Transition {
	cs := st.(CounterState)
	var out []Transition
	if cs.ADone < c.QuotaA {
		out = append(out, Transition{Actor: "A", Step: "inc"})
	}
	if cs.BDone < c.QuotaB {
		out = append(out, Transition{Actor: "B", Step: "inc"})
	}
	return out
}

// Apply implements Scenario.
func (c Counter) Apply(st State, t Transition) State {
	cs := st.(CounterState)
	switch t.Actor {
	case "A":
		cs.ADone++
		cs.Value++
	case "B":
		cs.BDone++
		cs.Value++
	default:
		panic("scenario: unknown actor " + string(t.Actor))
	}
	return cs
}
