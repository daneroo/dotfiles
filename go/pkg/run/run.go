// Package run reconciles the machine one reconciler at a time. The Runner
// alone shows reports, asks for confirmation, applies plans, switches to
// previewing after a decline or an error, and decides the final exit status.
package run

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Mode says whether actionable plans are applied.
type Mode int

const (
	// Confirm asks before applying each actionable plan.
	Confirm Mode = iota
	// DryRun previews every plan and applies none.
	DryRun
	// Yolo applies every actionable plan without asking.
	Yolo
)

// Runner observes, reports and applies reconcilers in sequence, so each one
// observes the actual state its predecessors left.
type Runner struct {
	// Mode is the execution mode the run starts in. A decline or a real
	// error switches the rest of the run to DryRun.
	Mode Mode
	// Interactive says whether In is a terminal. In Confirm mode, a
	// noninteractive run announces that it previews only, as DryRun does.
	Interactive bool
	// In supplies the answers to confirmation prompts.
	In io.Reader
	// Out receives the reports.
	Out io.Writer
	// Prompt receives the confirmation prompts, so they stay visible when
	// Out is captured; nil means Out.
	Prompt io.Writer
}

// Outcome is what the Runner did with one reconciler's plan.
type Outcome int

const (
	// Skipped plans were unchanged or unavailable: there was nothing to apply.
	Skipped Outcome = iota
	// Previewed plans were actionable but the run was not applying.
	Previewed
	// Declined plans were actionable and the owner said no.
	Declined
	// Applied plans were applied successfully.
	Applied
	// Failed plans hit a real error while planning or applying.
	Failed
	// Cancelled plans were interrupted by cancellation, which ends the run.
	Cancelled
)

// String names the outcome.
func (o Outcome) String() string {
	switch o {
	case Skipped:
		return "skipped"
	case Previewed:
		return "previewed"
	case Declined:
		return "declined"
	case Applied:
		return "applied"
	case Failed:
		return "failed"
	case Cancelled:
		return "cancelled"
	}
	return fmt.Sprintf("Outcome(%d)", int(o))
}

// Entry records one reconciler's plan status and outcome.
type Entry struct {
	Name    string
	Status  Status
	Outcome Outcome
	// Err is the real error of a Failed or Cancelled outcome.
	Err error
}

// Result is the record of a run.
type Result struct {
	// Entries are the reconcilers reached, in order.
	Entries []Entry
	// Err joins the run's real errors, including cancellation. Missing
	// tools, unmet prerequisites, previews and declines are not errors.
	Err error
}

// ExitCode is the run's exit status: 0 without real errors, 130 when
// cancelled, as after Ctrl-C, and 1 otherwise.
func (r Result) ExitCode() int {
	switch {
	case r.Err == nil:
		return 0
	case errors.Is(r.Err, context.Canceled):
		return 130
	}
	return 1
}

// Run plans each reconciler in turn, shows its report, and applies its plan
// as the mode allows. A decline or a real error stops applying, but later
// reconcilers are still planned and reported. Cancellation stops the run.
func (r Runner) Run(ctx context.Context, reconcilers []Reconciler) Result {
	if r.Prompt == nil {
		r.Prompt = r.Out
	}
	in := r.In
	if in == nil {
		in = strings.NewReader("")
	}
	answers := bufio.NewReader(in)

	mode := r.Mode
	if mode == Confirm && !r.Interactive {
		note(r.Out, Info, "Input is not interactive: previewing only, as with --dry-run")
		mode = DryRun
	}

	var res Result
	var errs []error
	for _, rec := range reconcilers {
		if err := ctx.Err(); err != nil {
			errs = append(errs, err)
			break
		}
		fmt.Fprintf(r.Out, "\n## %s Section\n\n", rec.Name())
		e := r.reconcile(ctx, rec, mode, answers)
		res.Entries = append(res.Entries, e)

		switch e.Outcome {
		case Declined:
			note(r.Out, Info, "Declined: the remaining reconcilers are previewed only")
			mode = DryRun
		case Failed:
			errs = append(errs, fmt.Errorf("%s: %w", e.Name, e.Err))
			note(r.Out, Problem, e.Err.Error())
			if mode != DryRun {
				note(r.Out, Info, "Stopped applying: the remaining reconcilers are previewed only")
				mode = DryRun
			}
		case Cancelled:
			errs = append(errs, e.Err)
			note(r.Out, Problem, "Cancelled")
		}
		if e.Outcome == Cancelled {
			break
		}
	}
	res.Err = errors.Join(errs...)
	return res
}

// reconcile plans one reconciler, shows its report, and applies the plan if
// it is actionable and the mode allows.
func (r Runner) reconcile(ctx context.Context, rec Reconciler, mode Mode, answers *bufio.Reader) Entry {
	e := Entry{Name: rec.Name()}
	plan, err := rec.Plan(ctx)
	if err != nil {
		return e.failed(ctx, fmt.Errorf("planning failed: %w", err))
	}
	report := plan.Report()
	report.write(r.Out)
	e.Status = report.Status()
	if e.Status != Actionable {
		e.Outcome = Skipped
		return e
	}

	switch mode {
	case DryRun:
		note(r.Out, Info, "Dry run: not applied")
		e.Outcome = Previewed
		return e
	case Confirm:
		approved, err := r.confirm(ctx, e.Name, answers)
		if err != nil {
			return e.failed(ctx, err)
		}
		if !approved {
			e.Outcome = Declined
			return e
		}
	}

	if err := plan.Apply(ctx); err != nil {
		return e.failed(ctx, fmt.Errorf("applying failed: %w", err))
	}
	note(r.Out, OK, e.Name+" applied")
	e.Outcome = Applied
	return e
}

// failed records a real error, or cancellation when the context is done.
func (e Entry) failed(ctx context.Context, err error) Entry {
	e.Outcome, e.Err = Failed, err
	if ctx.Err() != nil {
		e.Outcome, e.Err = Cancelled, ctx.Err()
	}
	return e
}

// confirm asks whether to apply the named plan; only y or yes approves. An
// unanswered prompt, at the end of the input, declines. Cancellation while
// waiting is returned as the context's error, not as a decline.
func (r Runner) confirm(ctx context.Context, name string, answers *bufio.Reader) (bool, error) {
	fmt.Fprintf(r.Prompt, "Apply %s? [y/N] ", name)
	type answer struct {
		line string
		err  error
	}
	got := make(chan answer, 1)
	go func() {
		line, err := answers.ReadString('\n')
		got <- answer{line, err}
	}()
	select {
	case <-ctx.Done():
		fmt.Fprintln(r.Prompt)
		return false, ctx.Err()
	case a := <-got:
		if a.err != nil && !errors.Is(a.err, io.EOF) {
			return false, fmt.Errorf("reading the answer: %w", a.err)
		}
		if a.err != nil {
			fmt.Fprintln(r.Prompt) // end the unanswered prompt's line
		}
		switch strings.ToLower(strings.TrimSpace(a.line)) {
		case "y", "yes":
			return true, nil
		}
		return false, nil
	}
}
