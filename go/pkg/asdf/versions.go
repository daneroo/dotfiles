package asdf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"slices"
	"strings"
)

// reconcileVersionsForPlugin handles the complete version reconciliation for a single plugin:
// 1. Resolve version specs to concrete versions
// 2. Get currently installed versions
// 3. Reconcile differences
func reconcileVersionsForPlugin(plugin string, specs []string) error {
	// Resolve version specs
	fmt.Printf("\nResolving %s versions:\n", plugin)
	var resolvedVersions []string
	for _, spec := range specs {
		resolved, err := resolveVersion(plugin, spec)
		if err != nil {
			return fmt.Errorf("resolving %s version %q: %w", plugin, spec, err)
		}
		fmt.Printf("✓ - %s: %s -> %s\n", plugin, spec, resolved)
		resolvedVersions = append(resolvedVersions, resolved)
	}

	// Remove duplicates and sort
	desired := uniqueVersions(resolvedVersions)
	fmt.Printf("\nResolved %s versions: %s\n", plugin, strings.Join(desired, " "))

	// Get actual installed versions
	actual, err := getInstalledVersions(plugin)
	if err != nil {
		return err
	}

	// Reconcile differences
	missing, extra := reconcileVersions(desired, actual)

	// Show already installed versions (excluding extraneous versions)
	for _, version := range actual {
		if !slices.Contains(extra, version) {
			fmt.Printf("✓ - %s: %s is already installed\n", plugin, version)
		}
	}

	if err := performVersionActions(plugin, missing, extra); err != nil {
		return err
	}

	// Set the last desired version as --home (used to be called global)
	if len(desired) > 0 {
		globalVersion := desired[len(desired)-1]
		if err := exec.Command("asdf", "set", "--home", plugin, globalVersion).Run(); err != nil {
			return fmt.Errorf("failed to set home %s version to %s: %w", plugin, globalVersion, err)
		}

		// Verify --home version was set
		out, err := exec.Command("asdf", "current", "--no-header", plugin).Output()
		if err != nil {
			return fmt.Errorf("failed to get current %s version: %w", plugin, err)
		}
		current := strings.Fields(string(out))[1] // [plugin] [version] [source]
		if current == globalVersion {
			fmt.Printf("✓ - %s %s is set as the home version\n", plugin, globalVersion)
		} else {
			fmt.Printf("✗ - Failed to set %s %s as the home version\n", plugin, globalVersion)
		}
	}

	return nil
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
	//  BECAUSE: asdf list all nodejs: IS BROKEN, we will handle everything
	case plugin == "nodejs":
		return resolveNodeVersion(spec)
	case spec == "latest":
		return resolveLatest(plugin)
	case isVersionPrefix(spec):
		return resolveLatestPatch(plugin, spec)
	default:
		return "", fmt.Errorf("unsupported version spec %q for plugin %q", spec, plugin)
	}
}

// resolveLatest returns the latest stable version for a plugin
// by running asdf latest <plugin>; when not horribley broken
func resolveLatest(plugin string) (string, error) {
	cmd := exec.Command("asdf", "latest", plugin)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output() // Only captures stdout

	if err != nil {
		return "", fmt.Errorf("failed to get latest %s version: %w\nstderr: %s\nNote: Might be due to GitHub API rate limiting (60 requests/hour)\nCommand:\nasdf latest %s", plugin, err, stderr.String(), plugin)
	}
	return strings.TrimSpace(string(out)), nil
}

// resolveNodeVersion returns the appropriate Node.js version based on the spec:
// - "lts": returns the latest LTS version
// - "latest": returns the latest version
// - "X[.Y[.Z]]": returns the latest version matching the prefix
func resolveNodeVersion(spec string) (string, error) {
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

	// three cases:
	// 1. lts
	// 2. latest
	// 3. version prefix
	// You can safely assume the versions are list in descending order
	for _, r := range releases {
		version := strings.TrimPrefix(r.Version, "v")

		switch {
		case spec == "latest":
			return version, nil // First version is latest
		case spec == "lts":
			// when this version is an active LTS version,
			// the LTS field is a release codename like "jod" otherwise it is false
			if r.LTS != false {
				return version, nil
			}
		case isVersionPrefix(spec):
			if strings.HasPrefix(version, spec) {
				return version, nil
			}
		default:
			return "", fmt.Errorf("unsupported Node.js version spec: %s", spec)
		}
	}

	if spec == "lts" {
		return "", fmt.Errorf("no LTS version found")
	}
	if isVersionPrefix(spec) {
		return "", fmt.Errorf("no version found matching prefix %s", spec)
	}
	return "", fmt.Errorf("no matching version found for spec %s", spec)
}

// resolveLatestPatch finds the latest version matching a prefix (like "3.12" for python).
// The process is:
// 1. Get all available versions from asdf list all
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
	fmt.Printf("Resolving latest patch for %s: %s\n", plugin, prefix)
	cmd := exec.Command("asdf", "list", "all", plugin)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list %s versions: %w\nstderr: %s\nCommand:\nasdf list all %s", plugin, err, stderr.String(), plugin)
	}

	versions := strings.Fields(string(out))
	matches := filterAndSortVersions(versions, prefix)
	if len(matches) == 0 {
		return "", fmt.Errorf("no versions found matching %s for %s\nCommand:\nasdf list all %s", prefix, plugin, plugin)
	}

	return matches[len(matches)-1], nil
}

// isVersionPrefix checks if the spec is a valid version prefix (like "3", "3.12", "3.12.0", etc)
func isVersionPrefix(spec string) bool {
	pattern := `^\d+(\.\d+){0,2}$`
	matched, _ := regexp.MatchString(pattern, spec)
	return matched
}

// uniqueVersions returns a sorted list of unique versions from the input slice.
// Duplicates are removed and the result is sorted in ascending order.
func uniqueVersions(versions []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, v := range versions {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return sortVersions(result)
}

// reconcileVersions compares desired and actual versions,
// returning lists of missing (to be installed) and extra (to be removed) versions.
// The comparison is done using sets to efficiently find differences.
func reconcileVersions(desired, actual []string) (missing, extra []string) {
	desiredSet := make(map[string]bool)
	for _, v := range desired {
		desiredSet[v] = true
	}

	actualSet := make(map[string]bool)
	for _, v := range actual {
		actualSet[v] = true
	}

	// Find missing versions
	for v := range desiredSet {
		if !actualSet[v] {
			missing = append(missing, v)
		}
	}

	// Find extra versions
	for v := range actualSet {
		if !desiredSet[v] {
			extra = append(extra, v)
		}
	}

	return missing, extra
}

// performVersionActions installs missing versions and shows removal instructions for extra versions.
// For each missing version:
// - Runs asdf install <plugin> <version>
// - Shows progress and completion messages
// For each extra version:
// - Shows the command to remove it: asdf uninstall <plugin> <version>
func performVersionActions(plugin string, missing, extra []string) error {
	// Install missing versions
	for _, version := range missing {
		fmt.Printf("✗ - %s version %s is missing. Installing...\n", plugin, version)
		if err := exec.Command("asdf", "install", plugin, version).Run(); err != nil {
			return fmt.Errorf("failed to install %s version %s: %w", plugin, version, err)
		}
		fmt.Printf("  ✓ - %s version %s was successfully installed\n", plugin, version)
	}

	// Show extra versions
	if len(extra) > 0 {
		fmt.Printf("✗ - Extraneous %s versions found:\n", plugin)
		for _, version := range extra {
			fmt.Printf("- To remove %s version %s:\n", plugin, version)
			fmt.Printf(" asdf uninstall %s %s\n", plugin, version)
		}
	}

	return nil
}

// getInstalledVersions returns a list of currently installed versions for a plugin
// by running asdf list <plugin> and cleaning up the output:
// - Removes '*' prefix which marks the default/global version
// - Example input:  "  21.7.3\n  22.12.0\n *22.12.0"
// - Example output: ["21.7.3", "22.12.0", "22.12.0"]
func getInstalledVersions(plugin string) ([]string, error) {
	// BECAUSE: asdf list <plugin> now writes "No compatible versions installed" to stderr
	// when no versions are installed, we need to capture stderr to handle this case
	cmd := exec.Command("asdf", "list", plugin)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output() // Only captures stdout

	if err != nil {
		// If stderr contains "No compatible versions installed", return empty slice
		if strings.Contains(stderr.String(), "No compatible versions installed") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to list %s versions: %w\nstderr: %s", plugin, err, stderr.String())
	}

	// Clean up version strings:
	// - Remove '*' prefix which marks the default/global version
	// - Example input:  "  21.7.3\n  22.12.0\n *22.12.0"
	// - Example output: ["21.7.3", "22.12.0", "22.12.0"]
	var versions []string
	for _, v := range strings.Fields(string(out)) {
		v = strings.TrimPrefix(v, "*")
		v = strings.TrimSpace(v)
		if v != "" {
			versions = append(versions, v)
		}
	}
	return versions, nil
}
