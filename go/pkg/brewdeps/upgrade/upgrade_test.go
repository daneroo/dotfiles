package upgrade_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/daneroo/dotfiles/go/pkg/brewdeps/upgrade"
	"github.com/daneroo/dotfiles/go/pkg/proc"
	"github.com/daneroo/dotfiles/go/pkg/run"
)

// pkg is one installed formula or cask in the controlled brew.
type pkg struct {
	cask      bool
	installed string
	latest    string
	pinned    bool
}

// fakeBrew is a controlled brew: upgrading a package installs its latest
// version.
type fakeBrew struct {
	missing bool
	pkgs    map[string]*pkg
	// outdatedJSON, when set, replaces the brew outdated output.
	outdatedJSON string
	// fail makes the command with this display form fail.
	fail string
	// outputs and runs record the observations and the changes.
	outputs, runs []string
}

func (f *fakeBrew) LookPath(name string) (string, error) {
	if name != "brew" || f.missing {
		return "", errors.New("not found")
	}
	return "/fake/bin/brew", nil
}

func (f *fakeBrew) Output(_ context.Context, c proc.Cmd) ([]byte, error) {
	cmd := c.String()
	f.outputs = append(f.outputs, cmd)
	if cmd == f.fail {
		return nil, fmt.Errorf("%s failed: exit status 1", cmd)
	}
	switch cmd {
	case "brew update":
		return []byte("Already up-to-date.\n"), nil
	case "brew outdated --json=v2":
		if f.outdatedJSON != "" {
			return []byte(f.outdatedJSON), nil
		}
		type entry struct {
			Name              string   `json:"name"`
			InstalledVersions []string `json:"installed_versions"`
			CurrentVersion    string   `json:"current_version"`
			Pinned            bool     `json:"pinned"`
		}
		response := map[string][]entry{"formulae": {}, "casks": {}}
		for _, name := range slices.Sorted(maps.Keys(f.pkgs)) {
			p := f.pkgs[name]
			if p.installed == p.latest {
				continue
			}
			kind := "formulae"
			if p.cask {
				kind = "casks"
			}
			response[kind] = append(response[kind], entry{name, []string{p.installed}, p.latest, p.pinned})
		}
		return json.Marshal(response)
	}
	return nil, fmt.Errorf("unexpected command: %s", cmd)
}

func (f *fakeBrew) Run(_ context.Context, c proc.Cmd) error {
	cmd := c.String()
	f.runs = append(f.runs, cmd)
	if !slices.Contains(c.Env, upgrade.NoAutoremove) {
		return fmt.Errorf("ran without %s: %s", upgrade.NoAutoremove, cmd)
	}
	if cmd == f.fail {
		return fmt.Errorf("%s failed: exit status 1", cmd)
	}
	switch {
	case slices.Equal(c.Args, []string{"cleanup"}):
		return nil
	case len(c.Args) > 2 && c.Args[0] == "upgrade":
		for _, name := range c.Args[2:] {
			p, ok := f.pkgs[name]
			if !ok || p.cask != (c.Args[1] == "--cask") {
				return fmt.Errorf("no such package: %s", name)
			}
			p.installed = p.latest
		}
		return nil
	}
	return fmt.Errorf("unexpected command: %s", cmd)
}

// newFake has hello and jq outdated, tree current, and the vlc cask outdated.
func newFake() *fakeBrew {
	return &fakeBrew{pkgs: map[string]*pkg{
		"hello": {installed: "2.12.1", latest: "2.12.2"},
		"jq":    {installed: "1.7.1", latest: "1.8.0"},
		"tree":  {installed: "2.2.1", latest: "2.2.1"},
		"vlc":   {cask: true, installed: "3.0.20", latest: "3.0.21"},
	}}
}

// later is an independent reconciler after the upgrades.
type later struct{ planned *bool }

func (l later) Name() string { return "Later" }

func (l later) Plan(context.Context) (run.Plan, error) {
	*l.planned = true
	return run.UnavailablePlan("later is not installed, skipping"), nil
}

func reconcile(t *testing.T, r run.Runner, f *fakeBrew) (run.Result, string, bool) {
	t.Helper()
	var out strings.Builder
	r.Out = &out
	planned := false
	res := r.Run(context.Background(), []run.Reconciler{upgrade.New(f), later{&planned}})
	return res, out.String(), planned
}

func TestUpToDate(t *testing.T) {
	f := newFake()
	f.pkgs = map[string]*pkg{"tree": {installed: "2.2.1", latest: "2.2.1"}}
	res, out, _ := reconcile(t, run.Runner{Mode: run.Yolo}, f)
	if !strings.Contains(out, "## Homebrew Upgrades Section\n\n✓ - All formulae and casks are up to date\n") {
		t.Errorf("output:\n%s", out)
	}
	if res.Entries[0].Status != run.Unchanged || len(f.runs) != 0 {
		t.Errorf("status %v, ran %q; want unchanged, nothing", res.Entries[0].Status, f.runs)
	}
	// Refreshing metadata is part of observing
	if !slices.Equal(f.outputs, []string{"brew update", "brew outdated --json=v2"}) {
		t.Errorf("observed with %q", f.outputs)
	}
}

// Applying upgrades the reported packages, then cleans up; observing again
// afterwards finds nothing pending.
func TestApplyThenFreshObservation(t *testing.T) {
	f := newFake()
	res, out, planned := reconcile(t, run.Runner{Mode: run.Yolo}, f)
	want := `
## Homebrew Upgrades Section

✗ - brew: hello is outdated (2.12.1 → 2.12.2)
✗ - brew: jq is outdated (1.7.1 → 1.8.0)
✗ - brew: vlc (cask) is outdated (3.0.20 → 3.0.21)

✗ - Formulae and casks to upgrade: (3)
 HOMEBREW_NO_AUTOREMOVE=1 brew upgrade --formula hello jq
 HOMEBREW_NO_AUTOREMOVE=1 brew upgrade --cask vlc
 HOMEBREW_NO_AUTOREMOVE=1 brew cleanup
✓ - Homebrew Upgrades applied
`
	if !strings.HasPrefix(out, want) {
		t.Errorf("output =\n%s\nwant prefix\n%s", out, want)
	}
	wantRuns := []string{
		"HOMEBREW_NO_AUTOREMOVE=1 brew upgrade --formula hello jq",
		"HOMEBREW_NO_AUTOREMOVE=1 brew upgrade --cask vlc",
		"HOMEBREW_NO_AUTOREMOVE=1 brew cleanup",
	}
	if !slices.Equal(f.runs, wantRuns) {
		t.Errorf("ran %q, want %q", f.runs, wantRuns)
	}
	if res.ExitCode() != 0 || !planned {
		t.Errorf("exit code %d, later planned %v", res.ExitCode(), planned)
	}

	outdated, err := upgrade.Observe(context.Background(), f)
	if err != nil || len(outdated.Pending()) != 0 {
		t.Errorf("fresh observation = %v, %v; want nothing pending", outdated, err)
	}
}

func TestDryRunAndDeclineUpgradeNothing(t *testing.T) {
	for _, r := range []run.Runner{
		{Mode: run.DryRun},
		{Mode: run.Confirm, Interactive: true, In: strings.NewReader("n\n")},
	} {
		f := newFake()
		res, _, planned := reconcile(t, r, f)
		if len(f.runs) != 0 {
			t.Errorf("mode %v ran %q", r.Mode, f.runs)
		}
		if !planned || res.ExitCode() != 0 {
			t.Errorf("mode %v: later planned %v, exit code %d; want planned, 0", r.Mode, planned, res.ExitCode())
		}
		outdated, err := upgrade.Observe(context.Background(), f)
		if err != nil || len(outdated.Pending()) != 3 {
			t.Errorf("mode %v: pending after preview = %v, %v; want 3", r.Mode, outdated, err)
		}
	}
}

func TestApplyFailures(t *testing.T) {
	tests := []struct {
		fail     string
		wantRuns int
	}{
		// cleanup never follows a failed upgrade
		{fail: "HOMEBREW_NO_AUTOREMOVE=1 brew upgrade --formula hello jq", wantRuns: 1},
		{fail: "HOMEBREW_NO_AUTOREMOVE=1 brew upgrade --cask vlc", wantRuns: 2},
		{fail: "HOMEBREW_NO_AUTOREMOVE=1 brew cleanup", wantRuns: 3},
	}
	for _, tt := range tests {
		t.Run(tt.fail, func(t *testing.T) {
			f := newFake()
			f.fail = tt.fail
			res, out, planned := reconcile(t, run.Runner{Mode: run.Yolo}, f)
			if len(f.runs) != tt.wantRuns || f.runs[len(f.runs)-1] != tt.fail {
				t.Errorf("ran %q, want to stop at %s", f.runs, tt.fail)
			}
			if res.Entries[0].Outcome != run.Failed || res.ExitCode() != 1 {
				t.Errorf("outcome %v, exit code %d; want failed, 1", res.Entries[0].Outcome, res.ExitCode())
			}
			if !planned || !strings.Contains(out, "✗ - applying failed: "+tt.fail+" failed") {
				t.Errorf("later planned %v, output:\n%s", planned, out)
			}
		})
	}
}

// Observation failures and malformed responses are errors returned to the
// runner, which still plans the later reconciler.
func TestObservationErrors(t *testing.T) {
	tests := []struct {
		name         string
		fail         string
		outdatedJSON string
		want         string
	}{
		{name: "update fails", fail: "brew update", want: "brew update failed"},
		{name: "outdated fails", fail: "brew outdated --json=v2", want: "brew outdated --json=v2 failed"},
		{name: "malformed", outdatedJSON: "Error: not JSON", want: "failed to parse brew outdated output"},
		{
			name:         "no installed version",
			outdatedJSON: `{"formulae": [{"name": "hello", "installed_versions": [], "current_version": "2.12.2"}], "casks": []}`,
			want:         "incomplete brew outdated entry",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFake()
			f.fail, f.outdatedJSON = tt.fail, tt.outdatedJSON
			res, out, planned := reconcile(t, run.Runner{Mode: run.Yolo}, f)
			if res.ExitCode() != 1 || !strings.Contains(out, tt.want) {
				t.Errorf("exit code %d, output lacks %q:\n%s", res.ExitCode(), tt.want, out)
			}
			if !planned || len(f.runs) != 0 {
				t.Errorf("later planned %v, ran %q", planned, f.runs)
			}
		})
	}
}

func TestPinnedIsReportedNotUpgraded(t *testing.T) {
	f := newFake()
	f.pkgs["jq"].pinned = true
	_, out, _ := reconcile(t, run.Runner{Mode: run.Yolo}, f)
	if !strings.Contains(out, "△ - brew: jq is pinned at 1.7.1 (1.8.0 available), not upgraded\n") {
		t.Errorf("output:\n%s", out)
	}
	if f.runs[0] != "HOMEBREW_NO_AUTOREMOVE=1 brew upgrade --formula hello" {
		t.Errorf("ran %q, want jq left out", f.runs)
	}

	// Only pinned packages outdated: nothing to apply
	f = newFake()
	f.pkgs = map[string]*pkg{"jq": {installed: "1.7.1", latest: "1.8.0", pinned: true}}
	res, _, _ := reconcile(t, run.Runner{Mode: run.Yolo}, f)
	if res.Entries[0].Status != run.Unchanged || len(f.runs) != 0 {
		t.Errorf("status %v, ran %q; want unchanged, nothing", res.Entries[0].Status, f.runs)
	}
}

func TestMissingBrewIsUnavailable(t *testing.T) {
	f := newFake()
	f.missing = true
	res, out, planned := reconcile(t, run.Runner{Mode: run.Yolo}, f)
	if res.Entries[0].Status != run.Unavailable || res.ExitCode() != 0 || !planned {
		t.Errorf("status %v, exit code %d, later planned %v", res.Entries[0].Status, res.ExitCode(), planned)
	}
	if !strings.Contains(out, "✗ - brew is not installed, skipping") || len(f.outputs) != 0 {
		t.Errorf("observed %q, output:\n%s", f.outputs, out)
	}
}
