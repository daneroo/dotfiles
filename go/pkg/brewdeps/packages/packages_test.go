package packages_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/daneroo/dotfiles/go/pkg/brewdeps/packages"
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/types"
	"github.com/daneroo/dotfiles/go/pkg/run"
)

// machine is a controlled brew on which:
//   - hello, wget and the vlc and editor casks are desired but missing
//   - openssl and ca-certs, wget's dependencies, are installed already
//   - editor-helper and ripgrep, the editor cask's requirements, are too
//   - jq is desired and installed, with oniguruma
//   - figlet and its dependency libfig, tree and the stray-app cask are
//     unwanted
func machine() *fakeBrew {
	return &fakeBrew{
		formulae: map[string]*fpkg{
			"ca-certs":  {installed: true},
			"figlet":    {installed: true, deps: []string{"libfig"}},
			"hello":     {},
			"jq":        {installed: true, deps: []string{"oniguruma"}},
			"libfig":    {installed: true},
			"oniguruma": {installed: true},
			"openssl":   {installed: true, deps: []string{"ca-certs"}},
			"ripgrep":   {installed: true},
			"tree":      {installed: true},
			"wget":      {deps: []string{"openssl"}},
		},
		casks: map[string]*fpkg{
			"editor":        {deps: []string{"ripgrep"}, caskDeps: []string{"editor-helper"}},
			"editor-helper": {installed: true},
			"stray-app":     {installed: true},
			"vlc":           {},
		},
	}
}

var desired = []types.Package{
	{Name: "hello"}, {Name: "jq"}, {Name: "wget"},
	{Name: "editor", IsCask: true}, {Name: "vlc", IsCask: true},
}

// later is an independent reconciler after the packages.
type later struct{ planned *bool }

func (l later) Name() string { return "Later" }

func (l later) Plan(context.Context) (run.Plan, error) {
	*l.planned = true
	return run.UnavailablePlan("later is not installed, skipping"), nil
}

func reconcile(t *testing.T, r run.Runner, f *fakeBrew, want []types.Package) (run.Result, string, bool) {
	t.Helper()
	var out strings.Builder
	r.Out = &out
	planned := false
	res := r.Run(context.Background(), []run.Reconciler{packages.New(want, f), later{&planned}})
	return res, out.String(), planned
}

// A successful run installs what is missing and removes the complete
// unwanted set, protecting the dependencies of missing desired packages and
// cask requirements; a rerun is unchanged.
func TestConvergesThenUnchanged(t *testing.T) {
	f := machine()
	res, out, planned := reconcile(t, run.Runner{Mode: run.Yolo}, f, desired)
	want := `
## Homebrew Packages Section

✓ - Dependency graph is complete: 10 installed, 5 desired

✗ - brew: hello is missing
✗ - brew: wget is missing
✗ - brew: editor (cask) is missing
✗ - brew: vlc (cask) is missing
✗ - brew: figlet is unwanted
✗ - brew: libfig is unwanted, needed only by unwanted packages
✗ - brew: tree is unwanted
✗ - brew: stray-app (cask) is unwanted

✗ - Missing formulae: (2)
 HOMEBREW_NO_AUTOREMOVE=1 brew install --formula hello wget
✗ - Missing casks: (2)
 HOMEBREW_NO_AUTOREMOVE=1 brew install --cask editor vlc
✗ - Unwanted casks: (1)
 HOMEBREW_NO_AUTOREMOVE=1 brew uninstall --cask stray-app
✗ - Unwanted formulae: (3)
 HOMEBREW_NO_AUTOREMOVE=1 brew uninstall --formula figlet libfig tree
✓ - Homebrew Packages applied
`
	if !strings.HasPrefix(out, want) {
		t.Errorf("output =\n%s\nwant prefix\n%s", out, want)
	}
	wantRuns := []string{
		"HOMEBREW_NO_AUTOREMOVE=1 brew install --formula hello wget",
		"HOMEBREW_NO_AUTOREMOVE=1 brew install --cask editor vlc",
		"HOMEBREW_NO_AUTOREMOVE=1 brew uninstall --cask stray-app",
		"HOMEBREW_NO_AUTOREMOVE=1 brew uninstall --formula figlet libfig tree",
	}
	if !slices.Equal(f.runs, wantRuns) {
		t.Errorf("ran %q, want %q", f.runs, wantRuns)
	}
	if res.ExitCode() != 0 || !planned {
		t.Fatalf("exit code %d, later planned %v: %v", res.ExitCode(), planned, res.Err)
	}

	f.runs = nil
	res, out, _ = reconcile(t, run.Runner{Mode: run.Yolo}, f, desired)
	want = `
## Homebrew Packages Section

✓ - Dependency graph is complete: 10 installed, 5 desired

✓ - No missing formulae
✓ - No missing casks
✓ - No unwanted casks
✓ - No unwanted formulae
`
	if !strings.HasPrefix(out, want) {
		t.Errorf("rerun output =\n%s\nwant prefix\n%s", out, want)
	}
	if len(f.runs) != 0 || res.Entries[0].Status != run.Unchanged {
		t.Errorf("rerun ran %q with status %v", f.runs, res.Entries[0].Status)
	}
}

// A desired formula that Homebrew installed as a dependency is kept when the
// package that needed it is removed.
func TestKeepsDesiredDependency(t *testing.T) {
	f := machine()
	_, out, _ := reconcile(t, run.Runner{Mode: run.Yolo}, f, []types.Package{{Name: "oniguruma"}})
	if !strings.Contains(out, "✗ - brew: jq is unwanted\n") || strings.Contains(out, "oniguruma is unwanted") {
		t.Errorf("output:\n%s", out)
	}
	if !f.formulae["oniguruma"].installed || f.formulae["jq"].installed {
		t.Errorf("oniguruma installed %v, jq installed %v; want kept, removed", f.formulae["oniguruma"].installed, f.formulae["jq"].installed)
	}
}

func TestDryRunAndDeclineChangeNothing(t *testing.T) {
	for _, r := range []run.Runner{
		{Mode: run.DryRun},
		{Mode: run.Confirm, Interactive: true, In: strings.NewReader("n\n")},
	} {
		f := machine()
		res, out, planned := reconcile(t, r, f, desired)
		if len(f.runs) != 0 || !planned || res.ExitCode() != 0 {
			t.Errorf("mode %v: ran %q, later planned %v, exit code %d", r.Mode, f.runs, planned, res.ExitCode())
		}
		if !strings.Contains(out, " HOMEBREW_NO_AUTOREMOVE=1 brew uninstall --formula figlet libfig tree\n") {
			t.Errorf("mode %v did not preview the removals:\n%s", r.Mode, out)
		}
	}
}

// After installing, a removal that became required is dropped, and nothing
// unreported is removed.
func TestRevalidationOnlyDropsRemovals(t *testing.T) {
	f := machine()
	// Installing hello records tree as a dependency and brings figlet-extra,
	// which was never reported.
	f.formulae["hello"].recorded = []string{"tree"}
	f.formulae["hello"].sideEffects = []string{"figlet-extra"}
	f.formulae["figlet-extra"] = &fpkg{}
	res, out, _ := reconcile(t, run.Runner{Mode: run.Yolo}, f, desired)
	if !strings.Contains(out, "✗ - brew: tree is unwanted") {
		t.Errorf("tree was not reported unwanted before installing:\n%s", out)
	}
	if got := f.runs[len(f.runs)-1]; got != "HOMEBREW_NO_AUTOREMOVE=1 brew uninstall --formula figlet libfig" {
		t.Errorf("last command = %s, want tree dropped and figlet-extra not added", got)
	}
	if !f.formulae["tree"].installed || !f.formulae["figlet-extra"].installed || res.ExitCode() != 0 {
		t.Errorf("tree installed %v, figlet-extra installed %v, exit code %d", f.formulae["tree"].installed, f.formulae["figlet-extra"].installed, res.ExitCode())
	}
}

func TestFailuresStopRemainingActions(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(f *fakeBrew)
		wantRuns int
		want     string
	}{
		{
			name:     "install fails",
			setup:    func(f *fakeBrew) { f.fail = "HOMEBREW_NO_AUTOREMOVE=1 brew install --formula hello wget" },
			wantRuns: 1,
			want:     "brew install --formula hello wget failed",
		},
		{
			name:     "cask removal fails",
			setup:    func(f *fakeBrew) { f.fail = "HOMEBREW_NO_AUTOREMOVE=1 brew uninstall --cask stray-app" },
			wantRuns: 3,
			want:     "brew uninstall --cask stray-app failed",
		},
		{
			// Homebrew knows a dependency the graph did not: its own guard
			// refuses, and nothing forces the removal.
			name:     "native guard refuses",
			setup:    func(f *fakeBrew) { f.formulae["jq"].hidden = []string{"tree"} },
			wantRuns: 4,
			want:     "Refusing to uninstall tree because it is required by jq",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := machine()
			tt.setup(f)
			res, out, planned := reconcile(t, run.Runner{Mode: run.Yolo}, f, desired)
			if len(f.runs) != tt.wantRuns {
				t.Errorf("ran %q, want %d commands", f.runs, tt.wantRuns)
			}
			if res.Entries[0].Outcome != run.Failed || res.ExitCode() != 1 || !planned {
				t.Errorf("outcome %v, exit code %d, later planned %v", res.Entries[0].Outcome, res.ExitCode(), planned)
			}
			if !strings.Contains(out, tt.want) {
				t.Errorf("output lacks %q:\n%s", tt.want, out)
			}
		})
	}
}

func TestUnavailable(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f *fakeBrew)
		want  string
	}{
		{name: "missing brew", setup: func(f *fakeBrew) { f.missing = true }, want: "✗ - brew is not installed, skipping"},
		{
			name:  "pending upgrade",
			setup: func(f *fakeBrew) { f.formulae["jq"].outdated = true },
			want:  "✗ - 1 pending upgrades: package changes wait for them, since upgrades can change dependencies",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := machine()
			tt.setup(f)
			res, out, planned := reconcile(t, run.Runner{Mode: run.Yolo}, f, desired)
			if res.Entries[0].Status != run.Unavailable || res.ExitCode() != 0 || !planned {
				t.Errorf("status %v, exit code %d, later planned %v", res.Entries[0].Status, res.ExitCode(), planned)
			}
			if !strings.Contains(out, tt.want) || len(f.runs) != 0 {
				t.Errorf("ran %q, output lacks %q:\n%s", f.runs, tt.want, out)
			}
			for _, o := range f.outputs {
				if strings.HasPrefix(o, "brew info") {
					t.Errorf("observed the graph while unavailable: %s", o)
				}
			}
		})
	}
}

// A graph Homebrew cannot complete is a planning error, not a guess.
func TestIncompleteGraphIsAnError(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f *fakeBrew)
		want  string
	}{
		{
			name:  "unknown desired formula",
			setup: func(f *fakeBrew) {},
			want:  `No available formula with the name "nonesuch"`,
		},
		{
			name:  "unknown dependency of an installed formula",
			setup: func(f *fakeBrew) { f.formulae["tree"].deps = []string{"vanished"} },
			want:  `No available formula with the name "vanished"`,
		},
		{
			name:  "installed query fails",
			setup: func(f *fakeBrew) { f.fail = "brew info --json=v2 --installed" },
			want:  "brew info --json=v2 --installed failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := machine()
			tt.setup(f)
			want := desired
			if tt.name == "unknown desired formula" {
				want = append(slices.Clone(desired), types.Package{Name: "nonesuch"})
			}
			res, out, planned := reconcile(t, run.Runner{Mode: run.Yolo}, f, want)
			if res.ExitCode() != 1 || !planned || len(f.runs) != 0 {
				t.Errorf("exit code %d, later planned %v, ran %q", res.ExitCode(), planned, f.runs)
			}
			if !strings.Contains(out, tt.want) {
				t.Errorf("output lacks %q:\n%s", tt.want, out)
			}
		})
	}
}

// A desired package named by an alias is matched to its installed formula.
func TestAliasIsNotMissing(t *testing.T) {
	f := machine()
	f.formulae["jq"].aliases = []string{"jq-cli"}
	_, out, _ := reconcile(t, run.Runner{Mode: run.DryRun}, f, []types.Package{{Name: "jq-cli"}})
	if strings.Contains(out, "jq-cli is missing") || strings.Contains(out, "jq is unwanted") {
		t.Errorf("output:\n%s", out)
	}
}
