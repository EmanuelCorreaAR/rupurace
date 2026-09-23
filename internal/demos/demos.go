// Package demos provides built-in abstract scenarios for the CLI.
package demos

import (
	"errors"
	"fmt"

	"github.com/EmanuelCorreaAR/rupurace/internal/invariant"
	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
)

// Loaded is a built-in scenario ready for explore/replay.
type Loaded struct {
	Name   string
	Params map[string]any
	Scene  scenario.Scenario
	Specs  []invariant.Spec
}

// Load builds a built-in scenario by name and params.
func Load(name string, params map[string]any) (Loaded, error) {
	switch name {
	case "double-withdraw", "double_withdraw":
		return loadDoubleWithdraw(params)
	case "counter":
		return loadCounter(params)
	default:
		return Loaded{}, fmt.Errorf("unsupported scenario %q (double-withdraw, counter)", name)
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
