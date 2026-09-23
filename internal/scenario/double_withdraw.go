package scenario

// DoubleWithdraw models a classic lost-update withdraw race with cooperative steps.
//
// Two workers, three steps each, one shared balance:
//
//	0 read   — copy balance into a private local
//	1 debit  — subtract Amount from the local copy
//	2 write  — store the local copy back into balance
//
// With Balance=100 and Amount=60, some interleavings leave balance < 0
// (both workers read 100 before either writes).
type DoubleWithdraw struct {
	Balance int
	Amount  int
}

// DoubleWithdrawState is the snapshot of a DoubleWithdraw run.
type DoubleWithdrawState struct {
	Balance int
	LocalA  int
	LocalB  int
	NextA   int // next step index for A (0..3; 3 = done)
	NextB   int
}

// Initial implements Scenario.
func (d DoubleWithdraw) Initial() State {
	return DoubleWithdrawState{Balance: d.Balance}
}

// Enabled implements Scenario.
func (d DoubleWithdraw) Enabled(st State) []Transition {
	s := st.(DoubleWithdrawState)
	var out []Transition
	if s.NextA < 3 {
		out = append(out, Transition{Worker: "A", Step: s.NextA})
	}
	if s.NextB < 3 {
		out = append(out, Transition{Worker: "B", Step: s.NextB})
	}
	return out
}

// Apply implements Scenario.
func (d DoubleWithdraw) Apply(st State, t Transition) State {
	s := st.(DoubleWithdrawState)
	switch t.Worker {
	case "A":
		s = applyDoubleStep(s, t.Step, d.Amount, true)
	case "B":
		s = applyDoubleStep(s, t.Step, d.Amount, false)
	default:
		panic("scenario: unknown worker " + string(t.Worker))
	}
	return s
}

func applyDoubleStep(s DoubleWithdrawState, step, amount int, isA bool) DoubleWithdrawState {
	var local *int
	var next *int
	if isA {
		local, next = &s.LocalA, &s.NextA
	} else {
		local, next = &s.LocalB, &s.NextB
	}
	switch step {
	case 0:
		*local = s.Balance
	case 1:
		*local = *local - amount
	case 2:
		s.Balance = *local
	default:
		panic("scenario: unknown step")
	}
	*next = step + 1
	return s
}
