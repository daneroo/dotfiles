// Package packages reconciles Homebrew formulae and casks with the desired
// state, from one dependency graph: missing packages are installed, then
// every installed package the desired state doesn't need is removed.
package packages

import (
	"context"
	"fmt"
	"slices"

	"github.com/daneroo/dotfiles/go/pkg/brewdeps/types"
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/upgrade"
	"github.com/daneroo/dotfiles/go/pkg/proc"
	"github.com/daneroo/dotfiles/go/pkg/run"
)

// Reconciler reconciles Homebrew formulae and casks under the run.Runner.
type Reconciler struct {
	desired []types.Package
	x       proc.Exec
}

// New returns the Homebrew packages reconciler for the desired formulae and
// casks, running brew through x.
func New(desired []types.Package, x proc.Exec) *Reconciler {
	return &Reconciler{desired: desired, x: x}
}

// Name is the reconciler's stable name.
func (r *Reconciler) Name() string { return "Homebrew Packages" }

// Plan checks that no upgrade is pending, then compares the dependency graph
// with the desired state. The desired packages and all their dependencies,
// installed or not, are protected; every other installed package is
// unwanted. A missing brew or a pending upgrade makes the plan unavailable.
func (r *Reconciler) Plan(ctx context.Context) (run.Plan, error) {
	if _, err := r.x.LookPath("brew"); err != nil {
		return run.UnavailablePlan("brew is not installed, skipping"), nil
	}
	// Observed afresh: an upgrade earlier in the run changes the answer
	outdated, err := upgrade.Observe(ctx, r.x)
	if err != nil {
		return nil, err
	}
	if pending := outdated.Pending(); len(pending) > 0 {
		return run.UnavailablePlan(fmt.Sprintf("%d pending upgrades: package changes wait for them, since upgrades can change dependencies", len(pending))), nil
	}

	g, err := observe(ctx, r.x, r.desired)
	if err != nil {
		return nil, err
	}
	missing, unwanted := g.compare(r.desired)
	return newPlan(r, g, missing, unwanted), nil
}

// plan is the Homebrew packages plan: its report, the installs, and the
// approved removals.
type plan struct {
	r        *Reconciler
	report   run.Report
	installs []proc.Cmd
	unwanted []types.Package
}

// Report describes the plan.
func (p *plan) Report() run.Report { return p.report }

// Apply installs the missing packages, then removes the unwanted casks as a
// set, then the unwanted formulae as a set, stopping at the first failure.
// After installing, it observes again and drops any removal that became
// required; it never removes anything the plan did not report. Homebrew's
// own dependency checks still guard every removal.
func (p *plan) Apply(ctx context.Context) error {
	for _, c := range p.installs {
		if err := p.r.x.Run(ctx, c); err != nil {
			return err
		}
	}
	unwanted := p.unwanted
	if len(p.installs) > 0 && len(unwanted) > 0 {
		g, err := observe(ctx, p.r.x, p.r.desired)
		if err != nil {
			return err
		}
		_, stillUnwanted := g.compare(p.r.desired)
		unwanted = slices.DeleteFunc(slices.Clone(unwanted), func(u types.Package) bool {
			return !slices.Contains(stillUnwanted, u)
		})
	}
	for _, c := range removals(unwanted) {
		if err := p.r.x.Run(ctx, c); err != nil {
			return err
		}
	}
	return nil
}

// newPlan reports the missing and unwanted packages and plans their
// installation and removal, with formulae and casks shown separately.
func newPlan(r *Reconciler, g *graph, missing, unwanted []types.Package) *plan {
	p := &plan{r: r, unwanted: unwanted}
	p.installs = commands("install", missing, false)
	p.installs = append(p.installs, commands("install", missing, true)...)

	var notes []run.Note
	for _, m := range missing {
		notes = append(notes, run.Note{Mark: run.Problem, Text: fmt.Sprintf("brew: %s is missing", describe(m))})
	}
	for _, u := range unwanted {
		text := fmt.Sprintf("brew: %s is unwanted", describe(u))
		if g.neededBy(unwanted, u) {
			text += ", needed only by unwanted packages"
		}
		notes = append(notes, run.Note{Mark: run.Problem, Text: text})
	}
	p.report.Paragraphs = [][]run.Note{{{
		Mark: run.OK,
		Text: fmt.Sprintf("Dependency graph is complete: %d installed, %d desired", installedCount(g), len(r.desired)),
	}}}
	if len(notes) > 0 {
		p.report.Paragraphs = append(p.report.Paragraphs, notes)
	}
	p.report.Groups = []run.Group{
		group("Missing formulae", commands("install", missing, false), count(missing, false)),
		group("Missing casks", commands("install", missing, true), count(missing, true)),
		group("Unwanted casks", commands("uninstall", unwanted, true), count(unwanted, true)),
		group("Unwanted formulae", commands("uninstall", unwanted, false), count(unwanted, false)),
	}
	return p
}

// removals returns the uninstall commands: casks first, since casks can
// depend on formulae.
func removals(unwanted []types.Package) []proc.Cmd {
	return append(commands("uninstall", unwanted, true), commands("uninstall", unwanted, false)...)
}

// commands returns the one brew command, without autoremove, that applies
// the verb to the packages of one kind, or none when there are none.
func commands(verb string, pkgs []types.Package, casks bool) []proc.Cmd {
	flag := "--formula"
	if casks {
		flag = "--cask"
	}
	args := []string{verb, flag}
	for _, p := range pkgs {
		if p.IsCask == casks {
			args = append(args, p.Name)
		}
	}
	if len(args) == 2 {
		return nil
	}
	return []proc.Cmd{proc.Command("brew", args...).WithEnv(upgrade.NoAutoremove)}
}

// group shows one kind of change.
func group(title string, cmds []proc.Cmd, n int) run.Group {
	g := run.Group{Title: title, Count: n}
	for _, c := range cmds {
		g.Actions = append(g.Actions, c.String())
	}
	return g
}

// count returns how many of the packages are of one kind.
func count(pkgs []types.Package, casks bool) int {
	n := 0
	for _, p := range pkgs {
		if p.IsCask == casks {
			n++
		}
	}
	return n
}

// installedCount returns how many packages are installed.
func installedCount(g *graph) int {
	n := 0
	for _, node := range g.nodes {
		if node.installed {
			n++
		}
	}
	return n
}
