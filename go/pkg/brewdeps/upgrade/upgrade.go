// Package upgrade reconciles Homebrew upgrades: outdated formulae and casks
// are upgraded, then old versions are cleaned up. Its observation of pending
// upgrades is also the prerequisite check for Homebrew package planning.
package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/proc"
	"github.com/daneroo/dotfiles/go/pkg/run"
)

// NoAutoremove stops Homebrew from removing packages it considers unneeded,
// which it decides from installation history rather than the desired state.
const NoAutoremove = "HOMEBREW_NO_AUTOREMOVE=1"

// Reconciler reconciles Homebrew upgrades under the run.Runner.
type Reconciler struct {
	x proc.Exec
}

// New returns the Homebrew upgrades reconciler, running brew through x.
func New(x proc.Exec) *Reconciler {
	return &Reconciler{x: x}
}

// Name is the reconciler's stable name.
func (r *Reconciler) Name() string { return "Homebrew Upgrades" }

// Plan refreshes Homebrew's metadata, then observes the outdated formulae and
// casks. Pinned packages are reported but not upgraded. A missing brew makes
// the plan unavailable.
func (r *Reconciler) Plan(ctx context.Context) (run.Plan, error) {
	if _, err := r.x.LookPath("brew"); err != nil {
		return run.UnavailablePlan("brew is not installed, skipping"), nil
	}
	// Refreshing metadata changes only Homebrew's own caches
	if _, err := r.x.Output(ctx, proc.Command("brew", "update")); err != nil {
		return nil, err
	}
	outdated, err := Observe(ctx, r.x)
	if err != nil {
		return nil, err
	}
	return planUpgrades(r.x, outdated), nil
}

// Upgrade is one outdated formula or cask.
type Upgrade struct {
	Name   string
	IsCask bool
	// Installed is the installed versions, Current the newest release.
	Installed []string
	Current   string
	Pinned    bool
}

// Outdated is the outdated formulae and casks Homebrew reports.
type Outdated []Upgrade

// Pending returns the upgrades that would be applied: the outdated packages
// that aren't pinned.
func (o Outdated) Pending() Outdated {
	var pending Outdated
	for _, u := range o {
		if !u.Pinned {
			pending = append(pending, u)
		}
	}
	return pending
}

// Observe returns the outdated formulae and casks, by running
// brew outdated --json=v2. It does not refresh Homebrew's metadata, so
// callers can observe again after upgrading without a second update.
func Observe(ctx context.Context, x proc.Exec) (Outdated, error) {
	out, err := x.Output(ctx, proc.Command("brew", "outdated", "--json=v2"))
	if err != nil {
		return nil, err
	}
	var response struct {
		Formulae []outdatedEntry `json:"formulae"`
		Casks    []outdatedEntry `json:"casks"`
	}
	if err := json.Unmarshal(out, &response); err != nil {
		return nil, fmt.Errorf("failed to parse brew outdated output: %w", err)
	}
	var outdated Outdated
	for _, group := range []struct {
		entries []outdatedEntry
		isCask  bool
	}{{response.Formulae, false}, {response.Casks, true}} {
		for _, e := range group.entries {
			if e.Name == "" || len(e.InstalledVersions) == 0 || e.CurrentVersion == "" {
				return nil, fmt.Errorf("incomplete brew outdated entry: %+v", e)
			}
			outdated = append(outdated, Upgrade{
				Name:      e.Name,
				IsCask:    group.isCask,
				Installed: e.InstalledVersions,
				Current:   e.CurrentVersion,
				Pinned:    e.Pinned,
			})
		}
	}
	return outdated, nil
}

// outdatedEntry is one formula or cask in brew outdated --json=v2.
type outdatedEntry struct {
	Name              string   `json:"name"`
	InstalledVersions []string `json:"installed_versions"`
	CurrentVersion    string   `json:"current_version"`
	Pinned            bool     `json:"pinned"`
}

// plan is the Homebrew upgrades plan: its report and the brew commands to run.
type plan struct {
	report run.Report
	x      proc.Exec
	steps  []proc.Cmd
}

// Report describes the plan.
func (p *plan) Report() run.Report { return p.report }

// Apply upgrades, then cleans up, stopping at the first failure: cleanup
// never follows a failed upgrade.
func (p *plan) Apply(ctx context.Context) error {
	for _, c := range p.steps {
		if err := p.x.Run(ctx, c); err != nil {
			return err
		}
	}
	return nil
}

// planUpgrades reports each outdated package and plans the upgrade of the
// named formulae, then casks, then a cleanup, all without autoremove. Naming
// the packages keeps the upgrade to the reported set.
func planUpgrades(x proc.Exec, outdated Outdated) *plan {
	p := &plan{x: x}
	if len(outdated) == 0 {
		p.report.Paragraphs = [][]run.Note{{{Mark: run.OK, Text: "All formulae and casks are up to date"}}}
		return p
	}

	var notes []run.Note
	var formulae, casks []string
	for _, u := range outdated {
		name := u.Name
		if u.IsCask {
			name += " (cask)"
		}
		installed := strings.Join(u.Installed, ", ")
		if u.Pinned {
			notes = append(notes, run.Note{Mark: run.Info, Text: fmt.Sprintf("brew: %s is pinned at %s (%s available), not upgraded", name, installed, u.Current)})
			continue
		}
		notes = append(notes, run.Note{Mark: run.Problem, Text: fmt.Sprintf("brew: %s is outdated (%s → %s)", name, installed, u.Current)})
		if u.IsCask {
			casks = append(casks, u.Name)
		} else {
			formulae = append(formulae, u.Name)
		}
	}
	p.report.Paragraphs = [][]run.Note{notes}

	pending := len(formulae) + len(casks)
	if pending == 0 {
		return p
	}
	if len(formulae) > 0 {
		p.steps = append(p.steps, proc.Command("brew", append([]string{"upgrade", "--formula"}, formulae...)...).WithEnv(NoAutoremove))
	}
	if len(casks) > 0 {
		p.steps = append(p.steps, proc.Command("brew", append([]string{"upgrade", "--cask"}, casks...)...).WithEnv(NoAutoremove))
	}
	p.steps = append(p.steps, proc.Command("brew", "cleanup").WithEnv(NoAutoremove))

	group := run.Group{Title: "Formulae and casks to upgrade", Count: pending}
	for _, c := range p.steps {
		group.Actions = append(group.Actions, c.String())
	}
	p.report.Groups = []run.Group{group}
	return p
}
