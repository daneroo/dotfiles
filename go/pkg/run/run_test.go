package run_test

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/daneroo/dotfiles/go/pkg/run"
)

// stub is a scripted reconciler. Its plan is actionable when it has an
// action, and applying it records its name in the shared log.
type stub struct {
	name        string
	action      string // empty: unchanged
	unavailable bool
	planErr     error
	applyErr    error
	applied     *[]string
	// planned, when set, is called while planning, to observe shared state.
	planned func()
	// apply, when set, replaces the default Apply.
	apply func(ctx context.Context) error
}

func (s stub) Name() string { return s.name }

func (s stub) Plan(context.Context) (run.Plan, error) {
	if s.planned != nil {
		s.planned()
	}
	if s.planErr != nil {
		return nil, s.planErr
	}
	if s.unavailable {
		return run.UnavailablePlan(s.name + " is not installed, skipping"), nil
	}
	return stubPlan{s}, nil
}

type stubPlan struct{ s stub }

func (p stubPlan) Report() run.Report {
	g := run.Group{Title: "Changes to " + p.s.name}
	if p.s.action != "" {
		g.Actions = []string{p.s.action}
	}
	return run.Report{Groups: []run.Group{g}}
}

func (p stubPlan) Apply(ctx context.Context) error {
	if p.s.apply != nil {
		return p.s.apply(ctx)
	}
	if p.s.applyErr != nil {
		return p.s.applyErr
	}
	*p.s.applied = append(*p.s.applied, p.s.name)
	return nil
}

// runAll runs the reconcilers and returns the result, the report output, and
// the prompt output.
func runAll(t *testing.T, r run.Runner, recs ...run.Reconciler) (run.Result, string, string) {
	t.Helper()
	var out, prompt strings.Builder
	r.Out, r.Prompt = &out, &prompt
	res := r.Run(context.Background(), recs)
	return res, out.String(), prompt.String()
}

func outcomes(res run.Result) []string {
	var got []string
	for _, e := range res.Entries {
		got = append(got, e.Name+":"+e.Outcome.String())
	}
	return got
}

func assertOutcomes(t *testing.T, res run.Result, want ...string) {
	t.Helper()
	if got := outcomes(res); !slices.Equal(got, want) {
		t.Errorf("outcomes = %q, want %q", got, want)
	}
}

func assertContains(t *testing.T, label, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Errorf("%s lacks %q:\n%s", label, want, got)
	}
}

func TestModes(t *testing.T) {
	tests := []struct {
		name        string
		runner      run.Runner
		wantApplied []string
		wantPrompts int
		wantOut     string
	}{
		{
			name:        "yolo applies actionable plans without asking",
			runner:      run.Runner{Mode: run.Yolo},
			wantApplied: []string{"a", "c"},
			wantOut:     "✓ - a applied",
		},
		{
			name:    "dry run applies nothing",
			runner:  run.Runner{Mode: run.DryRun},
			wantOut: "△ - Dry run: not applied",
		},
		{
			name:        "confirm applies approved plans",
			runner:      run.Runner{Mode: run.Confirm, Interactive: true, In: strings.NewReader("y\nyes\n")},
			wantApplied: []string{"a", "c"},
			wantPrompts: 2,
		},
		{
			name:    "noninteractive confirm announces and falls back to dry run",
			runner:  run.Runner{Mode: run.Confirm, In: strings.NewReader("y\ny\n")},
			wantOut: "△ - Input is not interactive: previewing only, as with --dry-run",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var applied []string
			res, out, prompt := runAll(t, tt.runner,
				stub{name: "a", action: "do a", applied: &applied},
				stub{name: "b", applied: &applied},
				stub{name: "u", unavailable: true, applied: &applied},
				stub{name: "c", action: "do c", applied: &applied},
			)
			if !slices.Equal(applied, tt.wantApplied) {
				t.Errorf("applied = %q, want %q", applied, tt.wantApplied)
			}
			if got := strings.Count(prompt, "? [y/N] "); got != tt.wantPrompts {
				t.Errorf("prompts = %d, want %d:\n%s", got, tt.wantPrompts, prompt)
			}
			assertContains(t, "output", out, tt.wantOut)
			assertContains(t, "output", out, "✓ - No changes to b")
			assertContains(t, "output", out, "✗ - u is not installed, skipping")
			if res.ExitCode() != 0 {
				t.Errorf("exit code = %d, want 0: %v", res.ExitCode(), res.Err)
			}
		})
	}
}

func TestPresentation(t *testing.T) {
	var applied []string
	_, out, prompt := runAll(t, run.Runner{Mode: run.Confirm, Interactive: true, In: strings.NewReader("y\n")},
		stub{name: "Things", action: "thing install x y", applied: &applied})
	want := "\n## Things Section\n\n✗ - Changes to Things: (1)\n thing install x y\n✓ - Things applied\n"
	if out != want {
		t.Errorf("output =\n%q\nwant\n%q", out, want)
	}
	if prompt != "Apply Things? [y/N] " {
		t.Errorf("prompt = %q", prompt)
	}
}

func TestDeclineSwitchesToPreview(t *testing.T) {
	var applied, planned []string
	observe := func(name string) func() { return func() { planned = append(planned, name) } }
	res, out, prompt := runAll(t, run.Runner{Mode: run.Confirm, Interactive: true, In: strings.NewReader("n\ny\ny\n")},
		stub{name: "a", action: "do a", applied: &applied, planned: observe("a")},
		stub{name: "b", action: "do b", applied: &applied, planned: observe("b")},
		stub{name: "c", action: "do c", applied: &applied, planned: observe("c")},
	)
	assertOutcomes(t, res, "a:declined", "b:previewed", "c:previewed")
	if len(applied) != 0 {
		t.Errorf("applied after a decline: %q", applied)
	}
	if !slices.Equal(planned, []string{"a", "b", "c"}) {
		t.Errorf("planned = %q, want every reconciler", planned)
	}
	if got := strings.Count(prompt, "? [y/N] "); got != 1 {
		t.Errorf("prompts = %d, want 1", got)
	}
	assertContains(t, "output", out, "△ - Declined: the remaining reconcilers are previewed only")
	if res.ExitCode() != 0 {
		t.Errorf("a decline exits %d, want 0", res.ExitCode())
	}
}

func TestUnansweredPromptDeclines(t *testing.T) {
	var applied []string
	res, _, _ := runAll(t, run.Runner{Mode: run.Confirm, Interactive: true, In: strings.NewReader("")},
		stub{name: "a", action: "do a", applied: &applied})
	assertOutcomes(t, res, "a:declined")
}

func TestErrorsStopApplyingButNotPlanning(t *testing.T) {
	boom := errors.New("boom")
	tests := []struct {
		name  string
		first stub
		want  string
	}{
		{name: "planning error", first: stub{name: "a", planErr: boom}, want: "✗ - planning failed: boom"},
		{name: "apply error", first: stub{name: "a", action: "do a", applyErr: boom}, want: "✗ - applying failed: boom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var applied []string
			tt.first.applied = &applied
			res, out, _ := runAll(t, run.Runner{Mode: run.Yolo},
				tt.first,
				stub{name: "b", action: "do b", applied: &applied},
				stub{name: "c", action: "do c", applied: &applied},
			)
			assertOutcomes(t, res, "a:failed", "b:previewed", "c:previewed")
			if len(applied) != 0 {
				t.Errorf("applied after an error: %q", applied)
			}
			assertContains(t, "output", out, tt.want)
			assertContains(t, "output", out, "△ - Stopped applying: the remaining reconcilers are previewed only")
			if res.ExitCode() != 1 || !errors.Is(res.Err, boom) {
				t.Errorf("exit code = %d, err = %v; want 1 and boom", res.ExitCode(), res.Err)
			}
		})
	}
}

func TestLaterErrorAfterDeclineStillFails(t *testing.T) {
	var applied []string
	res, _, _ := runAll(t, run.Runner{Mode: run.Confirm, Interactive: true, In: strings.NewReader("n\n")},
		stub{name: "a", action: "do a", applied: &applied},
		stub{name: "b", planErr: errors.New("boom")},
		stub{name: "c", action: "do c", applied: &applied},
	)
	assertOutcomes(t, res, "a:declined", "b:failed", "c:previewed")
	if res.ExitCode() != 1 {
		t.Errorf("exit code = %d, want 1", res.ExitCode())
	}
}

// Each reconciler observes the actual state left by the ones before it.
func TestPlansObserveEarlierChanges(t *testing.T) {
	var applied []string
	installed := false
	var sawInstalled bool
	res, _, _ := runAll(t, run.Runner{Mode: run.Yolo},
		stub{name: "installer", action: "install it", apply: func(context.Context) error { installed = true; return nil }},
		stub{name: "user", applied: &applied, planned: func() { sawInstalled = installed }},
	)
	assertOutcomes(t, res, "installer:applied", "user:skipped")
	if !sawInstalled {
		t.Error("the second plan did not observe the first one's change")
	}
}

// Cancellation while waiting at a prompt ends the run; it is not a decline.
func TestCancelAtPrompt(t *testing.T) {
	var applied, planned []string
	in, _ := io.Pipe() // never answers
	ctx, cancel := context.WithCancel(context.Background())
	var out, prompt strings.Builder
	r := run.Runner{Mode: run.Confirm, Interactive: true, In: in, Out: &out, Prompt: &prompt}
	time.AfterFunc(20*time.Millisecond, cancel)
	res := r.Run(ctx, []run.Reconciler{
		stub{name: "a", action: "do a", applied: &applied},
		stub{name: "b", action: "do b", applied: &applied, planned: func() { planned = append(planned, "b") }},
	})
	assertOutcomes(t, res, "a:cancelled")
	if len(applied) != 0 || len(planned) != 0 {
		t.Errorf("after cancellation: applied %q, planned %q", applied, planned)
	}
	if res.ExitCode() != 130 || !errors.Is(res.Err, context.Canceled) {
		t.Errorf("exit code = %d, err = %v; want 130 and context.Canceled", res.ExitCode(), res.Err)
	}
	assertContains(t, "output", out.String(), "✗ - Cancelled")
}

// Cancellation during apply ends the run, whatever error the apply returns.
func TestCancelDuringApply(t *testing.T) {
	var planned []string
	ctx, cancel := context.WithCancel(context.Background())
	var out strings.Builder
	r := run.Runner{Mode: run.Yolo, Out: &out}
	res := r.Run(ctx, []run.Reconciler{
		stub{name: "a", action: "do a", apply: func(context.Context) error {
			cancel()
			return errors.New("signal: interrupt")
		}},
		stub{name: "b", planned: func() { planned = append(planned, "b") }},
	})
	assertOutcomes(t, res, "a:cancelled")
	if len(planned) != 0 {
		t.Errorf("planned after cancellation: %q", planned)
	}
	if res.ExitCode() != 130 {
		t.Errorf("exit code = %d, want 130", res.ExitCode())
	}
}
