package stow_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/daneroo/dotfiles/go/pkg/proc"
	"github.com/daneroo/dotfiles/go/pkg/run"
	"github.com/daneroo/dotfiles/go/pkg/stow"
)

// fixture is a temporary home directory with the repository inside it, as
// on a real machine:
//
//	home/.dotfiles/core/.bashrc
//	home/.dotfiles/core/.config/inc/one.sh
type fixture struct {
	t                *testing.T
	root, home, repo string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	if _, err := exec.LookPath("stow"); err != nil {
		t.Skip("stow is not installed")
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{t: t, root: root, home: filepath.Join(root, "home")}
	f.repo = filepath.Join(f.home, ".dotfiles")
	f.write(".dotfiles/core/.bashrc", "bashrc\n")
	f.write(".dotfiles/core/.config/inc/one.sh", "one\n")
	return f
}

// write writes a file, relative to home.
func (f *fixture) write(name, content string) {
	f.t.Helper()
	path := filepath.Join(f.home, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

// read reads a file through any links, relative to home.
func (f *fixture) read(name string) string {
	f.t.Helper()
	b, err := os.ReadFile(filepath.Join(f.home, name))
	if err != nil {
		f.t.Fatal(err)
	}
	return string(b)
}

// stow links the package as the real stow does, to set up a layout.
func (f *fixture) stow() {
	f.t.Helper()
	out, err := exec.Command("stow", "--dir", f.repo, "--target", f.home, "core").CombinedOutput()
	if err != nil {
		f.t.Fatalf("stow failed: %v\n%s", err, out)
	}
}

// snapshot describes every entry under home: links by target, files by
// content.
func (f *fixture) snapshot() []string {
	f.t.Helper()
	var entries []string
	err := filepath.WalkDir(f.home, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := strings.TrimPrefix(path, f.home)
		switch {
		case d.Type()&fs.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			entries = append(entries, name+" -> "+target)
		case d.IsDir():
			entries = append(entries, name+"/")
		default:
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			entries = append(entries, fmt.Sprintf("%s %q", name, b))
		}
		return nil
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return entries
}

func (f *fixture) reconciler(x proc.Exec) *stow.Reconciler {
	return stow.New(f.repo, f.home, x)
}

// reconcile runs the reconcilers, with the fixture's home shown as $HOME.
func (f *fixture) reconcile(r run.Runner, rs ...run.Reconciler) (run.Result, string) {
	var out strings.Builder
	r.Out = &out
	res := r.Run(context.Background(), rs)
	return res, strings.ReplaceAll(out.String(), f.home, "$HOME")
}

// The simulated links are made, and a rerun is unchanged and never prompts.
func TestApplyThenUnchangedRerun(t *testing.T) {
	f := newFixture(t)
	res, out := f.reconcile(run.Runner{Mode: run.Yolo}, f.reconciler(proc.OS{}))
	want := `
## Stow Section

✓ - stow is installed

✓ - No dead links in $HOME

✗ - Stow operations: (2)
 LINK: .bashrc => .dotfiles/core/.bashrc
 LINK: .config => .dotfiles/core/.config
✓ - Stow applied
`
	if out != want {
		t.Errorf("output =\n%s\nwant\n%s", out, want)
	}
	if res.ExitCode() != 0 {
		t.Fatalf("exit code = %d: %v", res.ExitCode(), res.Err)
	}
	if f.read(".bashrc") != "bashrc\n" || f.read(".config/inc/one.sh") != "one\n" {
		t.Error("links do not reach the repository's files")
	}

	res, out = f.reconcile(run.Runner{Mode: run.Confirm, Interactive: true, In: strings.NewReader("")}, f.reconciler(proc.OS{}))
	want = `
## Stow Section

✓ - stow is installed

✓ - No dead links in $HOME

✓ - No stow operations
`
	if out != want {
		t.Errorf("rerun output =\n%s\nwant\n%s", out, want)
	}
	if res.Entries[0].Status != run.Unchanged {
		t.Errorf("rerun status = %v, want unchanged", res.Entries[0].Status)
	}
}

// A dry run and a decline both preview the links and change nothing.
func TestPreviewChangesNothing(t *testing.T) {
	for name, runner := range map[string]run.Runner{
		"dry run": {Mode: run.DryRun},
		"decline": {Mode: run.Confirm, Interactive: true, In: strings.NewReader("n\n")},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			before := f.snapshot()
			res, out := f.reconcile(runner, f.reconciler(proc.OS{}))
			if !strings.Contains(out, " LINK: .bashrc => .dotfiles/core/.bashrc\n") || res.ExitCode() != 0 {
				t.Errorf("exit code %d, output:\n%s", res.ExitCode(), out)
			}
			if got := f.snapshot(); !slices.Equal(got, before) {
				t.Errorf("home changed to %q, want %q", got, before)
			}
		})
	}
}

// A conflict is a planning error that links nothing, and the run goes on to
// preview later reconcilers.
func TestConflictIsAPlanningError(t *testing.T) {
	f := newFixture(t)
	f.write(".bashrc", "mine\n")
	before := f.snapshot()
	res, out := f.reconcile(run.Runner{Mode: run.Yolo}, f.reconciler(proc.OS{}), f.reconciler(proc.OS{}))
	for _, want := range []string{
		"✗ - planning failed: stow cannot link core:",
		"cannot stow .dotfiles/core/.bashrc over existing target .bashrc",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	if res.ExitCode() != 1 || res.Entries[0].Outcome != run.Failed || len(res.Entries) != 2 {
		t.Errorf("exit code %d, entries %+v; want 1, failed, and the second planned", res.ExitCode(), res.Entries)
	}
	if got := f.snapshot(); !slices.Equal(got, before) {
		t.Errorf("home changed to %q, want %q", got, before)
	}
}

// conflictBeforeApply creates a conflict after planning, just before stow
// runs for real.
type conflictBeforeApply struct {
	proc.OS
	f *fixture
}

func (c conflictBeforeApply) Run(ctx context.Context, cmd proc.Cmd) error {
	c.f.write(".bashrc", "mine\n")
	return c.OS.Run(ctx, cmd)
}

// Applying keeps stow's own conflict check: a conflict that appears after
// planning fails the apply, and nothing is linked.
func TestApplyFailure(t *testing.T) {
	f := newFixture(t)
	x := conflictBeforeApply{OS: proc.OS{Stderr: &strings.Builder{}}, f: f}
	res, out := f.reconcile(run.Runner{Mode: run.Yolo}, f.reconciler(x))
	if res.Entries[0].Outcome != run.Failed || !strings.Contains(out, "✗ - applying failed: stow --dir") {
		t.Errorf("outcome %v, output:\n%s", res.Entries[0].Outcome, out)
	}
	if _, err := os.Lstat(filepath.Join(f.home, ".config")); !errors.Is(err, fs.ErrNotExist) {
		t.Error("a failed apply linked .config")
	}
}

// noStow is a machine without stow.
type noStow struct{ proc.OS }

func (noStow) LookPath(name string) (string, error) { return "", errors.New("not found") }

func TestMissingStowIsUnavailable(t *testing.T) {
	f := newFixture(t)
	res, out := f.reconcile(run.Runner{Mode: run.Yolo}, f.reconciler(noStow{}))
	if !strings.Contains(out, "✗ - stow is not installed, skipping\n") {
		t.Errorf("output lacks the skip:\n%s", out)
	}
	if res.Entries[0].Status != run.Unavailable || res.ExitCode() != 0 {
		t.Errorf("status %v, exit code %d; want unavailable, 0", res.Entries[0].Status, res.ExitCode())
	}
}

// Dead links are reported, never acted on.
func TestDeadLinksAreReportedOnly(t *testing.T) {
	f := newFixture(t)
	f.stow()
	if err := os.Symlink("/nonexistent/nix-profile", filepath.Join(f.home, ".nix-profile")); err != nil {
		t.Fatal(err)
	}
	before := f.snapshot()
	res, out := f.reconcile(run.Runner{Mode: run.Yolo}, f.reconciler(proc.OS{}))
	if !strings.Contains(out, "✗ - Dead link, not managed by reconfig: $HOME/.nix-profile -> /nonexistent/nix-profile\n") {
		t.Errorf("output lacks the dead link:\n%s", out)
	}
	if res.Entries[0].Status != run.Unchanged {
		t.Errorf("status = %v, want unchanged", res.Entries[0].Status)
	}
	if got := f.snapshot(); !slices.Equal(got, before) {
		t.Errorf("home changed to %q, want %q", got, before)
	}
}

// Relative paths would depend on the caller's working directory.
func TestRelativePathsAreErrors(t *testing.T) {
	f := newFixture(t)
	res, out := f.reconcile(run.Runner{Mode: run.Yolo}, stow.New(".", f.home, proc.OS{}))
	if res.ExitCode() != 1 || !strings.Contains(out, "stow needs absolute repository and home paths") {
		t.Errorf("exit code %d, output:\n%s", res.ExitCode(), out)
	}
}

// A file added to the package is linked, whether home's .config is one
// folded link, which already shows the new file, or a real directory of
// links, which needs a new link and keeps its other files.
func TestNewPackageFilesAreLinked(t *testing.T) {
	for _, tc := range []struct {
		name   string
		layout func(f *fixture)
		// op is the operation stow plans for the new file, if any.
		op string
	}{
		{name: "folded", layout: func(f *fixture) { f.stow() }},
		{name: "unfolded", layout: func(f *fixture) {
			f.write(".config/inc/other", "not stowed\n")
			f.stow()
		}, op: " LINK: .config/inc/two.sh => ../../.dotfiles/core/.config/inc/two.sh\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			tc.layout(f)
			info, err := os.Lstat(filepath.Join(f.home, ".config"))
			if err != nil {
				t.Fatal(err)
			}
			if folded := info.Mode()&fs.ModeSymlink != 0; folded != (tc.op == "") {
				t.Fatalf(".config folded = %v, want the %s layout", folded, tc.name)
			}
			f.write(".dotfiles/core/.config/inc/two.sh", "two\n")
			res, out := f.reconcile(run.Runner{Mode: run.Yolo}, f.reconciler(proc.OS{}))
			if res.ExitCode() != 0 {
				t.Fatalf("exit code %d, output:\n%s", res.ExitCode(), out)
			}
			if tc.op != "" && !strings.Contains(out, tc.op) {
				t.Errorf("output lacks %q:\n%s", tc.op, out)
			}
			if got := f.read(".config/inc/two.sh"); got != "two\n" {
				t.Errorf("two.sh = %q, want it linked", got)
			}
			if tc.op != "" && f.read(".config/inc/other") != "not stowed\n" {
				t.Error("the unfolded directory's other file changed")
			}
		})
	}
}
