package npm

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"sort"
	"strings"
)

// Reconcile performs a complete reconciliation cycle for npm global packages:
// 1. Check if npm is installed
// 2. Get actual state (installed packages)
// 3. Compare with desired state
// 4. Take actions to reconcile differences
func Reconcile(desiredPackages []string) error {
	// Check if npm is installed
	if err := exec.Command("command", "-v", "npm").Run(); err != nil {
		return fmt.Errorf("npm is not installed")
	}
	fmt.Printf("✓ - npm is installed\n")

	// Get actual installed packages
	actual, err := getInstalledPackages()
	if err != nil {
		return err
	}

	// Reconcile differences
	missing, extra := reconcilePackages(desiredPackages, actual)

	// Show already installed packages - if not extraneous
	fmt.Printf("\n") // separator
	for _, pkg := range actual {
		if !slices.Contains(extra, pkg) {
			fmt.Printf("✓ - %s\n", pkg)
		}
	}

	// Install missing packages
	fmt.Printf("\n") // separator
	if err := performPackageActions(missing, extra); err != nil {
		return err
	}

	// Check for updates
	fmt.Printf("\n") // separator
	if err := checkOutdated(); err != nil {
		return err
	}
	//  prepareCorepackPnpm
	fmt.Printf("\n") // separator
	if err := prepareCorepackPnpm(); err != nil {
		return err
	}

	// update npm completions (this is really more a=of a config file...)
	fmt.Printf("\n") // separator
	if err := updateNpmCompletions(); err != nil {
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

// reconcilePackages determines which packages need to be installed/removed
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

	return missing, extra
}

// performPackageActions installs missing packages
func performPackageActions(missing, extra []string) error {
	// Install missing packages
	for _, pkg := range missing {
		fmt.Printf("✗ - npm: %s is missing. Installing...\n", pkg)
		if err := exec.Command("npm", "install", "-g", pkg).Run(); err != nil {
			return fmt.Errorf("failed to install %s: %w", pkg, err)
		}
		fmt.Printf("  ✓ - npm: %s was successfully installed\n", pkg)
	}

	// Show extra packages
	if len(extra) > 0 {
		fmt.Printf("\n✗ - Extraneous npm packages found:\n")
		for _, pkg := range extra {
			fmt.Printf("- To remove %s:\n", pkg)
			fmt.Printf("  npm uninstall -g %s\n", pkg)
		}
	}

	return nil
}

// checkOutdated checks for available updates in global packages
func checkOutdated() error {
	// Get outdated packages in JSON format
	out, err := exec.Command("npm", "outdated", "-g", "--json").Output()
	if err != nil {
		// npm outdated returns exit code 1 if updates are available
		if len(out) > 0 {
			var outdated map[string]struct {
				Current string `json:"current"`
				Wanted  string `json:"wanted"`
				Latest  string `json:"latest"`
			}
			if err := json.Unmarshal(out, &outdated); err != nil {
				return fmt.Errorf("failed to parse npm outdated output: %w", err)
			}

			// Update each outdated package
			for pkg, info := range outdated {
				fmt.Printf("✗ - npm: %s needs update (%s -> %s). Installing...\n", pkg, info.Current, info.Latest)
				if err := exec.Command("npm", "install", "-g", pkg).Run(); err != nil {
					return fmt.Errorf("failed to update %s: %w", pkg, err)
				}
				fmt.Printf("  ✓ - npm: %s was successfully updated\n", pkg)
			}
			return nil
		}
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	fmt.Printf("✓ - All global packages are up to date\n")
	return nil
}

// prepareCorepackPnpm prepares corepack for pnpm
func prepareCorepackPnpm() error {
	// Enable corepack
	if err := exec.Command("corepack", "enable").Run(); err != nil {
		return fmt.Errorf("failed to enable corepack: %w", err)
	}

	// Prepare pnpm
	if err := exec.Command("corepack", "prepare", "pnpm@latest", "--activate").Run(); err != nil {
		return fmt.Errorf("failed to prepare (corepack) pnpm: %w", err)
	}

	// Show version
	if out, err := exec.Command("pnpm", "--version").Output(); err == nil {
		fmt.Printf("✓ - pnpm version: %s (corepack)\n", strings.TrimSpace(string(out)))
		return nil
	}

	return fmt.Errorf("pnpm not working after corepack preparation")
}

// updateNpmCompletions updates the npm completion script if needed
func updateNpmCompletions() error {
	completionFile := "./incl/npm_completion.sh"

	// Get current completion text
	out, err := exec.Command("npm", "completion").Output()
	if err != nil {
		return fmt.Errorf("failed to get npm completion: %w", err)
	}
	completionText := string(out)

	// Create incl directory if it doesn't exist
	if err := os.MkdirAll("./incl", 0755); err != nil {
		return fmt.Errorf("failed to create incl directory: %w", err)
	}

	// Check if file exists and compare content
	current, err := os.ReadFile(completionFile)
	if err == nil && string(current) == completionText {
		fmt.Printf("✓ - npm completions are up to date (%s)\n", completionFile)
		return nil
	}

	// Write new completion file
	if err := os.WriteFile(completionFile, []byte(completionText), 0644); err != nil {
		return fmt.Errorf("failed to write completion file: %w", err)
	}
	fmt.Printf("✓ - npm completions were updated (%s)\n", completionFile)

	return nil
}
