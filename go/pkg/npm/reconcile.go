package npm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/daneroo/dotfiles/go/pkg/proc"
)

// getInstalledPackages returns a list of globally installed npm packages
// by running npm ls -g --json and parsing the output
func getInstalledPackages(ctx context.Context, x proc.Exec) ([]string, error) {
	out, err := x.Output(ctx, proc.Command("npm", "ls", "-g", "--json", "--depth=0"))
	if err != nil {
		return nil, fmt.Errorf("failed to list global packages: %w", err)
	}

	var npmOutput struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}

	if err := json.Unmarshal(out, &npmOutput); err != nil {
		return nil, fmt.Errorf("failed to parse npm output: %w", err)
	}

	var packages []string
	for pkg := range npmOutput.Dependencies {
		packages = append(packages, pkg)
	}
	sort.Strings(packages) // Sort for consistent output
	return packages, nil
}

// outdatedInfo is one package's entry in npm outdated -g --json.
type outdatedInfo struct {
	Current string `json:"current"`
	Latest  string `json:"latest"`
}

// getOutdatedPackages returns the global packages that have a newer release,
// keyed by name, by running npm outdated -g --json and parsing the output
func getOutdatedPackages(ctx context.Context, x proc.Exec) (map[string]outdatedInfo, error) {
	out, err := x.Output(ctx, proc.Command("npm", "outdated", "-g", "--json"))
	// npm outdated exits 1 when updates are available, still printing JSON
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}

	outdated := map[string]outdatedInfo{}
	if len(bytes.TrimSpace(out)) == 0 {
		return outdated, nil
	}
	if err := json.Unmarshal(out, &outdated); err != nil {
		return nil, fmt.Errorf("failed to parse npm outdated output: %w", err)
	}
	return outdated, nil
}

// reconcilePackages determines which packages need to be installed/removed,
// each sorted by name
func reconcilePackages(desired, actual []string) (missing, extra []string) {
	desiredSet := make(map[string]bool)
	for _, pkg := range desired {
		desiredSet[pkg] = true
	}

	actualSet := make(map[string]bool)
	for _, pkg := range actual {
		actualSet[pkg] = true
	}

	// Find missing packages
	for pkg := range desiredSet {
		if !actualSet[pkg] {
			missing = append(missing, pkg)
		}
	}

	// Find extra packages
	for pkg := range actualSet {
		if !desiredSet[pkg] {
			extra = append(extra, pkg)
		}
	}

	sort.Strings(missing)
	sort.Strings(extra)
	return missing, extra
}
