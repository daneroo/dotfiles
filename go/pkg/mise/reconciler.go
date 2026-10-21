package mise

import (
	"context"

	"github.com/daneroo/dotfiles/go/pkg/proc"
	"github.com/daneroo/dotfiles/go/pkg/run"
)

// Reconciler reconciles versioned tools through mise under the run.Runner.
type Reconciler struct {
	desired map[string][]string
	x       proc.Exec
}

// New returns the mise reconciler for the desired version specs of each
// versioned tool, running mise through x.
func New(desired map[string][]string, x proc.Exec) *Reconciler {
	return &Reconciler{desired: desired, x: x}
}

// Name is the reconciler's stable name.
func (r *Reconciler) Name() string { return "Mise" }

// Plan observes the installed versions and mise's global defaults, asks mise
// to resolve each version spec, and compares. Mise interprets the specs;
// reconfig only keeps and compares what they resolve to. A missing mise
// makes the plan unavailable.
func (r *Reconciler) Plan(ctx context.Context) (run.Plan, error) {
	if _, err := r.x.LookPath("mise"); err != nil {
		return run.UnavailablePlan("mise is not installed, skipping"), nil
	}
	actual, err := observe(ctx, r.x, r.desired)
	if err != nil {
		return nil, err
	}
	return newPlan(r.x, plan(r.desired, actual)), nil
}

// misePlan is the mise plan: its report and the mise commands to run.
type misePlan struct {
	report run.Report
	x      proc.Exec
	steps  []proc.Cmd
}

// Report describes the plan.
func (p *misePlan) Report() run.Report { return p.report }

// Apply runs the steps in order, stopping at the first failure, so a failed
// install or default change is never followed by cleanup.
func (p *misePlan) Apply(ctx context.Context) error {
	for _, c := range p.steps {
		if err := p.x.Run(ctx, c); err != nil {
			return err
		}
	}
	return nil
}

// newPlan turns the comparison into mise commands, in the order they run:
//  1. install the concrete versions the specs resolved to when planning
//  2. set the global defaults, editing only the tools' lines in mise's
//     global config: the first spec as a string, then each other spec
//     appended as a string, so commas inside a spec stay in it
//  3. uninstall the unwanted versions
//  4. remove the strays' lines, keeping their installations (uninstalled
//     above when unwanted): no pruning by mise's own rules
func newPlan(x proc.Exec, p planned) *misePlan {
	mp := &misePlan{x: x}
	var install, align, uninstall, unuse []proc.Cmd
	for _, v := range p.install {
		install = append(install, proc.Command("mise", "install", v.String()))
	}
	for _, a := range p.align {
		key := "tools." + a.tool
		align = append(align, proc.Command("mise", "config", "set", "--global", "--type", "string", key, a.specs[0]))
		for _, spec := range a.specs[1:] {
			align = append(align, proc.Command("mise", "config", "set", "--global", "--append", "--type", "string", key, spec))
		}
	}
	for _, v := range p.uninstall {
		uninstall = append(uninstall, proc.Command("mise", "uninstall", v.String()))
	}
	for _, tool := range p.unuse {
		unuse = append(unuse, proc.Command("mise", "unuse", "--global", "--no-prune", tool))
	}
	mp.steps = append(append(append(install, align...), uninstall...), unuse...)

	mp.report = run.Report{
		Paragraphs: [][]run.Note{{{Mark: run.OK, Text: "mise is installed"}}},
		Groups: []run.Group{
			group("Missing mise versions", install, 0),
			group("Mise defaults to change", align, len(p.align)),
			group("Unwanted mise versions", uninstall, 0),
			group("Strays in mise's global config", unuse, 0),
		},
	}
	if len(p.report) > 0 {
		mp.report.Paragraphs = append(mp.report.Paragraphs, p.report)
	}
	return mp
}

// group shows one kind of change.
func group(title string, cmds []proc.Cmd, count int) run.Group {
	g := run.Group{Title: title, Count: count}
	for _, c := range cmds {
		g.Actions = append(g.Actions, c.String())
	}
	return g
}
