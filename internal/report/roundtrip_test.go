package report_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/EmanuelCorreaAR/rupurace/internal/demos"
	"github.com/EmanuelCorreaAR/rupurace/internal/report"
	"github.com/EmanuelCorreaAR/rupurace/internal/scheduler"
)

// TestWitnessJSONRoundTrip: witness → JSON → witness' preserves product semantics.
func TestWitnessJSONRoundTrip(t *testing.T) {
	for _, name := range []string{
		"double-withdraw",
		"lost-update",
		"check-then-act",
		"init-ordering",
	} {
		t.Run(name, func(t *testing.T) {
			loaded, err := demos.Load(name, nil)
			if err != nil {
				t.Fatal(err)
			}
			found := scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, scheduler.ExhaustiveConfig{})
			if found.Witness == nil {
				t.Fatal("expected witness")
			}

			original := report.NewWitnessFile(loaded.Name, loaded.Params, *found.Witness)

			var buf bytes.Buffer
			enc := json.NewEncoder(&buf)
			enc.SetEscapeHTML(false)
			enc.SetIndent("", "  ")
			if err := enc.Encode(original); err != nil {
				t.Fatal(err)
			}

			var restored report.File
			if err := json.Unmarshal(buf.Bytes(), &restored); err != nil {
				t.Fatal(err)
			}

			if restored.Kind != report.WitnessKind {
				t.Fatalf("kind: %q", restored.Kind)
			}
			if restored.Scenario != original.Scenario {
				t.Fatalf("scenario: %q vs %q", original.Scenario, restored.Scenario)
			}
			if restored.Invariant != original.Invariant {
				t.Fatalf("invariant: %q vs %q", original.Invariant, restored.Invariant)
			}
			if restored.FailedAt != original.FailedAt {
				t.Fatalf("failedAt: %d vs %d", original.FailedAt, restored.FailedAt)
			}
			if len(restored.Schedule) != len(original.Schedule) {
				t.Fatalf("schedule len mismatch")
			}
			for i := range original.Schedule {
				if restored.Schedule[i] != original.Schedule[i] {
					t.Fatalf("schedule[%d]: %+v vs %+v", i, original.Schedule[i], restored.Schedule[i])
				}
			}

			// Schedule codec is lossless for the engine types.
			sched := report.ToSchedule(restored.Schedule)
			again := report.FromSchedule(sched)
			for i := range again {
				if again[i] != restored.Schedule[i] {
					t.Fatalf("schedule codec lost data at %d", i)
				}
			}
		})
	}
}

func TestWitnessKindIsVersioned(t *testing.T) {
	if report.WitnessKind != "witness_v1" {
		t.Fatalf("unexpected kind %q — bump consciously when the format changes", report.WitnessKind)
	}
}
