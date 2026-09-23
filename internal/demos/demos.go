// Package demos provides built-in abstract scenarios for the CLI.
package demos

import (
	"errors"
	"fmt"

	"github.com/EmanuelCorreaAR/rupurace/internal/invariant"
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
)

// Names lists built-in scenarios that exercise the cooperative model.
var Names = []string{
	"double-withdraw",
	"lost-update",
	"check-then-act",
	"init-ordering",
	"counter",
}

// Loaded is a built-in scenario ready for explore/replay.
type Loaded struct {
	Name   string
	Params map[string]any
	Scene  scenario.Scenario
	Specs  []invariant.Spec
}

// Load builds a built-in scenario by name and params.
func Load(name string, params map[string]any) (Loaded, error) {
	if params == nil {
		params = map[string]any{}
	}
	switch name {
	case "double-withdraw", "double_withdraw":
		return loadDoubleWithdraw(params)
	case "lost-update", "lost_update":
		return loadLostUpdate()
	case "check-then-act", "check_then_act":
		return loadCheckThenAct()
	case "init-ordering", "init_ordering":
		return loadInitOrdering(params)
	case "counter":
		return loadCounter(params)
	default:
		return Loaded{}, fmt.Errorf("unsupported scenario %q (see: %v)", name, Names)
	}
}

// LoadFromWitness rebuilds a scenario from a witness file.
func LoadFromWitness(scenarioName string, params map[string]any) (Loaded, error) {
	return Load(scenarioName, params)
}

func loadDoubleWithdraw(params map[string]any) (Loaded, error) {
	p := DoubleWithdrawParams{
		Balance: intFrom(params["balance"], 100),
		Amount:  intFrom(params["amount"], 60),
	}
	return Loaded{
		Name:   "double-withdraw",
		Params: DoubleWithdrawParamsMap(p),
		Scene:  scenario.DoubleWithdraw{Balance: p.Balance, Amount: p.Amount},
		Specs:  []invariant.Spec{DoubleWithdrawInvariant()},
	}, nil
}

func loadLostUpdate() (Loaded, error) {
	return Loaded{
		Name:   "lost-update",
		Params: map[string]any{},
		Scene:  scenario.LostUpdate{},
		Specs:  []invariant.Spec{LostUpdateInvariant()},
	}, nil
}

func loadCheckThenAct() (Loaded, error) {
	return Loaded{
		Name:   "check-then-act",
		Params: map[string]any{},
		Scene:  scenario.CheckThenAct{},
		Specs:  []invariant.Spec{CheckThenActInvariant()},
	}, nil
}

func loadInitOrdering(params map[string]any) (Loaded, error) {
	expected := intFrom(params["expected"], 7)
	return Loaded{
		Name:   "init-ordering",
		Params: map[string]any{"expected": expected},
		Scene:  scenario.InitOrdering{Expected: expected},
		Specs:  []invariant.Spec{InitOrderingInvariant(expected)},
	}, nil
}

func loadCounter(params map[string]any) (Loaded, error) {
	p := CounterParams{
		QuotaA:   intFrom(params["quota_a"], 3),
		QuotaB:   intFrom(params["quota_b"], 3),
		MaxValue: intFrom(params["max_value"], 100),
	}
	return Loaded{
		Name:   "counter",
		Params: CounterParamsMap(p),
		Scene:  scenario.Counter{QuotaA: p.QuotaA, QuotaB: p.QuotaB},
		Specs:  []invariant.Spec{CounterInvariant(p)},
	}, nil
}

// DoubleWithdrawParams configures the classic withdraw race demo.
type DoubleWithdrawParams struct {
	Balance int
	Amount  int
}

// DoubleWithdrawParamsMap is the portable params block.
func DoubleWithdrawParamsMap(p DoubleWithdrawParams) map[string]any {
	return map[string]any{
		"balance": p.Balance,
		"amount":  p.Amount,
	}
}

// DoubleWithdrawInvariant is "balance >= 0".
func DoubleWithdrawInvariant() invariant.Spec {
	return invariant.Spec{
		Name: "balance >= 0",
		Hold: func(st scenario.State) error {
			s := st.(scenario.DoubleWithdrawState)
			if s.Balance < 0 {
				return errors.New("balance >= 0")
			}
			return nil
		},
	}
}

// LostUpdateInvariant fails when both workers finished and Value != 2.
func LostUpdateInvariant() invariant.Spec {
	return invariant.Spec{
		Name: "value == 2",
		Hold: func(st scenario.State) error {
			s := st.(scenario.LostUpdateState)
			if s.Done() && s.Value != 2 {
				return errors.New("value == 2")
			}
			return nil
		},
	}
}

// CheckThenActInvariant is "owners <= 1".
func CheckThenActInvariant() invariant.Spec {
	return invariant.Spec{
		Name: "owners <= 1",
		Hold: func(st scenario.State) error {
			s := st.(scenario.CheckThenActState)
			if s.Owners > 1 {
				return errors.New("owners <= 1")
			}
			return nil
		},
	}
}

// InitOrderingInvariant is "got == 0 || got == expected".
func InitOrderingInvariant(expected int) invariant.Spec {
	name := fmt.Sprintf("got == 0 || got == %d", expected)
	return invariant.Spec{
		Name: name,
		Hold: func(st scenario.State) error {
			s := st.(scenario.InitOrderingState)
			if s.Got != 0 && s.Got != expected {
				return errors.New(name)
			}
			// Catch the zero-payload publish race: observed publish but got stale zero
			// after subscriber finished reading.
			if s.NextS >= 2 && s.LocalS && s.Got != expected {
				return errors.New(name)
			}
			return nil
		},
	}
}

// CounterParams configures the counter demo.
type CounterParams struct {
	QuotaA   int
	QuotaB   int
	MaxValue int
}

// CounterParamsMap is the portable params block.
func CounterParamsMap(p CounterParams) map[string]any {
	return map[string]any{
		"quota_a":   p.QuotaA,
		"quota_b":   p.QuotaB,
		"max_value": p.MaxValue,
	}
}

// CounterInvariant fails when the shared counter exceeds MaxValue.
func CounterInvariant(p CounterParams) invariant.Spec {
	limit := p.MaxValue
	return invariant.Spec{
		Name: "counter <= max",
		Hold: func(st scenario.State) error {
			cs := st.(scenario.CounterState)
			if cs.Value > limit {
				return errors.New("counter exceeded limit")
			}
			return nil
		},
	}
}

func intFrom(v any, fallback int) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return fallback
	}
}
