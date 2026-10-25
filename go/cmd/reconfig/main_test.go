package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/daneroo/dotfiles/go/pkg/proc"
	"github.com/daneroo/dotfiles/go/pkg/run"
)

func TestModes(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want options
	}{
		{nil, options{config: "config.yaml", mode: run.Confirm}},
		{[]string{"-n"}, options{config: "config.yaml", mode: run.DryRun}},
		{[]string{"--dry-run"}, options{config: "config.yaml", mode: run.DryRun}},
		{[]string{"-y"}, options{config: "config.yaml", mode: run.Yolo}},
		{[]string{"--yolo"}, options{config: "config.yaml", mode: run.Yolo}},
		{[]string{"-c", "e2e/config.yaml", "-v"}, options{config: "e2e/config.yaml", verbose: true}},
		{[]string{"--config=e2e/config.yaml", "--verbose"}, options{config: "e2e/config.yaml", verbose: true}},
	} {
		var stdout, stderr strings.Builder
		got, err := parseArgs(tc.args, &stdout, &stderr)
		if err != nil || got != tc.want {
			t.Errorf("parseArgs(%q) = %+v, %v; want %+v", tc.args, got, err, tc.want)
		}
		if stdout.Len()+stderr.Len() > 0 {
			t.Errorf("parseArgs(%q) wrote %q %q", tc.args, stdout.String(), stderr.String())
		}
	}
}

// Help goes to stdout under the stable name, with each option's aliases
// together.
func TestHelp(t *testing.T) {
	for _, arg := range []string{"-h", "--help"} {
		var stdout, stderr strings.Builder
		_, err := parseArgs([]string{arg}, &stdout, &stderr)
		if !errors.Is(err, flag.ErrHelp) || stderr.Len() > 0 {
			t.Errorf("%s: err %v, stderr %q", arg, err, stderr.String())
		}
		for _, want := range []string{
			"Usage: reconfig [options]\n",
			"  -c, --config FILE   desired state (default: config.yaml)\n",
			"  -n, --dry-run       show every plan; apply nothing\n",
			"  -y, --yolo          apply every plan without asking\n",
			"  -v, --verbose       show the commands that observe the machine\n",
			"  -h, --help          show this help\n",
		} {
			if !strings.Contains(stdout.String(), want) {
				t.Errorf("%s: help lacks %q:\n%s", arg, want, stdout.String())
			}
		}
	}
}

// Usage errors show the error and the help on stderr, and exit 2.
func TestUsageErrors(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"-n", "-y"}, "--dry-run and --yolo cannot be used together\n"},
		{[]string{"--yolo", "--dry-run"}, "--dry-run and --yolo cannot be used together\n"},
		{[]string{"--force"}, "flag provided but not defined: -force\n"},
		{[]string{"apply"}, "unexpected argument: apply\n"},
	} {
		var stdout, stderr strings.Builder
		if code := reconfig(tc.args, os.Stdin, &stdout, &stderr); code != 2 {
			t.Errorf("reconfig(%q) = %d, want 2", tc.args, code)
		}
		if !strings.HasPrefix(stderr.String(), tc.want) || !strings.Contains(stderr.String(), "Usage: reconfig [options]") {
			t.Errorf("reconfig(%q) stderr =\n%s\nwant %q then the help", tc.args, stderr.String(), tc.want)
		}
		if stdout.Len() > 0 {
			t.Errorf("reconfig(%q) wrote to stdout: %q", tc.args, stdout.String())
		}
	}
}

// Neither a pipe nor /dev/null can answer a prompt.
func TestInteractive(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	if interactive(r) {
		t.Error("a pipe is interactive")
	}
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	if interactive(null) {
		t.Error("/dev/null is interactive")
	}
	if tty, err := os.Open("/dev/tty"); err == nil {
		defer tty.Close()
		if !interactive(tty) {
			t.Error("the terminal is not interactive")
		}
	}
}

// Only the root of the dotfiles repository, by its go.mod, is accepted: not
// another directory, even one with a core directory.
func TestTargetsNeedTheRepository(t *testing.T) {
	var stdout, stderr strings.Builder
	dir := t.TempDir()
	if err := os.Mkdir(dir+"/core", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/go.mod", []byte("module example.com/other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/config.yaml", []byte("homebrew: {formulae: {main: [jq]}, casks: []}\nmise: {}\nnpm: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	if code := reconfig([]string{"-n"}, os.Stdin, &stdout, &stderr); code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stdout.String(), "is not the root of the dotfiles repository; run reconfig.sh") {
		t.Errorf("output lacks the reason:\n%s", stdout.String())
	}
}

// Verbose shows each observation before it runs.
func TestVerboseShowsObservations(t *testing.T) {
	var out strings.Builder
	x := verboseExec{Exec: proc.OS{}, out: &out}
	if _, err := x.Output(context.Background(), proc.Command("echo", "hi")); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "△ - observing: echo hi\n"; got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

// The repository's own root is accepted.
func TestTargetsAcceptTheRepositoryRoot(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(wd, "..", "..", "..")
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)
	repo, _, err := targets()
	if err != nil {
		t.Fatalf("targets() failed at the repository root: %v", err)
	}
	if want, _ := filepath.Abs(root); repo != want {
		t.Errorf("repo = %s, want %s", repo, want)
	}
}
