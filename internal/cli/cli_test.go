package cli_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/EmanuelCorreaAR/rupurace/internal/cli"
)

func TestHeart_FindWitnessReplay(t *testing.T) {
	dir := t.TempDir()
	witnessPath := filepath.Join(dir, "witness.json")

	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"explore",
		"--scenario", "double-withdraw",
		"--strategy", "exhaustive",
		"-o", witnessPath,
	}, &stdout, &stderr)
	if code != cli.ExitOK {
		t.Fatalf("explore exit %d stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("violation")) {
		t.Fatalf("expected violation output: %s", stderr.String())
	}

	raw, err := os.ReadFile(witnessPath)
	if err != nil {
		t.Fatal(err)
	}
	var wf map[string]any
	if err := json.Unmarshal(raw, &wf); err != nil {
		t.Fatal(err)
	}
	if wf["scenario"] != "double-withdraw" {
		t.Fatalf("scenario: %v", wf["scenario"])
	}
	if wf["invariant"] != "balance >= 0" {
		t.Fatalf("invariant: %v", wf["invariant"])
	}
	if _, ok := wf["failedAt"].(float64); !ok {
		t.Fatalf("failedAt missing: %+v", wf)
	}
	sched, ok := wf["schedule"].([]any)
	if !ok || len(sched) == 0 {
		t.Fatalf("schedule: %+v", wf["schedule"])
	}

	stdout.Reset()
	stderr.Reset()
	code = cli.Run([]string{
		"replay", witnessPath, "--fail-on-violation",
	}, &stdout, &stderr)
	if code != cli.ExitGate {
		t.Fatalf("replay want exit 2, got %d stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("balance >= 0")) {
		t.Fatalf("replay should mention invariant: %s", stderr.String())
	}

	// Same failure every time.
	for i := 0; i < 10; i++ {
		var out, errBuf bytes.Buffer
		c := cli.Run([]string{"replay", witnessPath, "--fail-on-violation"}, &out, &errBuf)
		if c != cli.ExitGate {
			t.Fatalf("replay %d exit %d", i+1, c)
		}
	}
}

func TestExplore_JSONDeterministic(t *testing.T) {
	args := []string{"explore", "--json", "--strategy", "exhaustive"}
	var first string
	for i := 0; i < 5; i++ {
		var stdout, stderr bytes.Buffer
		code := cli.Run(args, &stdout, &stderr)
		if code != cli.ExitOK {
			t.Fatalf("exit %d stderr=%s", code, stderr.String())
		}
		got := stdout.String()
		if i == 0 {
			first = got
			continue
		}
		if got != first {
			t.Fatalf("JSON diverged on run %d", i+1)
		}
	}
}

func TestExplore_AllFailingScenarios(t *testing.T) {
	for _, name := range []string{"double-withdraw", "lost-update", "check-then-act", "init-ordering"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			witnessPath := filepath.Join(dir, "witness.json")
			var stdout, stderr bytes.Buffer
			code := cli.Run([]string{
				"explore", "--scenario", name, "-o", witnessPath, "--fail-on-violation",
			}, &stdout, &stderr)
			if code != cli.ExitGate {
				t.Fatalf("explore want exit 2, got %d stderr=%s", code, stderr.String())
			}
			stdout.Reset()
			stderr.Reset()
			code = cli.Run([]string{"replay", witnessPath, "--fail-on-violation"}, &stdout, &stderr)
			if code != cli.ExitGate {
				t.Fatalf("replay want exit 2, got %d stderr=%s", code, stderr.String())
			}
		})
	}
}

func TestHelpAndVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := cli.Run(nil, &stdout, &stderr); code != cli.ExitOK {
		t.Fatalf("no-args help exit %d", code)
	}
	stdout.Reset()
	if code := cli.Run([]string{"version"}, &stdout, &stderr); code != cli.ExitOK {
		t.Fatalf("version exit %d", code)
	}
}

func TestExplore_InterleaveStats(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"explore", "--scenario", "interleave",
		"--workers", "2", "--steps", "3",
		"--stats",
	}, &stdout, &stderr)
	if code != cli.ExitOK {
		t.Fatalf("exit %d stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Schedules considered: 20")) {
		t.Fatalf("missing multinomial stats: %s", stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Schedules executed:   20")) {
		t.Fatalf("missing executed stats: %s", stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Equivalent pruned:    0")) {
		t.Fatalf("expected pruned=0 before hashing: %s", stderr.String())
	}
}

func TestExplore_InterleavePrune(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := cli.Run([]string{
		"explore", "--scenario", "interleave",
		"--workers", "2", "--steps", "5",
		"--stats", "--prune",
	}, &stdout, &stderr)
	if code != cli.ExitOK {
		t.Fatalf("exit %d stderr=%s", code, stderr.String())
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Schedules considered: 252")) {
		t.Fatalf("missing considered: %s", stderr.String())
	}
	if bytes.Contains(stderr.Bytes(), []byte("Equivalent pruned:    0")) {
		t.Fatalf("expected Equivalent pruned > 0: %s", stderr.String())
	}
}
