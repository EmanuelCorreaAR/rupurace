// Package cli implements the rupurace command-line interface.
package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/EmanuelCorreaAR/rupurace/internal/audit"
	"github.com/EmanuelCorreaAR/rupurace/internal/demos"
	"github.com/EmanuelCorreaAR/rupurace/internal/replay"
	"github.com/EmanuelCorreaAR/rupurace/internal/report"
	"github.com/EmanuelCorreaAR/rupurace/internal/scheduler"
	"github.com/EmanuelCorreaAR/rupurace/internal/version"
	"github.com/EmanuelCorreaAR/rupurace/internal/witness"
)

const (
	ExitOK    = 0
	ExitError = 1
	ExitGate  = 2
)

var errHelp = errors.New("help")

// Run is the CLI entrypoint. Returns a process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return ExitOK
	}

	switch args[0] {
	case "-h", "--help", "help":
		printHelp(stdout)
		return ExitOK
	case "-V", "--version", "version":
		fmt.Fprintf(stdout, "rupurace %s\n", version.Version)
		return ExitOK
	case "explore":
		return runExplore(args[1:], stdout, stderr)
	case "replay":
		return runReplay(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "rupurace: unknown command %q\n", args[0])
		fmt.Fprintf(stderr, "Try 'rupurace --help' for usage.\n")
		return ExitError
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintf(w, `rupurace %s — explore schedules, keep the witness

Usage:
  rupurace <command> [flags]

Commands:
  explore   Explore schedules for a built-in cooperative scenario
  replay    Replay a witness schedule deterministically
  version   Show version and exit
  help      Show this message

Explore flags:
  --scenario name       double-withdraw (default) | lost-update |
                        check-then-act | init-ordering | interleave | counter
  --strategy name       exhaustive (default) | seeded
  --seed N              Seed for seeded strategy (default: 0)
  --max-steps N         Cap for seeded strategy (default: 64)
  --balance N           double-withdraw initial balance (default: 100)
  --amount N            double-withdraw debit amount (default: 60)
  --expected N          init-ordering payload (default: 7)
  --workers N           interleave workers (default: 2)
  --steps N             interleave steps per worker (default: 3)
  --quota-a/--quota-b   counter quotas (default: 3)
  --max-value N         counter invariant limit (default: 100)
  --stats               Collect exploration stats (space / states / time)
  --continue            Keep exploring after first violation (measurement)
  --measure-memory      Approximate alloc delta with --stats (slower)
  --fail-on-violation   Exit 2 when an invariant fails
  --json                Emit deterministic JSON audit envelope
  -o, --output path     Write witness.json on failure

Replay flags:
  <witness.json>        Witness artifact from explore -o
  --fail-on-violation   Exit 2 when the violation is reproduced
  --json                Emit deterministic JSON audit envelope

Exit codes:
  0  success
  1  usage or I/O error
  2  invariant violation with --fail-on-violation

`, version.Version)
}

type exploreOpts struct {
	scenario           string
	strategy           string
	seed               uint64
	maxSteps           int
	balance            int
	amount             int
	expected           int
	workers            int
	steps              int
	quotaA             int
	quotaB             int
	maxValue           int
	stats              bool
	contAfterViolation bool
	measureMemory      bool
	failOnViolation    bool
	json               bool
	output             string
}

func runExplore(args []string, stdout, stderr io.Writer) int {
	opts := exploreOpts{
		scenario: "double-withdraw",
		strategy: "exhaustive",
		maxSteps: 64,
		balance:  100,
		amount:   60,
		expected: 7,
		workers:  2,
		steps:    3,
		quotaA:   3,
		quotaB:   3,
		maxValue: 100,
	}
	rest, err := parseExploreFlags(args, &opts)
	if errors.Is(err, errHelp) {
		printHelp(stdout)
		return ExitOK
	}
	if err != nil {
		fmt.Fprintf(stderr, "rupurace explore: %v\n", err)
		return ExitError
	}
	if len(rest) > 0 {
		fmt.Fprintf(stderr, "rupurace explore: unexpected argument %q\n", rest[0])
		return ExitError
	}

	params := map[string]any{}
	switch opts.scenario {
	case "double-withdraw", "double_withdraw":
		opts.scenario = "double-withdraw"
		params["balance"] = opts.balance
		params["amount"] = opts.amount
	case "init-ordering", "init_ordering":
		opts.scenario = "init-ordering"
		params["expected"] = opts.expected
	case "interleave":
		params["workers"] = opts.workers
		params["steps"] = opts.steps
	case "counter":
		params["quota_a"] = opts.quotaA
		params["quota_b"] = opts.quotaB
		params["max_value"] = opts.maxValue
	case "lost-update", "lost_update", "check-then-act", "check_then_act":
		// no params
	}

	loaded, err := demos.Load(opts.scenario, params)
	if err != nil {
		fmt.Fprintf(stderr, "rupurace explore: %v\n", err)
		return ExitError
	}

	var result witness.Result
	switch opts.strategy {
	case "exhaustive":
		exCfg := scheduler.ExhaustiveConfig{
			ContinueAfterViolation: opts.contAfterViolation,
			Measure:                opts.stats,
			MeasureMemory:          opts.measureMemory,
		}
		if loaded.Name == "interleave" {
			exCfg.Workers = opts.workers
			exCfg.StepsPerWorker = opts.steps
		}
		result = scheduler.ExploreExhaustive(loaded.Scene, loaded.Specs, exCfg)
	case "seeded":
		result = scheduler.Explore(loaded.Scene, loaded.Specs, scheduler.Config{
			Seed:     opts.seed,
			MaxSteps: opts.maxSteps,
		})
	default:
		fmt.Fprintf(stderr, "rupurace explore: unknown strategy %q\n", opts.strategy)
		return ExitError
	}

	view := report.FromResult(result, loaded.Name, loaded.Params)
	input := map[string]any{
		"scenario": loaded.Name,
		"params":   loaded.Params,
	}
	configuration := map[string]any{
		"strategy":                 opts.strategy,
		"seed":                     opts.seed,
		"max_steps":                opts.maxSteps,
		"stats":                    opts.stats,
		"continue_after_violation": opts.contAfterViolation,
		"fail_on_violation":        opts.failOnViolation,
		"json":                     opts.json,
	}
	envelope := audit.Build("explore", input, configuration, view)

	if err := emit(envelope, opts.json, stdout); err != nil {
		fmt.Fprintf(stderr, "rupurace explore: %v\n", err)
		return ExitError
	}
	if !opts.json {
		printExploreHuman(stderr, result)
	}

	if result.Witness != nil && opts.output != "" {
		wf := report.NewWitnessFile(loaded.Name, loaded.Params, *result.Witness)
		if err := writeJSON(opts.output, wf); err != nil {
			fmt.Fprintf(stderr, "rupurace explore: write witness: %v\n", err)
			return ExitError
		}
		if !opts.json {
			fmt.Fprintf(stderr, "witness written to %s\n", opts.output)
		}
	}

	if !result.Passed && opts.failOnViolation {
		return ExitGate
	}
	return ExitOK
}

func printExploreHuman(w io.Writer, result witness.Result) {
	if result.Stats != nil {
		printStats(w, result.Stats)
	}
	if result.Passed {
		fmt.Fprintf(w, "passed — strategy=%s", result.Strategy)
		if result.Explored > 0 {
			fmt.Fprintf(w, " explored=%d", result.Explored)
		}
		fmt.Fprintf(w, "\n")
		return
	}
	fmt.Fprintf(w, "violation at transition %d: %s\n", result.Witness.FailedAt, result.Witness.Invariant)
	fmt.Fprintf(w, "schedule:\n")
	for i, t := range result.Witness.Schedule {
		fmt.Fprintf(w, "  %d. %s\n", i+1, t)
	}
}

func printStats(w io.Writer, s *witness.Stats) {
	fmt.Fprintf(w, "Exploration complete\n")
	fmt.Fprintf(w, "\n")
	if s.PossibleSchedules > 0 {
		fmt.Fprintf(w, "Schedules considered: %d\n", s.PossibleSchedules)
	}
	fmt.Fprintf(w, "Schedules executed:   %d\n", s.SchedulesExplored)
	fmt.Fprintf(w, "States visited:       %d\n", s.StatesVisited)
	fmt.Fprintf(w, "Unique states:        %d\n", s.UniqueStates)
	fmt.Fprintf(w, "Equivalent pruned:    %d\n", s.EquivalentPruned)
	fmt.Fprintf(w, "Violations:           %d\n", s.Violations)
	if s.Workers > 0 {
		fmt.Fprintf(w, "Workers:              %d\n", s.Workers)
		fmt.Fprintf(w, "Transitions:          %d\n", s.Transitions)
	}
	fmt.Fprintf(w, "Duration:             %s\n", formatNanos(s.DurationNanos))
	if s.HeapBytes > 0 {
		fmt.Fprintf(w, "Alloc delta:          %d bytes\n", s.HeapBytes)
	}
	fmt.Fprintf(w, "\n")
}

func formatNanos(ns int64) string {
	if ns < 1_000_000 {
		return fmt.Sprintf("%dµs", ns/1_000)
	}
	if ns < 1_000_000_000 {
		return fmt.Sprintf("%.2fms", float64(ns)/1_000_000)
	}
	return fmt.Sprintf("%.2fs", float64(ns)/1_000_000_000)
}

type replayOpts struct {
	failOnViolation bool
	json            bool
	path            string
}

func runReplay(args []string, stdout, stderr io.Writer) int {
	opts := replayOpts{}
	rest, err := parseReplayFlags(args, &opts)
	if errors.Is(err, errHelp) {
		printHelp(stdout)
		return ExitOK
	}
	if err != nil {
		fmt.Fprintf(stderr, "rupurace replay: %v\n", err)
		return ExitError
	}
	if len(rest) != 1 {
		fmt.Fprintf(stderr, "rupurace replay: require exactly one witness path\n")
		return ExitError
	}
	opts.path = rest[0]

	raw, err := os.ReadFile(opts.path)
	if err != nil {
		fmt.Fprintf(stderr, "rupurace replay: %v\n", err)
		return ExitError
	}
	var wf report.File
	if err := json.Unmarshal(raw, &wf); err != nil {
		fmt.Fprintf(stderr, "rupurace replay: invalid witness JSON: %v\n", err)
		return ExitError
	}
	if wf.Kind != "" && wf.Kind != report.WitnessKind {
		fmt.Fprintf(stderr, "rupurace replay: unsupported kind %q\n", wf.Kind)
		return ExitError
	}
	if wf.Scenario == "" {
		fmt.Fprintf(stderr, "rupurace replay: missing scenario\n")
		return ExitError
	}

	loaded, err := demos.LoadFromWitness(wf.Scenario, wf.Params)
	if err != nil {
		fmt.Fprintf(stderr, "rupurace replay: %v\n", err)
		return ExitError
	}

	w := witness.Witness{
		Schedule:  report.ToSchedule(wf.Schedule),
		Invariant: wf.Invariant,
		Violation: wf.Violation,
		FailedAt:  wf.FailedAt,
	}

	result, err := replay.Run(loaded.Scene, loaded.Specs, w)
	if err != nil {
		fmt.Fprintf(stderr, "rupurace replay: %v\n", err)
		return ExitError
	}
	view := report.FromResult(result, loaded.Name, loaded.Params)

	input := map[string]any{
		"path":     opts.path,
		"scenario": loaded.Name,
		"params":   loaded.Params,
		"steps":    len(wf.Schedule),
	}
	configuration := map[string]any{
		"fail_on_violation": opts.failOnViolation,
		"json":              opts.json,
	}
	envelope := audit.Build("replay", input, configuration, view)

	if err := emit(envelope, opts.json, stdout); err != nil {
		fmt.Fprintf(stderr, "rupurace replay: %v\n", err)
		return ExitError
	}
	if !opts.json {
		printExploreHuman(stderr, result)
		if result.Witness != nil && wf.FailedAt != 0 && result.Witness.FailedAt != wf.FailedAt {
			fmt.Fprintf(stderr, "warning: failedAt differs (witness=%d replay=%d)\n",
				wf.FailedAt, result.Witness.FailedAt)
		}
	}

	if !result.Passed && opts.failOnViolation {
		return ExitGate
	}
	return ExitOK
}

func emit(envelope audit.Envelope, asJSON bool, stdout io.Writer) error {
	if !asJSON {
		return nil
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(envelope)
}

func writeJSON(path string, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func parseExploreFlags(args []string, opts *exploreOpts) ([]string, error) {
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--json":
			opts.json = true
		case a == "--fail-on-violation":
			opts.failOnViolation = true
		case a == "--stats":
			opts.stats = true
		case a == "--continue":
			opts.contAfterViolation = true
		case a == "--measure-memory":
			opts.measureMemory = true
		case a == "--scenario":
			v, err := needValue(args, &i, a)
			if err != nil {
				return nil, err
			}
			opts.scenario = v
		case a == "--strategy":
			v, err := needValue(args, &i, a)
			if err != nil {
				return nil, err
			}
			opts.strategy = v
		case a == "--seed":
			v, err := needValue(args, &i, a)
			if err != nil {
				return nil, err
			}
			n, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("--seed: %w", err)
			}
			opts.seed = n
		case a == "--max-steps":
			v, err := needValue(args, &i, a)
			if err != nil {
				return nil, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return nil, errors.New("--max-steps must be a non-negative integer")
			}
			opts.maxSteps = n
		case a == "--balance":
			v, err := needValue(args, &i, a)
			if err != nil {
				return nil, err
			}
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, errors.New("--balance must be an integer")
			}
			opts.balance = n
		case a == "--amount":
			v, err := needValue(args, &i, a)
			if err != nil {
				return nil, err
			}
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, errors.New("--amount must be an integer")
			}
			opts.amount = n
		case a == "--expected":
			v, err := needValue(args, &i, a)
			if err != nil {
				return nil, err
			}
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, errors.New("--expected must be an integer")
			}
			opts.expected = n
		case a == "--workers":
			v, err := needValue(args, &i, a)
			if err != nil {
				return nil, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				return nil, errors.New("--workers must be a positive integer")
			}
			opts.workers = n
		case a == "--steps":
			v, err := needValue(args, &i, a)
			if err != nil {
				return nil, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 1 {
				return nil, errors.New("--steps must be a positive integer")
			}
			opts.steps = n
		case a == "--quota-a":
			v, err := needValue(args, &i, a)
			if err != nil {
				return nil, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return nil, errors.New("--quota-a must be a non-negative integer")
			}
			opts.quotaA = n
		case a == "--quota-b":
			v, err := needValue(args, &i, a)
			if err != nil {
				return nil, err
			}
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return nil, errors.New("--quota-b must be a non-negative integer")
			}
			opts.quotaB = n
		case a == "--max-value":
			v, err := needValue(args, &i, a)
			if err != nil {
				return nil, err
			}
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, errors.New("--max-value must be an integer")
			}
			opts.maxValue = n
		case a == "-o" || a == "--output":
			v, err := needValue(args, &i, a)
			if err != nil {
				return nil, err
			}
			opts.output = v
		case a == "-h" || a == "--help":
			return nil, errHelp
		case strings.HasPrefix(a, "-"):
			return nil, fmt.Errorf("unknown flag %s", a)
		default:
			rest = append(rest, a)
		}
	}
	return rest, nil
}

func parseReplayFlags(args []string, opts *replayOpts) ([]string, error) {
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--json":
			opts.json = true
		case a == "--fail-on-violation":
			opts.failOnViolation = true
		case a == "-h" || a == "--help":
			return nil, errHelp
		case strings.HasPrefix(a, "-"):
			return nil, fmt.Errorf("unknown flag %s", a)
		default:
			rest = append(rest, a)
		}
	}
	return rest, nil
}

func needValue(args []string, i *int, flag string) (string, error) {
	if *i+1 >= len(args) {
		return "", fmt.Errorf("%s requires a value", flag)
	}
	*i++
	return args[*i], nil
}
