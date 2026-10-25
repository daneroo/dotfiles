package completions_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/daneroo/dotfiles/go/pkg/completions"
	"github.com/daneroo/dotfiles/go/pkg/proc"
	"github.com/daneroo/dotfiles/go/pkg/run"
)

// fixture is a controlled machine: tools are shell scripts, PATH holds only
// its bin directory, and TMPDIR, where scratch goes, is its own.
//
//	bin/       on PATH: docker, mise, and the few programs the scripts use
//	installs/    where mise installs tools, off PATH: bun and newtool
//	completions/ the per-user completions directory, not created yet
type fixture struct {
	t    *testing.T
	root string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{t: t, root: root}
	for _, dir := range []string{"bin", "tmp"} {
		f.mkdir(dir)
	}
	for _, name := range []string{"sh", "ln", "dirname", "sleep"} {
		path, err := exec.LookPath(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(path, f.path("bin", name)); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", f.path("bin"))
	t.Setenv("TMPDIR", f.path("tmp"))
	t.Setenv("FIXTURE_ROOT", root)

	f.tool("bin/docker", `echo "docker: diagnostics" >&2
echo "# docker completion $*"`)
	f.tool("bin/mise", `case "$1 $2" in
"which bun") echo "$FIXTURE_ROOT/installs/bun/bin/bun" ;;
"which newtool") echo "$FIXTURE_ROOT/installs/newtool/bin/newtool" ;;
*) echo "mise: $2 is not a mise bin" >&2; exit 1 ;;
esac`)
	// newtool runs a program beside it, as npm runs node
	f.tool("installs/newtool/bin/newtool", `exec newtool-helper "$@"`)
	f.tool("installs/newtool/bin/newtool-helper", `echo "# newtool completion $*"`)
	f.tool("installs/bun/bin/bun", bunScript)
	return f
}

// bunScript behaves like bun completions: without a bunx on PATH, it links
// one beside its executable, then prints the script for SHELL. It fails
// unless run in scratch, the parent of its working directory, with Bun's
// directories set there too.
const bunScript = `scratch=$(dirname "$PWD")
case "$scratch" in "$FIXTURE_ROOT/tmp/"*) ;; *) echo "bun: not run in scratch: $PWD" >&2; exit 1 ;; esac
for d in "$BUN_INSTALL" "$XDG_CONFIG_HOME" "$XDG_DATA_HOME" "$XDG_STATE_HOME" "$XDG_CACHE_HOME" "$TMPDIR" "$HOMEBREW_PREFIX"; do
  case "$d" in "$scratch"/*) ;; *) echo "bun: directory not in scratch: $d" >&2; exit 1 ;; esac
done
command -v bunx >/dev/null || ln -s "$0" "$(dirname "$0")/bunx"
[ -n "$BUN_SLEEP" ] && exec sleep "$BUN_SLEEP"
[ "$SHELL" = /bin/bash ] || { echo "bun: no script for $SHELL" >&2; exit 1; }
echo "bun: diagnostics" >&2
[ -n "$BUN_EMPTY" ] && exit 0
echo "_bun() {"
[ -z "$BUN_FIXED" ] && echo '  local re_prev_script="(bun run) "'
echo "}"`

// fixedBun is the script the fixture's bun prints, with bug 2 fixed.
const fixedBun = "_bun() {\n  return; local re_prev_script=\"(bun run) \"\n}\n"

func (f *fixture) path(elem ...string) string {
	return filepath.Join(append([]string{f.root}, elem...)...)
}

func (f *fixture) mkdir(dir string) {
	f.t.Helper()
	if err := os.MkdirAll(f.path(dir), 0o755); err != nil {
		f.t.Fatal(err)
	}
}

// tool writes an executable shell script.
func (f *fixture) tool(name, body string) {
	f.t.Helper()
	f.mkdir(filepath.Dir(name))
	if err := os.WriteFile(f.path(name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) write(name, content string) {
	f.t.Helper()
	f.mkdir(filepath.Dir(name))
	if err := os.WriteFile(f.path(name), []byte(content), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) read(name string) string {
	f.t.Helper()
	b, err := os.ReadFile(f.path(name))
	if err != nil {
		f.t.Fatal(err)
	}
	return string(b)
}

// entries lists a directory's tree, so a test can check it is unchanged.
func (f *fixture) entries(dir string) []string {
	f.t.Helper()
	var names []string
	err := filepath.WalkDir(f.path(dir), func(path string, _ os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		names = append(names, strings.TrimPrefix(path, f.root))
		return nil
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return names
}

// assertScratchRemoved checks no scratch is left in TMPDIR.
func (f *fixture) assertScratchRemoved() {
	f.t.Helper()
	if left := f.entries("tmp"); len(left) != 1 {
		f.t.Errorf("scratch left behind: %q", left[1:])
	}
}

// marker is the first line of every script reconfig writes.
const marker = "# Written by reconfig (daneroo/dotfiles); changes are overwritten on its next run.\n"

// tools are the fixture's tools: bun through mise, docker on PATH, newtool
// only through mise, and pnpm missing.
func (f *fixture) tools() []completions.Tool {
	return []completions.Tool{
		completions.Bun,
		{Name: "docker", Command: "docker", Args: []string{"completion", "bash"}},
		{Name: "newtool", Command: "newtool", Args: []string{"completion", "bash"}},
		{Name: "pnpm", Command: "pnpm", Args: []string{"completion", "bash"}},
	}
}

func (f *fixture) reconciler() *completions.Reconciler {
	return completions.New(f.path("completions"), f.tools(), proc.OS{})
}

// reconcile runs the reconciler, with the fixture's root shown as $ROOT.
func (f *fixture) reconcile(ctx context.Context, mode run.Mode, more ...run.Reconciler) (run.Result, string) {
	var out strings.Builder
	res := run.Runner{Mode: mode, Out: &out}.Run(ctx, append([]run.Reconciler{f.reconciler()}, more...))
	return res, strings.ReplaceAll(out.String(), f.root, "$ROOT")
}

// Fresh scripts are written marked, only where they differ, without
// diagnostics, and a rerun finds nothing to do.
func TestApplyThenUnchangedRerun(t *testing.T) {
	f := newFixture(t)
	f.write("completions/newtool", marker+"# stale\n")
	installs := f.entries("installs")

	res, out := f.reconcile(context.Background(), run.Yolo)
	want := `
## CLI Completions Section

✗ - bun completions are missing ($ROOT/completions/bun)
✗ - docker completions are missing ($ROOT/completions/docker)
✗ - newtool completions are outdated ($ROOT/completions/newtool)
△ - pnpm is not installed, skipping its completions

✗ - Completion files to write: (3)
 write $ROOT/completions/bun
 write $ROOT/completions/docker
 write $ROOT/completions/newtool
✓ - No completion files to remove
✓ - CLI Completions applied
`
	if out != want {
		t.Errorf("output =\n%s\nwant\n%s", out, want)
	}
	if res.ExitCode() != 0 {
		t.Fatalf("exit code = %d: %v", res.ExitCode(), res.Err)
	}
	for name, want := range map[string]string{
		"completions/bun":     marker + fixedBun,
		"completions/docker":  marker + "# docker completion completion bash\n",
		"completions/newtool": marker + "# newtool completion completion bash\n",
	} {
		if got := f.read(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if got := f.entries("installs"); !slices.Equal(got, installs) {
		t.Errorf("installs changed to %q, want %q", got, installs)
	}
	if got := f.entries("completions"); len(got) != 4 {
		t.Errorf("completions holds %q, want only the 3 scripts", got)
	}
	f.assertScratchRemoved()

	res, out = f.reconcile(context.Background(), run.Yolo)
	want = `
## CLI Completions Section

✓ - bun completions are up to date ($ROOT/completions/bun)
✓ - docker completions are up to date ($ROOT/completions/docker)
✓ - newtool completions are up to date ($ROOT/completions/newtool)
△ - pnpm is not installed, skipping its completions

✓ - No completion files to write
✓ - No completion files to remove
`
	if out != want {
		t.Errorf("rerun output =\n%s\nwant\n%s", out, want)
	}
	if res.Entries[0].Status != run.Unchanged {
		t.Errorf("rerun status = %v, want unchanged", res.Entries[0].Status)
	}
}

// A dry run writes nothing, creates no directory, and leaves the
// installations and scratch as they were.
func TestDryRunWritesNothing(t *testing.T) {
	f := newFixture(t)
	installs := f.entries("installs")

	res, out := f.reconcile(context.Background(), run.DryRun)
	if res.Entries[0].Outcome != run.Previewed || !strings.Contains(out, " write $ROOT/completions/bun\n") {
		t.Errorf("outcome %v, want the writes previewed:\n%s", res.Entries[0].Outcome, out)
	}
	if _, err := os.Lstat(f.path("completions")); !os.IsNotExist(err) {
		t.Error("dry run created the completions directory")
	}
	if got := f.entries("installs"); !slices.Equal(got, installs) {
		t.Errorf("installs changed to %q, want %q", got, installs)
	}
	f.assertScratchRemoved()
}

// A link at a target is replaced by a file, leaving the file it points to
// as it was.
func TestLinkIsReplaced(t *testing.T) {
	f := newFixture(t)
	f.write("elsewhere/docker", "# not reconfig's\n")
	f.mkdir("completions")
	if err := os.Symlink(f.path("elsewhere", "docker"), f.path("completions", "docker")); err != nil {
		t.Fatal(err)
	}
	res, out := f.reconcile(context.Background(), run.Yolo)
	if !strings.Contains(out, "✗ - docker completions: $ROOT/completions/docker is a symlink, to be replaced by a file\n") || res.ExitCode() != 0 {
		t.Errorf("exit code %d, output:\n%s", res.ExitCode(), out)
	}
	info, err := os.Lstat(f.path("completions", "docker"))
	if err != nil || !info.Mode().IsRegular() {
		t.Fatalf("docker target is %v (%v), want a regular file", info, err)
	}
	if got := f.read("elsewhere/docker"); got != "# not reconfig's\n" {
		t.Errorf("the linked file changed to %q", got)
	}
}

// A fixed script is written as it is, marked, when its command is
// installed, and skipped when it is missing.
func TestFixedScript(t *testing.T) {
	f := newFixture(t)
	f.tool("bin/aws_completer", "exit 0")
	tools := []completions.Tool{
		{Name: "aws", Command: "aws_completer", Script: "complete -C aws_completer aws\n"},
		{Name: "gone", Command: "gone_completer", Script: "complete -C gone_completer gone\n"},
	}
	var out strings.Builder
	res := run.Runner{Mode: run.Yolo, Out: &out}.Run(context.Background(),
		[]run.Reconciler{completions.New(f.path("completions"), tools, proc.OS{})})
	if res.ExitCode() != 0 || !strings.Contains(out.String(), "△ - gone is not installed, skipping its completions\n") {
		t.Errorf("exit code %d, output:\n%s", res.ExitCode(), out.String())
	}
	if got, want := f.read("completions/aws"), marker+"complete -C aws_completer aws\n"; got != want {
		t.Errorf("aws = %q, want %q", got, want)
	}
	if got := f.entries("completions"); len(got) != 2 {
		t.Errorf("completions holds %q, want only aws", got)
	}
}

// A marked script whose tool is no longer listed is removed, as an
// approved action. Unmarked files and links are never touched, and neither is
// the script of a listed tool that is missing.
func TestUnlistedScriptsAreRemoved(t *testing.T) {
	f := newFixture(t)
	f.write("completions/old", marker+"# old completion\n")
	f.write("completions/copilot", "# copilot's own\n")
	f.write("completions/pnpm", marker+"# pnpm completion\n")
	if err := os.Symlink("old", f.path("completions", "alias")); err != nil {
		t.Fatal(err)
	}

	res, out := f.reconcile(context.Background(), run.DryRun)
	for _, line := range []string{
		"✗ - old completions are no longer listed ($ROOT/completions/old)\n",
		"✗ - Completion files to remove: (1)\n remove $ROOT/completions/old\n",
	} {
		if !strings.Contains(out, line) {
			t.Errorf("dry run output lacks %q:\n%s", line, out)
		}
	}
	if got := f.read("completions/old"); got != marker+"# old completion\n" {
		t.Errorf("dry run changed old to %q", got)
	}

	res, out = f.reconcile(context.Background(), run.Yolo)
	if res.ExitCode() != 0 {
		t.Fatalf("exit code %d, output:\n%s", res.ExitCode(), out)
	}
	if _, err := os.Lstat(f.path("completions", "old")); !os.IsNotExist(err) {
		t.Error("old was not removed")
	}
	for name, want := range map[string]string{
		"completions/copilot": "# copilot's own\n",
		"completions/pnpm":    marker + "# pnpm completion\n",
	} {
		if got := f.read(name); got != want {
			t.Errorf("%s = %q, want it untouched", name, got)
		}
	}
	if _, err := os.Lstat(f.path("completions", "alias")); err != nil {
		t.Errorf("the alias link was removed: %v", err)
	}

	_, out = f.reconcile(context.Background(), run.Yolo)
	if !strings.Contains(out, "✓ - No completion files to remove\n") {
		t.Errorf("rerun output still removes:\n%s", out)
	}
}

// The directory is where bash-completion looks first: under XDG_DATA_HOME,
// or under ~/.local/share when it is unset or empty.
func TestDir(t *testing.T) {
	for _, tc := range []struct{ xdg, want string }{
		{"", "/home/me/.local/share/bash-completion/completions"},
		{"/data", "/data/bash-completion/completions"},
	} {
		t.Setenv("XDG_DATA_HOME", tc.xdg)
		if got := completions.Dir("/home/me"); got != tc.want {
			t.Errorf("XDG_DATA_HOME=%q: Dir = %q, want %q", tc.xdg, got, tc.want)
		}
	}
}

// Once Bun fixes bug 2, its script is written as it is, with a note.
func TestFixedBunIsNoted(t *testing.T) {
	f := newFixture(t)
	t.Setenv("BUN_FIXED", "1")
	_, out := f.reconcile(context.Background(), run.Yolo)
	if !strings.Contains(out, "△ - bun: its completion script no longer has the known regex bug, nothing to fix\n") {
		t.Errorf("output lacks the note:\n%s", out)
	}
	if got, want := f.read("completions/bun"), marker+"_bun() {\n}\n"; got != want {
		t.Errorf("bun = %q, want %q", got, want)
	}
}

// Without mise or the tools, everything is skipped: nothing to do, and no
// error.
func TestMissingToolsAreSkipped(t *testing.T) {
	f := newFixture(t)
	for _, name := range []string{"docker", "mise"} {
		if err := os.Remove(f.path("bin", name)); err != nil {
			t.Fatal(err)
		}
	}
	res, out := f.reconcile(context.Background(), run.Yolo)
	for _, name := range []string{"bun", "docker", "newtool", "pnpm"} {
		if !strings.Contains(out, "△ - "+name+" is not installed, skipping its completions\n") {
			t.Errorf("output lacks %s's skip:\n%s", name, out)
		}
	}
	if res.Entries[0].Status != run.Unchanged || res.ExitCode() != 0 {
		t.Errorf("status %v, exit code %d; want unchanged, 0", res.Entries[0].Status, res.ExitCode())
	}
}

// recorder stands in for a later reconciler, recording whether it applied.
type recorder struct{ applied *bool }

func (r recorder) Name() string { return "Later" }

func (r recorder) Plan(context.Context) (run.Plan, error) { return r, nil }

func (r recorder) Report() run.Report {
	return run.Report{Groups: []run.Group{{Title: "Later changes", Actions: []string{"change"}}}}
}

func (r recorder) Apply(context.Context) error {
	*r.applied = true
	return nil
}

// A failed or empty generation is a planning error: nothing is written, and
// later reconcilers are only previewed.
func TestGenerationFailureStopsLaterApplies(t *testing.T) {
	for name, setup := range map[string]func(f *fixture){
		"failed docker": func(f *fixture) { f.tool("bin/docker", `echo "docker: daemon gone" >&2; exit 1`) },
		"empty bun":     func(f *fixture) { f.t.Setenv("BUN_EMPTY", "1") },
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			setup(f)
			applied := false
			res, out := f.reconcile(context.Background(), run.Yolo, recorder{&applied})
			if res.ExitCode() != 1 || res.Entries[0].Outcome != run.Failed || !strings.Contains(out, "✗ - planning failed: failed to get") {
				t.Errorf("exit code %d, outcome %v, output:\n%s", res.ExitCode(), res.Entries[0].Outcome, out)
			}
			if applied || res.Entries[1].Outcome != run.Previewed {
				t.Errorf("later reconciler applied: outcome %v", res.Entries[1].Outcome)
			}
			if _, err := os.Lstat(f.path("completions")); !os.IsNotExist(err) {
				t.Error("a failed plan wrote a script")
			}
			f.assertScratchRemoved()
		})
	}
}

// A failed write stops the apply: earlier targets keep their new scripts,
// later ones are not written, and no partial file is left.
func TestPartialWriteFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write a read-only directory")
	}
	f := newFixture(t)
	f.mkdir("completions/locked")
	if err := os.Chmod(f.path("completions", "locked"), 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(f.path("completions", "locked"), 0o755) })
	// The middle name lands in a read-only directory, so only its write fails.
	tools := []completions.Tool{
		{Name: "one", Command: "docker", Args: []string{"one"}},
		{Name: "locked/two", Command: "docker", Args: []string{"two"}},
		{Name: "three", Command: "docker", Args: []string{"three"}},
	}
	var out strings.Builder
	res := run.Runner{Mode: run.Yolo, Out: &out}.Run(context.Background(),
		[]run.Reconciler{completions.New(f.path("completions"), tools, proc.OS{})})
	if res.Entries[0].Outcome != run.Failed || !strings.Contains(out.String(), "✗ - applying failed: failed to write "+f.path("completions", "locked", "two")) {
		t.Errorf("outcome %v, output:\n%s", res.Entries[0].Outcome, out.String())
	}
	if got := f.read("completions/one"); got != marker+"# docker completion one\n" {
		t.Errorf("one = %q", got)
	}
	if got := f.entries("completions"); !slices.Equal(got, []string{"/completions", "/completions/locked", "/completions/one"}) {
		t.Errorf("completions holds %q, want only one and the locked directory", got)
	}
}

// Cancellation interrupts Bun's generation and removes its scratch.
func TestCancellationRemovesScratch(t *testing.T) {
	f := newFixture(t)
	t.Setenv("BUN_SLEEP", "30")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	time.AfterFunc(300*time.Millisecond, cancel)

	start := time.Now()
	res, _ := f.reconcile(ctx, run.Yolo)
	if res.Entries[0].Outcome != run.Cancelled || res.ExitCode() != 130 {
		t.Errorf("outcome %v, exit code %d; want cancelled, 130", res.Entries[0].Outcome, res.ExitCode())
	}
	if elapsed := time.Since(start); elapsed > 4*time.Second {
		t.Errorf("cancellation took %v", elapsed)
	}
	f.assertScratchRemoved()
}
