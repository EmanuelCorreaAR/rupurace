package witness_test

import (
	"testing"

	"github.com/EmanuelCorreaAR/rupurace/internal/scenario"
	"github.com/EmanuelCorreaAR/rupurace/internal/witness"
)

func TestWitness_FieldsAreTheProductContract(t *testing.T) {
	w := witness.Witness{
		Schedule: scenario.Schedule{
			{Worker: "A", Step: 0},
			{Worker: "B", Step: 0},
		},
		Invariant: "owners <= 1",
		Violation: "owners <= 1",
		FailedAt:  2,
	}
	if w.FailedAt != len(w.Schedule) {
		t.Fatalf("failedAt should match schedule length for a just-failed check")
	}
	if w.Invariant == "" || len(w.Schedule) == 0 {
		t.Fatal("witness requires invariant + schedule")
	}
}

func TestResult_CarriesStrategy(t *testing.T) {
	r := witness.Result{Passed: false, Strategy: "exhaustive", Steps: 4}
	if r.Strategy != "exhaustive" {
		t.Fatalf("strategy: %q", r.Strategy)
	}
}
