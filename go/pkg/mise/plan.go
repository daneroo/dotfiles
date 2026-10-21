package mise

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/run"
)

// observed is the actual state of the versioned tools, as mise reports it.
type observed struct {
	// installed maps each tool to its installed versions
	installed map[string][]string
	// defaults maps each tool to the version specs in mise's global config, in order
	defaults map[string][]string
	// resolved maps each tool's version spec to the newest release it matches
	resolved map[string]map[string]string
	// globalConfig is the path of mise's global config, when it lists any tool
	globalConfig string
}

// planned is the result of comparing: a report line per check, then the
// changes that close the gap, grouped by kind.
type planned struct {
	report []run.Note
	// install holds the concrete versions the specs resolved to, so applying
	// never resolves a spec again.
	install []toolVersion
	// align holds the tools whose global default list must become the specs.
	align     []toolSpecs
	uninstall []toolVersion
	// unuse holds the strays to remove from mise's global config.
	unuse []string
}

// toolVersion is one concrete version of a tool, such as python@3.12.15.
type toolVersion struct {
	tool, version string
}

// String renders the version as mise names it.
func (v toolVersion) String() string { return v.tool + "@" + v.version }

// toolSpecs is a tool's version specs, in config order.
type toolSpecs struct {
	tool  string
	specs []string
}

// plan compares the desired version specs with the actual state and returns
// the report lines and the changes that reconcile the differences:
//   - a resolved version that isn't installed → install it
//   - a global default list that differs from the specs → align it
//   - an installed version that no spec resolves to → uninstall it
//   - a stray in mise's global config, a tool that config.yaml doesn't list → unuse it
//
// Tools are taken in name order, so the output is stable.
func plan(desired map[string][]string, actual observed) planned {
	var p planned

	for _, tool := range slices.Sorted(maps.Keys(desired)) {
		specs := desired[tool]
		installed := actual.installed[tool]
		var queued []string
		for _, spec := range specs {
			version := actual.resolved[tool][spec]
			if slices.Contains(installed, version) {
				p.report = append(p.report, note(run.OK, "%s %s → %s", tool, spec, version))
				continue
			}
			p.report = append(p.report, note(run.Problem, "%s %s → %s is not installed", tool, spec, version))
			// Two specs, such as 3.12 and latest, may resolve to one version
			if !slices.Contains(queued, version) {
				queued = append(queued, version)
				p.install = append(p.install, toolVersion{tool, version})
			}
		}

		defaults := actual.defaults[tool]
		if slices.Equal(defaults, specs) {
			p.report = append(p.report, note(run.OK, "%s defaults match config.yaml: %s", tool, strings.Join(specs, ", ")))
		} else {
			p.report = append(p.report, note(run.Problem, "%s defaults are %s, config.yaml wants %s", tool, listOrNone(defaults), strings.Join(specs, ", ")))
			p.align = append(p.align, toolSpecs{tool, specs})
		}
	}

	for _, tool := range slices.Sorted(maps.Keys(actual.installed)) {
		for _, version := range actual.installed[tool] {
			if resolvesTo(actual.resolved[tool], desired[tool], version) {
				continue
			}
			p.report = append(p.report, note(run.Problem, "%s %s is installed but no version spec resolves to it", tool, version))
			p.uninstall = append(p.uninstall, toolVersion{tool, version})
		}
	}

	// Unusing only removes the stray's line; its versions are uninstalled above
	for _, tool := range slices.Sorted(maps.Keys(actual.defaults)) {
		if _, ok := desired[tool]; ok {
			continue
		}
		p.report = append(p.report, note(run.Problem, "%s is a stray in mise's global config, %s: config.yaml doesn't list it", tool, actual.globalConfig))
		p.unuse = append(p.unuse, tool)
	}

	return p
}

// resolvesTo reports whether one of the specs resolves to the version.
func resolvesTo(resolved map[string]string, specs []string, version string) bool {
	for _, spec := range specs {
		if resolved[spec] == version {
			return true
		}
	}
	return false
}

// note makes a report line.
func note(m run.Mark, format string, args ...any) run.Note {
	return run.Note{Mark: m, Text: fmt.Sprintf(format, args...)}
}

// listOrNone joins the items for a report line, or says (none).
func listOrNone(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, ", ")
}
