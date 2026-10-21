package mise

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

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
	p := plan(r.desired, actual)
	var create string
	if len(p.align) > 0 {
		path, err := globalConfigPath()
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
			create = path
		} else if err != nil {
			return nil, fmt.Errorf("failed to check mise's global config: %w", err)
		}
	}
	return newPlan(r.x, p, create), nil
}

// globalConfigPath is the file mise config set --global edits, by mise's
// rules: MISE_GLOBAL_CONFIG_FILE, or config.toml in MISE_CONFIG_DIR, or in
// mise under XDG_CONFIG_HOME, or in ~/.config/mise.
func globalConfigPath() (string, error) {
	if path := os.Getenv("MISE_GLOBAL_CONFIG_FILE"); path != "" {
		return path, nil
	}
	if dir := os.Getenv("MISE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "config.toml"), nil
	}
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "mise", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to find mise's global config: %w", err)
	}
	return filepath.Join(home, ".config", "mise", "config.toml"), nil
}

// misePlan is the mise plan: its report and the mise commands to run.
type misePlan struct {
	report run.Report
	x      proc.Exec
	// install, align and cleanup are the commands, in the order they run.
	install, align, cleanup []proc.Cmd
	// create is mise's missing global config, created empty before align.
	create string
}

// Report describes the plan.
func (p *misePlan) Report() run.Report { return p.report }

// Apply runs the steps in order, stopping at the first failure, so a failed
// install or default change is never followed by cleanup.
func (p *misePlan) Apply(ctx context.Context) error {
	if err := p.run(ctx, p.install); err != nil {
		return err
	}
	if p.create != "" {
		if err := createEmpty(p.create); err != nil {
			return err
		}
	}
	if err := p.run(ctx, p.align); err != nil {
		return err
	}
	return p.run(ctx, p.cleanup)
}

// run runs the commands in order, stopping at the first failure.
func (p *misePlan) run(ctx context.Context, cmds []proc.Cmd) error {
	for _, c := range cmds {
		if err := p.x.Run(ctx, c); err != nil {
			return err
		}
	}
	return nil
}

// createEmpty creates mise's missing global config, empty: mise config set
// refuses to create it, reporting the missing file as not trusted. A file
// that appeared since planning is kept as it is.
func createEmpty(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create mise's global config: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create mise's global config: %w", err)
	}
	return f.Close()
}

// newPlan turns the comparison into mise commands, in the order they run:
//  1. install the concrete versions the specs resolved to when planning
//  2. set the global defaults, editing only the tools' lines in mise's
//     global config, create (empty) when missing: the first spec as a
//     string, then each other spec appended as a string, so commas inside a
//     spec stay in it
//  3. uninstall the unwanted versions
//  4. remove the strays' lines, keeping their installations (uninstalled
//     above when unwanted): no pruning by mise's own rules
func newPlan(x proc.Exec, p planned, create string) *misePlan {
	mp := &misePlan{x: x, create: create}
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
	mp.install, mp.align, mp.cleanup = install, align, append(uninstall, unuse...)
	defaults := group("Mise defaults to change", align, len(p.align))
	if create != "" {
		defaults.Actions = append([]string{"create empty " + create}, defaults.Actions...)
	}

	mp.report = run.Report{
		Paragraphs: [][]run.Note{{{Mark: run.OK, Text: "mise is installed"}}},
		Groups: []run.Group{
			group("Missing mise versions", install, 0),
			defaults,
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
