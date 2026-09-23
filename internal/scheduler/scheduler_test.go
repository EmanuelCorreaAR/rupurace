package scheduler_test

import (
	"testing"

	"github.com/EmanuelCorreaAR/rupurace/internal/demos"
	"github.com/EmanuelCorreaAR/rupurace/internal/scheduler"
)

func TestExhaustive_DoubleWithdraw_InvariantName(t *testing.T) {
	loaded, err := demos.Load("double-withdraw", map[string]any{
		"balance": 100,
		"amount":  60,
	})
	if err != nil {
		t.Fatal(err)
	}

	found := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
	if found.Passed || found.Witness == nil {
		t.Fatalf("expected a violation, got %+v", found)
	}
	if found.Witness.Invariant != "balance >= 0" {
		t.Fatalf("invariant: %q", found.Witness.Invariant)
	}
	if found.Witness.FailedAt != len(found.Witness.Schedule) {
		t.Fatalf("failedAt=%d len=%d", found.Witness.FailedAt, len(found.Witness.Schedule))
	}
}
