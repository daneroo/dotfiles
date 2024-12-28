// go/pkg/asdf/reconcile.go
package asdf

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// Reconcile performs a complete reconciliation cycle for asdf:
// 1. Check if asdf is installed
// 2. Get actual state (installed plugins)
// 3. Compare with desired state
// 4. Take actions to reconcile differences
func Reconcile(desiredVersions map[string][]string) error {
	// Check if asdf is installed
	if err := exec.Command("command", "-v", "asdf").Run(); err != nil {
		return fmt.Errorf("asdf is not installed")
	}
	fmt.Printf("✓ - asdf is installed\n")

	// Get actual state
	actualPlugins, err := getActualPlugins()
	if err != nil {
		return err
	}

	// Determine required actions
	missing, extra := reconcilePlugins(desiredVersions, actualPlugins)

	// Perform all plugin actions
	if err := performPluginActions(desiredVersions, missing, extra); err != nil {
		return err
	}

	// Show version resolution
	fmt.Printf("\nResolving versions:\n")
	for plugin, specs := range desiredVersions {
		for _, spec := range specs {
			resolved, err := resolveVersion(plugin, spec)
			if err != nil {
				return fmt.Errorf("resolving %s version %q: %w", plugin, spec, err)
			}
			fmt.Printf("✓ - %s: %s -> %s\n", plugin, spec, resolved)
		}
	}

	// TODO: Handle version installation in next PR
	return nil
}

// performPluginActions handles installation, updates, and removal hints for plugins
func performPluginActions(desiredVersions map[string][]string, missing, extra []string) error {
	// Install missing plugins
	for _, plugin := range missing {
		fmt.Printf("✗ - asdf plugin %s is missing. Installing\n", plugin)
		if err := exec.Command("asdf", "plugin-add", plugin).Run(); err != nil {
			return fmt.Errorf("failed to install plugin %s: %w", plugin, err)
		}
		fmt.Printf("✓ - asdf plugin %s is installed\n", plugin)
	}

	// Update all plugins that should be installed (including newly installed ones)
	// Note: missing plugins have been installed above, so we can update them too
	for plugin := range desiredVersions {
		// There is no way to only "check for updates" for a plugin
		// so we just update it and parse the output for status
		updateOut, err := exec.Command("asdf", "plugin-update", plugin).CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to update plugin %s: %w", plugin, err)
		}
		output := string(updateOut)
		if strings.Contains(output, "Already on 'master'") ||
			strings.Contains(output, "Your branch is up to date") {
			fmt.Printf("✓ - %s plugin is installed and up to date\n", plugin)
		} else {
			fmt.Printf("✓ - %s plugin is installed and was updated\n", plugin)
			fmt.Printf("%s\n", output)
		}
	}

	// Remove extraneous plugins (hint)
	if len(extra) > 0 {
		fmt.Printf("✗ - Extraneous plugins found:\n")
		for _, plugin := range extra {
			fmt.Printf("- To remove %s plugin:\n", plugin)
			fmt.Printf(" asdf plugin remove %s\n", plugin)
		}
	}

	return nil
}

// reconcilePlugins determines which plugins need to be installed/removed
func reconcilePlugins(desired map[string][]string, actual []string) (missing, extra []string) {
	// Convert to sets
	desiredSet := make(map[string]bool)
	for plugin := range desired {
		desiredSet[plugin] = true
	}
	actualSet := make(map[string]bool)
	for _, plugin := range actual {
		actualSet[plugin] = true
	}

	// Find missing (in desired but not actual)
	for plugin := range desiredSet {
		if !actualSet[plugin] {
			missing = append(missing, plugin)
		}
	}

	// Find extra (in actual but not desired)
	for plugin := range actualSet {
		if !desiredSet[plugin] {
			extra = append(extra, plugin)
		}
	}

	return missing, extra
}

// getActualPlugins returns the list of currently installed plugins
func getActualPlugins() ([]string, error) {
	out, err := exec.Command("asdf", "plugin-list").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", err)
	}
	return strings.Fields(string(out)), nil
}

// resolveVersion converts a version spec into a concrete version number.
// Supported formats:
// - "latest": resolves to the latest stable version (using asdf latest <plugin>)
// - "lts": for nodejs only, resolves to the latest LTS version from nodejs.org
// - "X[.Y[.Z]]": resolves to the latest version matching the prefix:
//   - "3" -> latest 3.x.x
//   - "3.12" -> latest 3.12.x
//   - "3.12.0" -> exact version
func resolveVersion(plugin, spec string) (string, error) {
	switch {
	case spec == "latest":
		return resolveLatest(plugin)
	case plugin == "nodejs" && spec == "lts":
		return resolveNodeLTS()
	case isVersionPrefix(spec):
		return resolveLatestPatch(plugin, spec)
	default:
		return "", fmt.Errorf("unsupported version spec %q for plugin %q", spec, plugin)
	}
}

func resolveLatest(plugin string) (string, error) {
	out, err := exec.Command("asdf", "latest", plugin).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get latest %s version: %w", plugin, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func resolveNodeLTS() (string, error) {
	out, err := exec.Command("curl", "-s", "https://nodejs.org/dist/index.json").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get Node.js versions: %w", err)
	}

	var releases []struct {
		Version string      `json:"version"`
		LTS     interface{} `json:"lts"`
	}
	if err := json.Unmarshal(out, &releases); err != nil {
		return "", fmt.Errorf("failed to parse Node.js versions: %w", err)
	}

	for _, r := range releases {
		if r.LTS != false {
			return strings.TrimPrefix(r.Version, "v"), nil
		}
	}
	return "", fmt.Errorf("no LTS version found")
}

// resolveLatestPatch finds the latest version matching a prefix (like "3.12" for python).
// The process is:
// 1. Get all available versions from asdf list-all
// 2. Filter versions that match the prefix exactly followed by a patch number
// 3. Sort the matches using version sort (-V)
// 4. Return the last (highest) version
//
// Examples for prefix "3.12":
// - Input versions: ["3.12-dev", "3.12.0", "3.12.1", "3.12.0-rc1", "3.13.0", "3.2.0"]
// - Regex "^3.12\.[0-9]+$" matches: ["3.12.0", "3.12.1"]
// - After sorting: returns "3.12.1"
//
// Assumptions:
// - We filter for clean versions (X.Y.Z where X,Y,Z are numbers)
// - We need proper version sorting for correct results
// - The prefix is already validated by isVersionPrefix
func resolveLatestPatch(plugin, prefix string) (string, error) {
	out, err := exec.Command("asdf", "list-all", plugin).Output()
	if err != nil {
		return "", fmt.Errorf("failed to list %s versions: %w", plugin, err)
	}

	versions := strings.Fields(string(out))
	matches := filterAndSortVersions(versions, prefix)
	if len(matches) == 0 {
		return "", fmt.Errorf("no versions found matching %s for %s", prefix, plugin)
	}

	return matches[len(matches)-1], nil
}

// isVersionPrefix checks if the spec is a valid version prefix (like "3", "3.12", "3.12.0", etc)
// This format is used to find the latest matching version
func isVersionPrefix(spec string) bool {
	pattern := `^\d+(\.\d+){0,2}$`
	matched, _ := regexp.MatchString(pattern, spec)
	return matched
}
