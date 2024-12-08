package main

// 1-Compares requested dependencies: `brewDeps`
// to make sure that are all installed
// 2- makes sure any other installed casks are dependants of the requested ones.

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/logsetup"
	"gopkg.in/yaml.v3"
)

type BrewDeps struct {
	FormulaeBySection map[string][]Package `yaml:"formulae"`
	Casks             []Package            `yaml:"casks"`
}

type Package struct {
	Name   string
	IsCask bool
}

var verbose bool

const (
	brewDepsFile     = "brewDeps"
	brewDepsYamlFile = "brewDeps.yaml"
)

func main() {
	flag.BoolVar(&verbose, "verbose", false, "turn on verbose logging")
	flag.BoolVar(&verbose, "v", false, "turn on verbose logging (shorthand)")
	flag.Parse()
	logsetup.SetupFormat()

	required := getRequired()
	deps := getDeps()
	installed := getInstalled()

	fmt.Printf("--- Checks: --- ( ✓ / ✗ ) \n")

	ok := sanity(installed, deps)
	if ok {
		fmt.Printf("✓ - Sanity passed: installed < keys(deps)\n")
	} else {
		fmt.Printf("✗ - Sanity failed: installed > keys(deps)\n")
	}

	missing := checkMissing(required, installed)
	if len(missing) > 0 {
		fmt.Printf("✗ -Missing casks/formulae:\n")
		for _, pkg := range missing {
			flag := "--formula"
			if pkg.IsCask {
				flag = "--cask"
			}
			fmt.Printf(" brew install %s %s\n", flag, pkg.Name)
		}
		fmt.Printf("\n  or\n\n")
		// Group by type for combined command
		formulas := []string{}
		casks := []string{}
		for _, pkg := range missing {
			if pkg.IsCask {
				casks = append(casks, pkg.Name)
			} else {
				formulas = append(formulas, pkg.Name)
			}
		}
		if len(formulas) > 0 {
			fmt.Printf(" brew install --formula %s\n", strings.Join(formulas, " "))
		}
		if len(casks) > 0 {
			fmt.Printf(" brew install --cask %s\n", strings.Join(casks, " "))
		}
	} else {
		fmt.Printf("✓ - No missing casks/formulae\n")
	}

	// Check if all installed are either required, or a dependant of a required package
	extra := extraneous(required, installed, deps)
	if len(extra) > 0 {
		fmt.Printf("✗ -Extraneous casks/formulae:\n")
		for _, e := range extra {
			flag := "--formula"
			if e.IsCask {
				flag = "--cask"
			}
			fmt.Printf(" brew uninstall %s %s\n", flag, e.Name)
		}
		fmt.Printf("\n  or\n\n")
		// Group by type for combined command
		formulas := []string{}
		casks := []string{}
		for _, pkg := range extra {
			if pkg.IsCask {
				casks = append(casks, pkg.Name)
			} else {
				formulas = append(formulas, pkg.Name)
			}
		}
		if len(formulas) > 0 {
			fmt.Printf(" brew uninstall --formula %s\n", strings.Join(formulas, " "))
		}
		if len(casks) > 0 {
			fmt.Printf(" brew uninstall --cask %s\n", strings.Join(casks, " "))
		}
	} else {
		fmt.Printf("✓ - No extraneous casks/formulae\n")
	}
	fmt.Printf("---\n")

}

// checkMissing returns a list of packages that are required but not installed.
// The caller (main) will use this information to suggest appropriate
// `brew install --formula` or `brew install --cask` commands.
func checkMissing(required, installed []Package) []Package {
	missing := []Package{}
	for _, req := range required {
		if !containsPackage(installed, req) {
			missing = append(missing, req)
		}
	}
	return missing
}

// containsPackage checks if a Package is in a slice, matching both Name and IsCask.
// Used by checkMissing to compare required vs installed packages.
func containsPackage(s []Package, e Package) bool {
	for _, a := range s {
		if a.Name == e.Name && a.IsCask == e.IsCask {
			return true
		}
	}
	return false
}

// isTransitiveDep checks if pkg is a transitive dependency of any required package.
// A package is considered a transitive dependency if it is:
//  1. A direct dependency of a required package
//  2. A direct dependency of a required package's dependency (depth=2)
//
// The seen map is used to avoid cycles in the dependency graph.
// Note: We only check up to depth 2 because:
//   - Direct dependencies (depth 1): pkg is directly required by a required package
//   - Dependencies of dependencies (depth 2): pkg is required by a dependency
//   - We don't check deeper to avoid counting distant transitive dependencies
func isTransitiveDep(pkg Package, required []Package, deps map[Package][]Package, seen map[Package]bool) bool {
	if seen[pkg] {
		return false // avoid cycles
	}
	seen[pkg] = true

	// Direct dependency of a required package?
	for _, req := range required {
		if pkgDeps, ok := deps[req]; ok {
			// Is pkg directly required by req?
			if containsPackage(pkgDeps, pkg) {
				return true
			}
			// Is pkg a dependency of any of req's dependencies?
			for _, dep := range pkgDeps {
				// Only check if pkg is a dependency of dep
				if containsPackage(deps[dep], pkg) {
					return true
				}
			}
		}
	}
	return false
}

func extraneous(required, installed []Package, deps map[Package][]Package) []Package {
	extra := []Package{}
	for _, inst := range installed {
		ok := false
		if containsPackage(required, inst) {
			ok = true
			if verbose {
				fmt.Printf(" - %s is required\n", inst.Name)
			}
		} else {
			seen := make(map[Package]bool)
			ok = isTransitiveDep(inst, required, deps, seen)
			if ok {
				if verbose {
					fmt.Printf(" - %s is required transitively\n", inst.Name)
				}
			}
		}
		if !ok {
			extra = append(extra, inst)
			fmt.Printf(" - %s is not required (transitively)\n", inst.Name)
		}
	}
	return extra
}

// sanity checks if all installed packages appear as keys in the deps map.
// This verifies that `brew deps --installed` returns dependency information
// for every installed package.
//
// Note: This is a precondition for the extraneous check, which assumes
// we can look up dependencies for any installed package.
func sanity(installed []Package, deps map[Package][]Package) bool {
	// Sanity: make sure all installed appear as a key in deps
	insane := false
	for _, inst := range installed {
		_, ok := deps[inst]
		if !ok {
			insane = true
			fmt.Printf("(In)Sanity: Installed package %q (cask=%v) not present in dependencies\n",
				inst.Name, inst.IsCask)
		}
	}
	return !insane
}

func getRequired() []Package {
	config, err := parseDeps()
	if err != nil {
		log.Fatal(err)
	}

	required := make([]Package, 0)

	// Process all sections
	for _, formulae := range config.FormulaeBySection {
		required = append(required, formulae...)
	}
	required = append(required, config.Casks...)

	fmt.Printf("✓ - Got Required\n")
	if verbose {
		fmt.Printf("Required: (%s)\n %v\n\n", brewDepsYamlFile, required)
	}
	return required
}

// getDeps returns a map of installed packages to their dependencies by running:
//   - brew deps --installed --formula
//   - brew deps --installed --cask
//
// Note: We don't use --full-name for deps because:
//   - Dependencies are always formulae (not casks)
//   - Simple names are sufficient for dependencies
//   - Tap qualification is only needed for explicitly required packages
//
// The returned map includes both formulae and casks as keys, but all dependencies
// (values) are formulae, following Homebrew's (ASSUMED) rules:
//   - Formulae can depend on other formulae
//   - Casks can depend on formulae
//   - Neither can depend on casks
func getDeps() map[Package][]Package {
	deps := make(map[Package][]Package)

	configs := []struct {
		arg    string
		isCask bool
	}{
		{"--formula", false},
		{"--cask", true},
	}

	for _, cfg := range configs {
		out, err := exec.Command("brew", "deps", "--installed", cfg.arg).Output()
		if err != nil {
			log.Fatal(err)
		}
		for _, line := range splitByLineNoEmpty(string(out)) {
			ss := strings.SplitN(line, ":", 2)
			if len(ss) != 2 {
				log.Fatalf("Cannot split(:) %q \n", line)
			}
			pkg := Package{Name: ss[0], IsCask: cfg.isCask}
			// Note: dependencies are always formulae
			deps[pkg] = parseDepsAsPackages(ss[1])
		}
	}

	fmt.Printf("✓ - Got Deps\n")
	if verbose {
		fmt.Printf("Deps: (brew deps --installed)\n %v\n\n", deps)
	}
	return deps
}

// parseDepsAsPackages converts a space-separated string of package names into Package objects.
// Note: All dependencies (the right-hand side of `brew deps` output) are assumed to be formulae,
// not casks. This matches Homebrew's (ASSUMED) rules where:
//   - Formulae can depend on other formulae
//   - Casks can depend on formulae
//   - Neither can depend on casks
func parseDepsAsPackages(s string) []Package {
	var deps []Package
	for _, name := range strings.Fields(s) {
		// All dependencies are formulae
		deps = append(deps, Package{Name: name, IsCask: false})
	}
	return deps
}

// getInstalled returns a list of all installed packages (both formulae and casks) by running:
//   - brew ls --full-name --formula
//   - brew ls --full-name --cask
//
// The returned slice is the typed union of both commands, with each package marked
// as either formula or cask. We use --full-name to get tap-qualified names where
// appropriate (unlike dependencies which are always simple names).
//
// This list represents the current state of the system and will be compared against:
//   - Required packages from brewDeps.yaml
//   - Dependencies from brew deps --installed (--formula|--cask)
func getInstalled() []Package {
	var pkgs []Package

	configs := []struct {
		arg    string
		isCask bool
	}{
		{"--formula", false},
		{"--cask", true},
	}

	for _, cfg := range configs {
		out, err := exec.Command("brew", "ls", "--full-name", cfg.arg).Output()
		if err != nil {
			log.Fatal(err)
		}
		for _, name := range splitByLineNoEmpty(string(out)) {
			pkgs = append(pkgs, Package{Name: name, IsCask: cfg.isCask})
		}
	}

	fmt.Printf("✓ - Got Installed\n")
	if verbose {
		fmt.Printf("Installed: (brew ls --full-name)\n %v\n\n", pkgs)
	}
	return pkgs
}

func splitByLineNoEmpty(s string) []string {
	return filter(
		strings.Split(s, "\n"),
		nonEmptyString,
	)
}

func nonEmptyString(s string) bool {
	return len(s) > 0
}

func filter(vs []string, f func(string) bool) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

func parseDeps() (BrewDeps, error) {
	out, err := os.ReadFile(brewDepsYamlFile)
	if err != nil {
		return BrewDeps{}, fmt.Errorf("reading %s: %w", brewDepsYamlFile, err)
	}

	// Temporary struct for YAML parsing
	var temp struct {
		FormulaeBySection map[string][]string `yaml:"formulae"`
		Casks             []string            `yaml:"casks"`
	}
	if err := yaml.Unmarshal(out, &temp); err != nil {
		return BrewDeps{}, fmt.Errorf("parsing %s: %v", brewDepsYamlFile, err)
	}

	// Convert to Package types
	config := BrewDeps{
		FormulaeBySection: make(map[string][]Package),
		Casks:             make([]Package, 0),
	}

	// Convert formulae
	for section, formulae := range temp.FormulaeBySection {
		config.FormulaeBySection[section] = make([]Package, 0)
		for _, f := range formulae {
			config.FormulaeBySection[section] = append(
				config.FormulaeBySection[section],
				Package{Name: f, IsCask: false},
			)
		}
	}

	// Convert casks
	for _, c := range temp.Casks {
		config.Casks = append(config.Casks, Package{Name: c, IsCask: true})
	}

	var violations []string
	var sectionViolations []string

	// Validate format for all sections
	for section, formulae := range config.FormulaeBySection {
		for _, f := range formulae {
			if err := validateFormat(f); err != nil {
				violations = append(violations, fmt.Sprintf("  ✗ - Section %q: %v", section, err))
			}
		}
	}

	// Validate format for casks
	for _, c := range config.Casks {
		if err := validateFormat(c); err != nil {
			violations = append(violations, fmt.Sprintf("  ✗ - Casks: %v", err))
		}
	}

	// Validate all sections
	for section, formulae := range config.FormulaeBySection {
		if sortViolations := validateSorting(formulae); len(sortViolations) > 0 {
			sectionViolations = append(sectionViolations, fmt.Sprintf("  ✗ - Section %q is not sorted", section))
			for _, v := range sortViolations {
				sectionViolations = append(sectionViolations, fmt.Sprintf("    ✗ - %s", v))
			}
		}
	}

	// If we have any section violations, add the header
	if len(sectionViolations) > 0 {
		violations = append(violations, "✗ - Formulae are not sorted")
		violations = append(violations, sectionViolations...)
	}

	// Validate casks
	if sortViolations := validateSorting(config.Casks); len(sortViolations) > 0 {
		violations = append(violations, "✗ - Casks are not sorted")
		for _, v := range sortViolations {
			violations = append(violations, fmt.Sprintf("  ✗ - %s", v))
		}
	}

	// Print violations and return validation error
	if len(violations) > 0 {
		fmt.Println(strings.Join(violations, "\n"))
		return config, fmt.Errorf("validation failed for %s", brewDepsYamlFile)
	}

	return config, nil
}

// validateFormat checks if a formula/cask name is either:
// - simple name: [a-zA-Z0-9-]+
// - or fully qualified: name/tap/name
func validateFormat(pkg Package) error {
	parts := strings.Split(pkg.Name, "/")
	if len(parts) != 1 && len(parts) != 3 {
		return fmt.Errorf("invalid format %q: must be 'name' or 'tap/repo/name'", pkg.Name)
	}
	return nil
}

// For use with validateSorting
func compareByBasename(i, j Package) bool {
	iBase := path.Base(i.Name)
	jBase := path.Base(j.Name)
	if iBase == jBase {
		return i.Name < j.Name // Use full path as tiebreaker
	}
	return iBase < jBase
}

func validateSorting(items []Package) []string {
	// Empty or single item is always sorted
	if len(items) <= 1 {
		return []string{}
	}

	var violations []string
	for i := 1; i < len(items); i++ {
		if !compareByBasename(items[i-1], items[i]) {
			violations = append(violations, fmt.Sprintf("%q should come before %q",
				items[i].Name, items[i-1].Name))
		}
	}
	return violations
}
