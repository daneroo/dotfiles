package npm_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/daneroo/dotfiles/go/pkg/npm"
	"github.com/daneroo/dotfiles/go/pkg/proc"
	"github.com/daneroo/dotfiles/go/pkg/run"
)

// fakeNpm is a controlled npm: its global packages are a map from name to
// version, and installing a package installs its latest version.
type fakeNpm struct {
	missing bool
	// prefix is npm's global prefix; npm ls -g fails while it does not
	// exist, and installing creates it.
	prefix    string
	installed map[string]string
	latest    map[string]string
	// fail makes the npm command with these arguments fail.
	fail string
	// runs records the commands that changed the state.
	runs []string
}

func (f *fakeNpm) LookPath(name string) (string, error) {
	if name != "npm" || f.missing {
		return "", errors.New("not found")
	}
	return "/fake/bin/npm", nil
}

func (f *fakeNpm) Output(_ context.Context, c proc.Cmd) ([]byte, error) {
	cmd := c.String()
	if cmd == f.fail {
		return nil, fmt.Errorf("%s failed: exit status 1", cmd)
	}
	switch cmd {
	case "npm prefix -g":
		return []byte(f.prefix + "\n"), nil
	case "npm ls -g --json --depth=0":
		if _, err := os.Stat(f.prefix); err != nil {
			return nil, fmt.Errorf("%s failed: exit status 254\nstderr: npm error enoent", cmd)
		}
		deps := map[string]map[string]string{}
		for pkg, v := range f.installed {
			deps[pkg] = map[string]string{"version": v}
		}
		return json.Marshal(map[string]any{"dependencies": deps})
	case "npm outdated -g --json":
		outdated := map[string]map[string]string{}
		for pkg, v := range f.installed {
			if latest, ok := f.latest[pkg]; ok && latest != v {
				outdated[pkg] = map[string]string{"current": v, "latest": latest}
			}
		}
		out, _ := json.Marshal(outdated)
		if len(outdated) > 0 {
			// npm outdated exits 1 when updates are available
			return out, errors.New("exit status 1")
		}
		return out, nil
	}
	return nil, fmt.Errorf("unexpected command: %s", cmd)
}

func (f *fakeNpm) Run(_ context.Context, c proc.Cmd) error {
	cmd, args := c.String(), c.Args
	f.runs = append(f.runs, cmd)
	if cmd == f.fail {
		return fmt.Errorf("%s failed: exit status 1", cmd)
	}
	if len(args) < 2 || args[1] != "-g" {
		return fmt.Errorf("unexpected command: %s", cmd)
	}
	if args[0] == "install" {
		if err := os.MkdirAll(f.prefix, 0o755); err != nil {
			return err
		}
	}
	for _, pkg := range args[2:] {
		switch args[0] {
		case "install":
			f.installed[pkg] = f.latest[pkg]
		case "uninstall":
			delete(f.installed, pkg)
		default:
			return fmt.Errorf("unexpected command: %s", cmd)
		}
	}
	return nil
}

// newFake has json outdated, serve current, and leftpad unwanted; config
// wants json, prettier and serve.
func newFake() *fakeNpm {
	return &fakeNpm{
		prefix:    os.TempDir(),
		installed: map[string]string{"json": "11.0.0", "leftpad": "1.0.0", "serve": "14.2.4"},
		latest:    map[string]string{"json": "11.1.0", "leftpad": "1.3.0", "prettier": "3.6.2", "serve": "14.2.4"},
	}
}

var desired = []string{"json", "prettier", "serve"}

func reconcile(t *testing.T, mode run.Mode, f *fakeNpm) (run.Result, string) {
	t.Helper()
	var out strings.Builder
	res := run.Runner{Mode: mode, Out: &out}.Run(context.Background(), []run.Reconciler{npm.New(desired, f)})
	return res, out.String()
}

// The fixture's report is applied, installs before uninstalls, and a rerun
// finds nothing to do.
func TestApplyThenUnchangedRerun(t *testing.T) {
	f := newFake()
	res, out := reconcile(t, run.Yolo, f)
	want := `
## NPM Globals Section

✓ - npm is installed

✗ - npm: json is outdated (11.0.0 → 11.1.0)
✓ - serve
✗ - npm: prettier is missing
✗ - npm: leftpad is unwanted

✗ - Missing or outdated npm globals: (2)
 npm install -g json prettier
✗ - Unwanted npm globals: (1)
 npm uninstall -g leftpad
✓ - NPM Globals applied
`
	if out != want {
		t.Errorf("output =\n%s\nwant\n%s", out, want)
	}
	if res.ExitCode() != 0 {
		t.Fatalf("exit code = %d: %v", res.ExitCode(), res.Err)
	}
	wantRuns := []string{"npm install -g json prettier", "npm uninstall -g leftpad"}
	if !slices.Equal(f.runs, wantRuns) {
		t.Errorf("ran %q, want %q", f.runs, wantRuns)
	}

	f.runs = nil
	res, out = reconcile(t, run.Yolo, f)
	want = `
## NPM Globals Section

✓ - npm is installed

✓ - json
✓ - prettier
✓ - serve

✓ - No missing or outdated npm globals
✓ - No unwanted npm globals
`
	if out != want {
		t.Errorf("rerun output =\n%s\nwant\n%s", out, want)
	}
	if len(f.runs) != 0 || res.Entries[0].Status != run.Unchanged {
		t.Errorf("rerun ran %q with status %v, want nothing and unchanged", f.runs, res.Entries[0].Status)
	}
}

func TestDryRunChangesNothing(t *testing.T) {
	f := newFake()
	res, out := reconcile(t, run.DryRun, f)
	if len(f.runs) != 0 {
		t.Errorf("dry run ran %q", f.runs)
	}
	if !strings.Contains(out, " npm install -g json prettier\n") || res.Entries[0].Outcome != run.Previewed {
		t.Errorf("dry run did not preview the install:\n%s", out)
	}
}

func TestMissingNpmIsUnavailable(t *testing.T) {
	f := newFake()
	f.missing = true
	res, out := reconcile(t, run.Yolo, f)
	if !strings.Contains(out, "✗ - npm is not installed, skipping\n") {
		t.Errorf("output lacks the skip:\n%s", out)
	}
	if res.Entries[0].Status != run.Unavailable || res.ExitCode() != 0 || len(f.runs) != 0 {
		t.Errorf("status %v, exit code %d, ran %q; want unavailable, 0, nothing", res.Entries[0].Status, res.ExitCode(), f.runs)
	}
}

func TestObservationFailureIsAnError(t *testing.T) {
	f := newFake()
	f.fail = "npm ls -g --json --depth=0"
	res, out := reconcile(t, run.Yolo, f)
	if res.ExitCode() != 1 || !strings.Contains(out, "✗ - planning failed: failed to list global packages: npm ls -g --json --depth=0 failed") {
		t.Errorf("exit code %d, output:\n%s", res.ExitCode(), out)
	}
}

// A failed install stops the plan: nothing is uninstalled after it.
func TestInstallFailureStopsRemovals(t *testing.T) {
	f := newFake()
	f.fail = "npm install -g json prettier"
	res, _ := reconcile(t, run.Yolo, f)
	if !slices.Equal(f.runs, []string{"npm install -g json prettier"}) {
		t.Errorf("ran %q, want only the failed install", f.runs)
	}
	if res.Entries[0].Outcome != run.Failed || res.ExitCode() != 1 {
		t.Errorf("outcome %v, exit code %d; want failed, 1", res.Entries[0].Outcome, res.ExitCode())
	}
}

// A global prefix that does not exist yet holds no packages; installing
// creates it, and a rerun observes it.
func TestMissingGlobalPrefix(t *testing.T) {
	f := newFake()
	f.prefix = filepath.Join(t.TempDir(), ".npm-global")
	f.installed = map[string]string{}
	res, out := reconcile(t, run.Yolo, f)
	for _, want := range []string{
		"△ - npm's global prefix " + f.prefix + " does not exist yet: no globals are installed\n",
		" npm install -g json prettier serve\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if res.ExitCode() != 0 {
		t.Fatalf("exit code %d: %v", res.ExitCode(), res.Err)
	}
	res, out = reconcile(t, run.Yolo, f)
	if res.Entries[0].Status != run.Unchanged {
		t.Errorf("rerun status = %v, want unchanged:\n%s", res.Entries[0].Status, out)
	}
}

// installsNpm stands in for an earlier reconciler that installs npm.
type installsNpm struct{ f *fakeNpm }

func (i installsNpm) Name() string { return "Node" }

func (i installsNpm) Plan(context.Context) (run.Plan, error) { return i, nil }

func (i installsNpm) Report() run.Report {
	return run.Report{Groups: []run.Group{{Title: "Missing node", Actions: []string{"install node"}}}}
}

func (i installsNpm) Apply(context.Context) error {
	i.f.missing = false
	return nil
}

// npm installed by an earlier reconciler is observed in the same run.
func TestObservesNpmInstalledEarlierInTheRun(t *testing.T) {
	f := newFake()
	f.missing = true
	var out strings.Builder
	res := run.Runner{Mode: run.Yolo, Out: &out}.Run(context.Background(),
		[]run.Reconciler{installsNpm{f}, npm.New(desired, f)})
	if got := res.Entries[1]; got.Status != run.Actionable || got.Outcome != run.Applied {
		t.Errorf("npm entry = %+v, want actionable and applied:\n%s", got, out.String())
	}
}
