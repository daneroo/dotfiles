package mise

import (
	"fmt"
	"maps"
	"slices"
	"strings"
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

// planned is what reconciling prints: a report line per check, then the
// commands that close the gap, grouped by kind.
type planned struct {
	report    []string
	install   []string
	use       []string
	uninstall []string
	unuse     []string
}

// plan compares the desired version specs with the actual state and returns
// the report lines and the mise commands that reconcile the differences:
//   - a resolved version that isn't installed → mise install <tool>@<version>
//   - a global default list that differs from the specs → mise use -g <tool>@<spec> ...
//   - an installed version that no spec resolves to → mise uninstall <tool>@<version>
//   - a stray in mise's global config, a tool that config.yaml doesn't list → mise unuse -g <tool>
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
				p.report = append(p.report, fmt.Sprintf("✓ - %s %s → %s", tool, spec, version))
				continue
			}
			p.report = append(p.report, fmt.Sprintf("✗ - %s %s → %s is not installed", tool, spec, version))
			// Two specs, such as 3.12 and latest, may resolve to one version
			if !slices.Contains(queued, version) {
				queued = append(queued, version)
				p.install = append(p.install, fmt.Sprintf("mise install %s@%s", tool, version))
			}
		}

		defaults := actual.defaults[tool]
		if slices.Equal(defaults, specs) {
			p.report = append(p.report, fmt.Sprintf("✓ - %s defaults match config.yaml: %s", tool, strings.Join(specs, ", ")))
		} else {
			p.report = append(p.report, fmt.Sprintf("✗ - %s defaults are %s, config.yaml wants %s", tool, listOrNone(defaults), strings.Join(specs, ", ")))
			p.use = append(p.use, useCommand(tool, specs))
		}
	}

	for _, tool := range slices.Sorted(maps.Keys(actual.installed)) {
		for _, version := range actual.installed[tool] {
			if resolvesTo(actual.resolved[tool], desired[tool], version) {
				continue
			}
			p.report = append(p.report, fmt.Sprintf("✗ - %s %s is installed but no version spec resolves to it", tool, version))
			p.uninstall = append(p.uninstall, fmt.Sprintf("mise uninstall %s@%s", tool, version))
		}
	}

	// mise unuse -g only removes the stray's line; its versions are uninstalled above
	for _, tool := range slices.Sorted(maps.Keys(actual.defaults)) {
		if _, ok := desired[tool]; ok {
			continue
		}
		p.report = append(p.report, fmt.Sprintf("✗ - %s is a stray in mise's global config, %s: config.yaml doesn't list it", tool, actual.globalConfig))
		p.unuse = append(p.unuse, "mise unuse -g "+tool)
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

// useCommand returns the mise use -g command that sets a tool's global
// default list to its specs, in config order.
func useCommand(tool string, specs []string) string {
	args := make([]string, len(specs))
	for i, spec := range specs {
		args[i] = tool + "@" + spec
	}
	return "mise use -g " + strings.Join(args, " ")
}

// listOrNone joins the items for a report line, or says (none).
func listOrNone(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, ", ")
}
