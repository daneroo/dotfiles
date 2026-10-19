package npm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"slices"
	"sort"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/tool"
)

// Reconcile compares the desired npm global packages with the installed ones
// and prints the npm commands that would close the gap. It never installs,
// updates or uninstalls anything itself:
// 1. Check if npm is installed (a missing tool is skipped, not an error)
// 2. Get actual state (installed and outdated packages)
// 3. Compare with desired state
// 4. Print the commands that reconcile the differences
func Reconcile(desiredPackages []string) error {
	if !tool.Installed("npm") {
		return nil
	}
	fmt.Printf("✓ - npm is installed\n")

	// Get actual state
	actual, err := getInstalledPackages()
	if err != nil {
		return err
	}
	outdated, err := getOutdatedPackages()
	if err != nil {
		return err
	}

	// Compare with desired state
	missing, extra := reconcilePackages(desiredPackages, actual)

	// Report each package: up to date, outdated, missing or unwanted
	fmt.Printf("\n") // separator
	var toInstall []string
	for _, pkg := range actual {
		if slices.Contains(extra, pkg) {
			continue
		}
		if info, ok := outdated[pkg]; ok {
			fmt.Printf("✗ - npm: %s is outdated (%s → %s)\n", pkg, info.Current, info.Latest)
			toInstall = append(toInstall, pkg)
		} else {
			fmt.Printf("✓ - %s\n", pkg)
		}
	}
	for _, pkg := range missing {
		fmt.Printf("✗ - npm: %s is missing\n", pkg)
		toInstall = append(toInstall, pkg)
	}
	for _, pkg := range extra {
		fmt.Printf("✗ - npm: %s is unwanted\n", pkg)
	}
	sort.Strings(toInstall)

	// Print the commands: installs before uninstalls
	fmt.Printf("\n") // separator
	showCommands(toInstall, "install", "Missing or outdated")
	fmt.Printf("\n") // separator
	showCommands(extra, "uninstall", "Unwanted")

	//  deprecateCorepackPnpm
	fmt.Printf("\n") // separator
	if err := deprecateCorepackPnpm(); err != nil {
		return err
	}

	return nil
}

// getInstalledPackages returns a list of globally installed npm packages
// by running npm ls -g --json and parsing the output
func getInstalledPackages() ([]string, error) {
	out, err := exec.Command("npm", "ls", "-g", "--json", "--depth=0").Output()
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
func getOutdatedPackages() (map[string]outdatedInfo, error) {
	out, err := exec.Command("npm", "outdated", "-g", "--json").Output()
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

// showCommands prints the npm commands for one verb, as the brew reconciler
// does: one command per package, then all packages in one command.
//
// Example output for missing or outdated packages:
//
//	✗ - Missing or outdated npm globals: (2 packages)
//	 npm install -g json
//	 npm install -g serve
//
//	  or all together:
//	 npm install -g json serve
func showCommands(pkgs []string, verb, label string) {
	if len(pkgs) == 0 {
		fmt.Printf("✓ - No %s npm globals\n", strings.ToLower(label))
		return
	}
	fmt.Printf("✗ - %s npm globals: (%d packages)\n", label, len(pkgs))
	for _, pkg := range pkgs {
		fmt.Printf(" npm %s -g %s\n", verb, pkg)
	}
	fmt.Printf("\n  or all together:\n")
	fmt.Printf(" npm %s -g %s\n", verb, strings.Join(pkgs, " "))
}

// deprecateCorepackPnpm prepares corepack for pnpm
func deprecateCorepackPnpm() error {
	fmt.Printf("△ - corepack - DEPRECATED - we now install pnpm with homebrew\n")

	// // Enable corepack
	// if err := exec.Command("corepack", "enable").Run(); err != nil {
	// 	return fmt.Errorf("failed to enable corepack: %w", err)
	// }

	// // Prepare pnpm
	// if err := exec.Command("corepack", "prepare", "pnpm@latest", "--activate").Run(); err != nil {
	// 	return fmt.Errorf("failed to prepare (corepack) pnpm: %w", err)
	// }

	// Show version
	if !tool.Installed("pnpm") {
		return nil
	}
	if out, err := exec.Command("pnpm", "--version").Output(); err == nil {
		fmt.Printf("✓ - pnpm version: %s (homebrew)\n", strings.TrimSpace(string(out)))
		return nil
	}

	return fmt.Errorf("pnpm not working after corepack preparation")
}
