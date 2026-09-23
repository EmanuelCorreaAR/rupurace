package scenario_test

import (
	"fmt"
	"testing"

	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
)

func TestScenarios_EnabledApplyContract(t *testing.T) {
	cases := []struct {
		name string
		sc   scenario.Scenario
	}{
		{"double-withdraw", scenario.DoubleWithdraw{Balance: 100, Amount: 60}},
		{"lost-update", scenario.LostUpdate{}},
		{"check-then-act", scenario.CheckThenAct{}},
		{"init-ordering", scenario.InitOrdering{Expected: 7}},
		{"interleave", scenario.Interleave{Workers: 2, StepsPerWorker: 2}},
		{"counter", scenario.Counter{QuotaA: 2, QuotaB: 2}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := tc.sc.Initial()
			enabled := tc.sc.Enabled(st)
			if len(enabled) == 0 {
				t.Fatal("initial state must enable at least one transition")
			}
			for _, tr := range enabled {
				if tr.Step < 0 {
					t.Fatalf("negative step: %s", tr)
				}
			}

			before := fmt.Sprintf("%#v", st)
			_ = tc.sc.Apply(st, enabled[0])
			after := fmt.Sprintf("%#v", st)
			if before != after {
				t.Fatal("Apply must not mutate the prior state value")
			}

			// Every enabled transition must be accepted (no panic).
			for _, tr := range enabled {
				_ = tc.sc.Apply(st, tr)
			}
		})
	}
}

func TestTransition_String(t *testing.T) {
	tr := scenario.Transition{Worker: "A", Step: 2}
	if got := tr.String(); got != "A:2" {
		t.Fatalf("got %q", got)
	}
}
