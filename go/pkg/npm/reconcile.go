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
func getInstalledPackages(ctx context.Context, x proc.Exec, npm launcher) ([]string, error) {
	out, err := x.Output(ctx, npm("ls", "-g", "--json", "--depth=0"))
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
func getOutdatedPackages(ctx context.Context, x proc.Exec, npm launcher) (map[string]outdatedInfo, error) {
	out, err := x.Output(ctx, npm("outdated", "-g", "--json"))
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
	// A failure also prints JSON, {"error": {"code": ..., "summary": ...}},
	// which would read as a package named error
	var failure struct {
		Error *struct {
			Code    string `json:"code"`
			Summary string `json:"summary"`
		} `json:"error"`
	}
	if json.Unmarshal(out, &failure) == nil && failure.Error != nil && (failure.Error.Code != "" || failure.Error.Summary != "") {
		return nil, fmt.Errorf("failed to check for updates: npm %s: %s", failure.Error.Code, failure.Error.Summary)
	}
	if err != nil && len(outdated) == 0 {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
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
