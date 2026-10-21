// Package stow links the dotfiles repository's core package into the home
// directory with GNU Stow.
package stow

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/proc"
	"github.com/daneroo/dotfiles/go/pkg/run"
)

// Package is the Stow package mirrored into the home directory.
const Package = "core"

// Reconciler links the core package into the home directory under the
// run.Runner.
type Reconciler struct {
	repo, home string
	x          proc.Exec
}

// New returns the Stow reconciler linking repo's core package into home,
// both absolute paths, running stow through x.
func New(repo, home string, x proc.Exec) *Reconciler {
	return &Reconciler{repo: repo, home: home, x: x}
}

// Name is the reconciler's stable name.
func (r *Reconciler) Name() string { return "Stow" }

// Plan asks stow to simulate linking, which changes nothing, and lists the
// home directory's dead links, which reconfig only reports. A conflict is an
// error; a missing stow makes the plan unavailable.
func (r *Reconciler) Plan(ctx context.Context) (run.Plan, error) {
	if !filepath.IsAbs(r.repo) || !filepath.IsAbs(r.home) {
		return nil, fmt.Errorf("stow needs absolute repository and home paths, not %q and %q", r.repo, r.home)
	}
	if _, err := r.x.LookPath("stow"); err != nil {
		return run.UnavailablePlan("stow is not installed, skipping"), nil
	}
	ops, other, err := r.simulate(ctx)
	if err != nil {
		return nil, err
	}
	dead, err := deadLinks(r.home)
	if err != nil {
		return nil, err
	}
	return &stowPlan{x: r.x, apply: r.command(), ops: ops, other: other, dead: dead, home: r.home}, nil
}

// command is the stow command that links the package. It runs in the
// repository, so a .stowrc in the caller's directory has no effect.
func (r *Reconciler) command(flags ...string) proc.Cmd {
	args := slices.Concat(flags, []string{"--dir", r.repo, "--target", r.home, Package})
	return proc.Command("stow", args...).WithDir(r.repo)
}

// operation matches the operation lines stow reports, such as
// "LINK: .bashrc => .dotfiles/core/.bashrc".
var operation = regexp.MustCompile(`^[A-Z]+: `)

// simulationWarning is the line stow adds to every simulation, even one with
// nothing to do.
const simulationWarning = "WARNING: in simulation mode so not modifying filesystem."

// simulate runs stow --simulate, which reports on standard error, and splits
// what it reports into operations and other lines. A conflict fails the
// simulation.
func (r *Reconciler) simulate(ctx context.Context) (ops, other []string, err error) {
	var stderr strings.Builder
	c := r.command("--simulate", "--verbose=1")
	c.Stderr = &stderr
	if _, err := r.x.Output(ctx, c); err != nil {
		return nil, nil, fmt.Errorf("stow cannot link %s: %w", Package, err)
	}
	lines := bufio.NewScanner(strings.NewReader(stderr.String()))
	for lines.Scan() {
		line := strings.TrimSpace(lines.Text())
		switch {
		case line == "" || line == simulationWarning:
		case operation.MatchString(line) && !strings.HasPrefix(line, "WARNING"):
			ops = append(ops, line)
		default:
			other = append(other, line)
		}
	}
	return ops, other, nil
}

// deadLink is a link in the home directory whose target is missing.
type deadLink struct {
	path, target string
}

// deadLinks lists the dot entries directly in home that are links stat
// cannot follow, as find -L "$HOME" -maxdepth 1 -name '.*' -type l does.
func deadLinks(home string) ([]deadLink, error) {
	entries, err := os.ReadDir(home)
	if err != nil {
		return nil, fmt.Errorf("failed to check for dead links: %w", err)
	}
	var dead []deadLink
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") || e.Type()&os.ModeSymlink == 0 {
			continue
		}
		path := filepath.Join(home, e.Name())
		if _, err := os.Stat(path); err == nil {
			continue
		}
		target, err := os.Readlink(path)
		if err != nil {
			return nil, fmt.Errorf("failed to check for dead links: %w", err)
		}
		dead = append(dead, deadLink{path: path, target: target})
	}
	return dead, nil
}

// stowPlan is the Stow plan: the simulated operations and the command that
// performs them.
type stowPlan struct {
	x     proc.Exec
	apply proc.Cmd
	// ops are the operations stow simulated; other is anything else it
	// reported.
	ops, other []string
	dead       []deadLink
	home       string
}

// Report describes the plan: stow's other output and the dead links are
// observations, never actions.
func (p *stowPlan) Report() run.Report {
	installed := []run.Note{{Mark: run.OK, Text: "stow is installed"}}
	for _, line := range p.other {
		installed = append(installed, run.Note{Mark: run.Info, Text: "stow: " + line})
	}
	var dead []run.Note
	for _, d := range p.dead {
		dead = append(dead, run.Note{Mark: run.Problem, Text: fmt.Sprintf("Dead link, not managed by reconfig: %s -> %s", d.path, d.target)})
	}
	if len(dead) == 0 {
		dead = []run.Note{{Mark: run.OK, Text: "No dead links in " + p.home}}
	}
	return run.Report{
		Paragraphs: [][]run.Note{installed, dead},
		Groups:     []run.Group{{Title: "Stow operations", Actions: p.ops}},
	}
}

// Apply runs stow for real: it plans again and checks for conflicts itself.
// The simulated operations are only shown, never run.
func (p *stowPlan) Apply(ctx context.Context) error {
	return p.x.Run(ctx, p.apply)
}
