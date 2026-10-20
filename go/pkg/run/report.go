package run

import (
	"context"
	"fmt"
	"io"
	"strings"
)

// Reconciler brings one kind of actual state into agreement with its desired
// state. Its desired configuration and process dependencies are bound when it
// is constructed. It never prompts and never chooses an execution mode: the
// Runner decides whether its plan is applied.
type Reconciler interface {
	// Name is the reconciler's stable name, used in headings and prompts.
	Name() string
	// Plan observes the actual state and compares it with the desired state.
	// Missing tools and unmet prerequisites are unavailable plans, not errors.
	Plan(ctx context.Context) (Plan, error)
}

// Plan is the changes a reconciler proposes.
type Plan interface {
	// Report describes the plan for display. It is never executed.
	Report() Report
	// Apply makes the changes, stopping at the first failure. The Runner
	// calls it only for an actionable plan, after approval.
	Apply(ctx context.Context) error
}

// Status says whether a plan has anything to apply.
type Status int

const (
	// Unchanged plans have nothing to do.
	Unchanged Status = iota
	// Actionable plans have actions to apply.
	Actionable
	// Unavailable plans could not be made: a missing tool or an unmet
	// prerequisite. They are reported and skipped; they are not errors.
	Unavailable
)

// String names the status.
func (s Status) String() string {
	switch s {
	case Unchanged:
		return "unchanged"
	case Actionable:
		return "actionable"
	case Unavailable:
		return "unavailable"
	}
	return fmt.Sprintf("Status(%d)", int(s))
}

// Report is what a plan displays: observations, then the actions grouped by
// kind. It holds only text; executable arguments stay inside the plan.
type Report struct {
	// Unavailable marks a plan that could not be made; Paragraphs say why.
	Unavailable bool
	// Paragraphs are the observations, separated by blank lines.
	Paragraphs [][]Note
	// Groups are the actions, by kind, each shown once.
	Groups []Group
}

// Status derives the plan's status from its report.
func (r Report) Status() Status {
	if r.Unavailable {
		return Unavailable
	}
	for _, g := range r.Groups {
		if len(g.Actions) > 0 {
			return Actionable
		}
	}
	return Unchanged
}

// Mark is the symbol that starts a report line.
type Mark string

// The report symbols.
const (
	OK      Mark = "✓"
	Problem Mark = "✗"
	Info    Mark = "△"
)

// Note is one observation line, such as "✓ - npm is installed".
type Note struct {
	Mark Mark
	Text string
}

// Group is one kind of action. Its title is shown as "✗ - <Title>: (n)"
// followed by the actions, or as "✓ - No <title>" when it has none.
type Group struct {
	Title string
	// Count is n, how many things the actions change, such as packages
	// installed by one command; zero means one per action.
	Count   int
	Actions []string
}

// write renders the report in the Markdown-like style reconfig has always
// used.
//
// Example output:
//
//	✓ - npm is installed
//
//	✗ - npm: json is outdated (1.0.0 → 1.1.0)
//	✗ - npm: serve is missing
//
//	✗ - Missing or outdated npm globals: (2)
//	 npm install -g json serve
//	✓ - No unwanted npm globals
func (r Report) write(w io.Writer) {
	for i, p := range r.Paragraphs {
		if i > 0 {
			fmt.Fprintln(w)
		}
		for _, n := range p {
			note(w, n.Mark, n.Text)
		}
	}
	if len(r.Groups) == 0 {
		return
	}
	if len(r.Paragraphs) > 0 {
		fmt.Fprintln(w)
	}
	for _, g := range r.Groups {
		if len(g.Actions) == 0 {
			note(w, OK, "No "+lowerFirst(g.Title))
			continue
		}
		count := g.Count
		if count == 0 {
			count = len(g.Actions)
		}
		note(w, Problem, fmt.Sprintf("%s: (%d)", g.Title, count))
		for _, a := range g.Actions {
			fmt.Fprintf(w, " %s\n", a)
		}
	}
}

// note writes one report line.
func note(w io.Writer, m Mark, text string) {
	fmt.Fprintf(w, "%s - %s\n", m, text)
}

// lowerFirst lower-cases the first letter, so "Unwanted npm globals" reads
// "No unwanted npm globals".
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// UnavailablePlan returns a plan that could not be made, explained by the
// note, such as "npm is not installed, skipping".
func UnavailablePlan(text string) Plan {
	return unavailable{Report{
		Unavailable: true,
		Paragraphs:  [][]Note{{{Mark: Problem, Text: text}}},
	}}
}

// unavailable is a plan with nothing to apply.
type unavailable struct{ report Report }

func (u unavailable) Report() Report { return u.report }

func (u unavailable) Apply(context.Context) error {
	return fmt.Errorf("an unavailable plan cannot be applied")
}
