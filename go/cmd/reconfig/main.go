// Command reconfig reconciles this machine with config.yaml. Each reconciler
// observes the machine, compares it with the desired state and shows its
// plan; the execution mode decides whether the plan is applied.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/daneroo/dotfiles/go/pkg/brewdeps/packages"
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/upgrade"
	"github.com/daneroo/dotfiles/go/pkg/completions"
	"github.com/daneroo/dotfiles/go/pkg/config"
	"github.com/daneroo/dotfiles/go/pkg/mise"
	"github.com/daneroo/dotfiles/go/pkg/npm"
	"github.com/daneroo/dotfiles/go/pkg/proc"
	"github.com/daneroo/dotfiles/go/pkg/run"
	"github.com/daneroo/dotfiles/go/pkg/stow"
)

// programName is the name help shows, even under go run, whose executable
// has a temporary name.
const programName = "reconfig"

func main() {
	os.Exit(reconfig(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// reconfig runs the command line and returns its exit status: 0 without
// real errors, 1 after one, 2 for a usage error, 130 when interrupted.
func reconfig(args []string, stdin *os.File, stdout, stderr io.Writer) int {
	opts, err := parseArgs(args, stdout, stderr)
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		return 2
	}

	fmt.Fprintf(stdout, "Global Flags:\n")
	fmt.Fprintf(stdout, " - verbose: %v\n", opts.verbose)
	fmt.Fprintf(stdout, " - mode: %s\n", opts.mode)
	fmt.Fprintf(stdout, "Config: %s\n", opts.config)

	fmt.Fprintf(stdout, "\n## Loading Configuration\n\n")
	cfg, err := config.LoadConfig(opts.config)
	if err != nil {
		fmt.Fprintf(stdout, "✗ - %v\n", err)
		return 1
	}
	repo, home, err := targets()
	if err != nil {
		fmt.Fprintf(stdout, "✗ - %v\n", err)
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	var x proc.Exec = proc.OS{}
	if opts.verbose {
		x = verboseExec{Exec: x, out: stdout}
	}
	bun, err := bunTarget(ctx, x)
	if err != nil {
		fmt.Fprintf(stdout, "✗ - %v\n", err)
		return 1
	}
	reconcilers := []run.Reconciler{
		upgrade.New(x),
		packages.New(cfg.Homebrew, x),
		mise.New(cfg.Mise, x),
		npm.New(cfg.Npm, x),
		completions.New(generators(repo), bun, x),
		stow.New(repo, home, x),
	}
	runner := run.Runner{Mode: opts.mode, Interactive: interactive(stdin), In: stdin, Out: stdout, Prompt: stderr}
	return runner.Run(ctx, reconcilers).ExitCode()
}

// options are the parsed command line.
type options struct {
	config  string
	verbose bool
	mode    run.Mode
}

// parseArgs parses the command line with the standard flag parser, which
// accepts each option with one dash or two. Help goes to stdout; usage
// errors, with the help, go to stderr.
func parseArgs(args []string, stdout, stderr io.Writer) (options, error) {
	var opts options
	var dryRun, yolo, help bool
	fs := flag.NewFlagSet(programName, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { usage(fs.Output()) }
	for _, name := range []string{"c", "config"} {
		fs.StringVar(&opts.config, name, "config.yaml", "")
	}
	for name, v := range map[string]*bool{
		"v": &opts.verbose, "verbose": &opts.verbose,
		"n": &dryRun, "dry-run": &dryRun,
		"y": &yolo, "yolo": &yolo,
		"h": &help, "help": &help,
	} {
		fs.BoolVar(v, name, false, "")
	}
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if help {
		usage(stdout)
		return options{}, flag.ErrHelp
	}
	switch {
	case fs.NArg() > 0:
		return options{}, usageError(fs, "unexpected argument: %s", fs.Arg(0))
	case dryRun && yolo:
		return options{}, usageError(fs, "--dry-run and --yolo cannot be used together")
	case dryRun:
		opts.mode = run.DryRun
	case yolo:
		opts.mode = run.Yolo
	}
	return opts, nil
}

// usageError reports a usage error as the flag parser does: the error, then
// the help.
func usageError(fs *flag.FlagSet, format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	fmt.Fprintln(fs.Output(), err)
	fs.Usage()
	return err
}

// usage writes the help.
func usage(w io.Writer) {
	fmt.Fprintf(w, `Usage: %s [options]

Reconcile this machine with its desired state: Homebrew upgrades and
packages, mise versioned tools, npm globals, cached shell completions, then
dotfile links. Each reconciler shows its plan before anything is applied.

Options:
  -c, --config FILE   desired state (default: config.yaml)
  -n, --dry-run       show every plan; apply nothing
  -y, --yolo          apply every plan without asking
  -v, --verbose       show the commands that observe the machine
  -h, --help          show this help

Without --dry-run or --yolo, reconfig asks before applying each plan. When
input is not a terminal, it shows every plan and applies nothing, as with
--dry-run. A declined plan or an error does the same for the rest of the run.

Run it from the root of the dotfiles repository, as reconfig.sh does: it links
the repository's core directory into the home directory.
`, programName)
}

// module identifies the dotfiles repository's root: its go.mod declares it.
const module = "module github.com/daneroo/dotfiles"

// targets returns the dotfiles repository, which must be the working
// directory, and the home directory its core package is linked into.
func targets() (repo, home string, err error) {
	repo, err = os.Getwd()
	if err != nil {
		return "", "", err
	}
	mod, err := os.ReadFile(filepath.Join(repo, "go.mod"))
	if err != nil || !strings.HasPrefix(string(mod), module+"\n") {
		return "", "", fmt.Errorf("%s is not the root of the dotfiles repository; run reconfig.sh", repo)
	}
	home, err = os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	return repo, home, nil
}

// generators are the completion scripts cached in the repository, so shells
// source a file instead of running the tool at startup. Not loaded on demand
// from the tools' own files: npm's and pnpm's live under versioned Cellar
// paths that move on every brew upgrade.
func generators(repo string) []completions.Generator {
	dir := filepath.Join(repo, stow.Package, ".config", "bash_includes")
	return []completions.Generator{
		{Name: "npm", Command: "npm", Args: []string{"completion"}, Target: filepath.Join(dir, "npm_completion.sh")},
		{Name: "pnpm", Command: "pnpm", Args: []string{"completion", "bash"}, Target: filepath.Join(dir, "pnpm_completion.bash")},
		{Name: "docker", Command: "docker", Args: []string{"completion", "bash"}, Target: filepath.Join(dir, "docker_completion.bash")},
		{Name: "mise", Command: "mise", Args: []string{"completion", "bash"}, Target: filepath.Join(dir, "mise_completion.bash")},
	}
}

// bunTarget is where Bun's corrected completion goes: "bun" in Homebrew's
// bash-completion directory, which bash-completion v2 loads on demand.
// Without Homebrew there is none, and the completions reconciler reports
// Bun's completion as skipped.
func bunTarget(ctx context.Context, x proc.Exec) (string, error) {
	prefix := os.Getenv("HOMEBREW_PREFIX")
	if prefix == "" {
		if _, err := x.LookPath("brew"); err != nil {
			return "", nil
		}
		out, err := x.Output(ctx, proc.Command("brew", "--prefix"))
		if err != nil {
			return "", fmt.Errorf("failed to find Homebrew's prefix: %w", err)
		}
		prefix = strings.TrimSpace(string(out))
		if prefix == "" {
			return "", fmt.Errorf("failed to find Homebrew's prefix: brew --prefix printed nothing")
		}
	}
	return filepath.Join(prefix, "share", "bash-completion", "completions", "bun"), nil
}

// interactive reports whether input is a terminal, where someone can answer
// the prompts. /dev/null is a character device too, but nobody answers it.
func interactive(f *os.File) bool {
	info, err := f.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	null, err := os.Stat(os.DevNull)
	return err != nil || !os.SameFile(info, null)
}

// verboseExec shows each command that observes the machine before it runs.
type verboseExec struct {
	proc.Exec
	out io.Writer
}

func (v verboseExec) Output(ctx context.Context, c proc.Cmd) ([]byte, error) {
	fmt.Fprintf(v.out, "△ - observing: %s\n", c)
	return v.Exec.Output(ctx, c)
}
