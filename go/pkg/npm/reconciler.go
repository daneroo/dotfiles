// Package npm reconciles npm global packages.
package npm

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/proc"
	"github.com/daneroo/dotfiles/go/pkg/run"
)

// Reconciler reconciles npm global packages under the run.Runner.
type Reconciler struct {
	desired []string
	x       proc.Exec
}

// New returns the npm globals reconciler for the desired packages, running
// npm through x.
func New(desired []string, x proc.Exec) *Reconciler {
	return &Reconciler{desired: desired, x: x}
}

// Name is the reconciler's stable name.
func (r *Reconciler) Name() string { return "NPM Globals" }

// Plan observes the installed and outdated global packages and compares them
// with the desired state:
//   - a missing or outdated package is installed (npm install -g installs the
//     latest release)
//   - an unwanted package is uninstalled, after the installs
//
// A missing npm makes the plan unavailable. A global prefix that does not
// exist yet, such as one .npmrc names before the first global install, holds
// no packages: npm ls -g fails for it.
func (r *Reconciler) Plan(ctx context.Context) (run.Plan, error) {
	npm, installed, ok := r.findNpm(ctx)
	if !ok {
		return run.UnavailablePlan("npm is not installed, skipping"), nil
	}
	prefix, err := r.x.Output(ctx, npm("prefix", "-g"))
	if err != nil {
		return nil, fmt.Errorf("failed to find the global prefix: %w", err)
	}
	if _, err := os.Stat(strings.TrimSpace(string(prefix))); errors.Is(err, fs.ErrNotExist) {
		p := planPackages(r.x, npm, installed, r.desired, nil, nil)
		p.report.Paragraphs[0] = append(p.report.Paragraphs[0], run.Note{Mark: run.Info, Text: fmt.Sprintf("npm's global prefix %s does not exist yet: no globals are installed", strings.TrimSpace(string(prefix)))})
		return p, nil
	}
	actual, err := getInstalledPackages(ctx, r.x, npm)
	if err != nil {
		return nil, err
	}
	outdated, err := getOutdatedPackages(ctx, r.x, npm)
	if err != nil {
		return nil, err
	}
	return planPackages(r.x, npm, installed, r.desired, actual, outdated), nil
}

// launcher builds the command that runs npm with the arguments.
type launcher func(args ...string) proc.Cmd

// findNpm finds npm on PATH or, when npm came with a node that mise
// installed earlier in this run and the parent shell has not loaded its
// location, through mise: mise exec runs it in that node's environment,
// with its automatic installs off, so observing installs nothing. It
// returns how to run npm and the note saying where it is.
func (r *Reconciler) findNpm(ctx context.Context) (launcher, run.Note, bool) {
	if _, err := r.x.LookPath("npm"); err == nil {
		npm := func(args ...string) proc.Cmd { return proc.Command("npm", args...) }
		return npm, run.Note{Mark: run.OK, Text: "npm is installed"}, true
	}
	if _, err := r.x.LookPath("mise"); err != nil {
		return nil, run.Note{}, false
	}
	if out, err := r.x.Output(ctx, proc.Command("mise", "which", "npm")); err != nil || strings.TrimSpace(string(out)) == "" {
		return nil, run.Note{}, false
	}
	npm := func(args ...string) proc.Cmd {
		return proc.Command("mise", append([]string{"exec", "--", "npm"}, args...)...).WithEnv("MISE_EXEC_AUTO_INSTALL=false")
	}
	return npm, run.Note{Mark: run.OK, Text: "npm is installed, through mise"}, true
}

// plan is the npm globals plan: its report and the npm commands to run.
type plan struct {
	report run.Report
	x      proc.Exec
	steps  []proc.Cmd
}

// Report describes the plan.
func (p *plan) Report() run.Report { return p.report }

// Apply runs each step in order, stopping at the first failure.
func (p *plan) Apply(ctx context.Context) error {
	for _, c := range p.steps {
		if err := p.x.Run(ctx, c); err != nil {
			return err
		}
	}
	return nil
}

// planPackages reports each package as up to date, outdated, missing or
// unwanted, and plans one install of the missing and outdated packages, then
// one uninstall of the unwanted ones.
func planPackages(x proc.Exec, npm launcher, installed run.Note, desired, actual []string, outdated map[string]outdatedInfo) *plan {
	missing, extra := reconcilePackages(desired, actual)

	var notes []run.Note
	var toInstall []string
	for _, pkg := range actual {
		if slices.Contains(extra, pkg) {
			continue
		}
		if info, ok := outdated[pkg]; ok {
			notes = append(notes, run.Note{Mark: run.Problem, Text: fmt.Sprintf("npm: %s is outdated (%s → %s)", pkg, info.Current, info.Latest)})
			toInstall = append(toInstall, pkg)
		} else {
			notes = append(notes, run.Note{Mark: run.OK, Text: pkg})
		}
	}
	for _, pkg := range missing {
		notes = append(notes, run.Note{Mark: run.Problem, Text: fmt.Sprintf("npm: %s is missing", pkg)})
		toInstall = append(toInstall, pkg)
	}
	for _, pkg := range extra {
		notes = append(notes, run.Note{Mark: run.Problem, Text: fmt.Sprintf("npm: %s is unwanted", pkg)})
	}
	sort.Strings(toInstall)

	p := &plan{x: x}
	install := run.Group{Title: "Missing or outdated npm globals", Count: len(toInstall)}
	if len(toInstall) > 0 {
		c := npm(append([]string{"install", "-g"}, toInstall...)...)
		p.steps = append(p.steps, c)
		install.Actions = []string{c.String()}
	}
	uninstall := run.Group{Title: "Unwanted npm globals", Count: len(extra)}
	if len(extra) > 0 {
		c := npm(append([]string{"uninstall", "-g"}, extra...)...)
		p.steps = append(p.steps, c)
		uninstall.Actions = []string{c.String()}
	}

	paragraphs := [][]run.Note{{installed}}
	if len(notes) > 0 {
		paragraphs = append(paragraphs, notes)
	}
	p.report = run.Report{Paragraphs: paragraphs, Groups: []run.Group{install, uninstall}}
	return p
}
