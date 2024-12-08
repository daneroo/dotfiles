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
	FormulaeBySection map[string][]string `yaml:"formulae"`
	Casks             []string            `yaml:"casks"`
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
		for _, cask := range missing {
			fmt.Printf(" brew install %s\n", cask)
		}
		fmt.Printf("\n  or\n\n")
		fmt.Printf(" brew install %s\n", strings.Join(missing, " "))
	} else {
		fmt.Printf("✓ - No missing casks/formulae\n")
	}

	// Check if all installed are either required, or a dependant of a required package
	extra := extraneous(required, installed, deps)
	if len(extra) > 0 {
		fmt.Printf("✗ -Extraneous casks/formulae: (brew uninstall or brew rmtree)\n")
		for _, e := range extra {
			fmt.Printf(" brew uninstall %s\n", e)
		}
		fmt.Printf("\n  or\n\n")
		fmt.Printf(" brew uninstall %s\n\n", strings.Join(extra, " "))
	} else {
		fmt.Printf("✓ - No extraneous casks/formulae\n")
	}
	fmt.Printf("---\n")

}

func checkMissing(required, installed []string) []string {
	missing := []string{}
	for _, req := range required {
		if !contains(installed, req) {
			missing = append(missing, req)
			// fmt.Printf(" - required cask: %s is missing\n", req)
		} else {
			// fmt.Printf(" - required cask: %s is installed\n", req)
		}
	}
	return missing
}

func isTransitiveDep(pkg string, required []string, deps map[string][]string, seen map[string]bool) bool {
	if seen[pkg] {
		return false // avoid cycles
	}
	seen[pkg] = true

	// Direct dependency of a required package?
	for _, req := range required {
		if pkgDeps, ok := deps[req]; ok {
			if contains(pkgDeps, pkg) {
				return true
			}
			// Check deps of deps
			for _, dep := range pkgDeps {
				if isTransitiveDep(pkg, []string{dep}, deps, seen) {
					return true
				}
			}
		}
	}
	return false
}

func extraneous(required, installed []string, deps map[string][]string) []string {
	extra := []string{}
	for _, inst := range installed {
		ok := false
		if contains(required, inst) {
			ok = true
			if verbose {
				fmt.Printf(" - %s is required\n", inst)
			}
		} else {
			seen := make(map[string]bool)
			ok = isTransitiveDep(inst, required, deps, seen)
			if ok {
				if verbose {
					fmt.Printf(" - %s is required transitively\n", inst)
				}
			}
		}
		if !ok {
			extra = append(extra, inst)
			fmt.Printf(" - %s is not required (transitively)\n", inst)
		}
	}
	return extra
}

func sanity(installed []string, deps map[string][]string) bool {
	// Sanity: make sure all installed appear as a key in deps
	insane := false
	for _, inst := range installed {
		_, ok := deps[inst]
		if !ok {
			insane = true
			fmt.Printf("(In)Sanity: Installed package %s not present in dependencies\n", inst)
		}
	}
	return !insane
}

func getRequired() []string {
	config, err := parseDeps()
	if err != nil {
		log.Fatal(err)
	}

	required := make([]string, 0)
	// Process all sections
	for _, formulae := range config.FormulaeBySection {
		for _, f := range formulae {
			if strings.Contains(f, ",") {
				parts := strings.Split(f, ",")
				basename := strings.TrimSpace(parts[0])
				tap := strings.TrimSpace(parts[1])
				tap = strings.TrimSuffix(tap, "/")
				required = append(required, tap+"/"+basename)
			} else {
				required = append(required, f)
			}
		}
	}
	required = append(required, config.Casks...)

	fmt.Printf("✓ - Got Required\n")
	if verbose {
		fmt.Printf("Required: (%s)\n %s\n\n", brewDepsYamlFile, strings.Join(required, ", "))
	}
	return required
}

func getDeps() map[string][]string {
	// Parse the output of: brew deps --installed
	// asciinema: gdbm openssl python readline sqlite xz
	// aws-iam-authenticator:
	out, err := exec.Command("brew", "deps", "--installed").Output()
	if err != nil {
		log.Fatal(err)
	}

	// split by line, remove empty lines
	installedColonDeps := spliyByLineNoEmpty(string(out))

	deps := map[string][]string{}

	for _, line := range installedColonDeps {
		ss := strings.SplitN(line, ":", 2)
		if len(ss) != 2 {
			log.Fatalf("Cannot split(:) %q \n", line)
		}
		c := ss[0]
		ds := strings.Fields(ss[1])
		deps[c] = ds
	}

	fmt.Printf("✓ - Got Deps\n")
	if verbose {
		fmt.Printf("Deps: (brew deps --installed)\n %v\n\n", deps)
	}
	return deps
}

func getInstalled() []string {
	// list both formulae and casks
	out, err := exec.Command("brew", "ls", "--full-name").Output()
	// we previously excluded casks
	// out, err := exec.Command("brew", "ls", "--formula", "--full-name").Output()
	if err != nil {
		log.Fatal(err)
	}

	// split by line, remove empty lines
	fmt.Printf("✓ - Got Installed\n")
	installed := spliyByLineNoEmpty(string(out))
	if verbose {
		fmt.Printf("Installed: (brew ls --full-name)\n %v\n\n", strings.Join(installed, ", "))
	}
	return installed
}

func spliyByLineNoEmpty(s string) []string {
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

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func parseDeps() (BrewDeps, error) {
	out, err := os.ReadFile(brewDepsYamlFile)
	if err != nil {
		return BrewDeps{}, fmt.Errorf("reading %s: %w", brewDepsYamlFile, err)
	}

	var config BrewDeps
	if err := yaml.Unmarshal(out, &config); err != nil {
		return BrewDeps{}, fmt.Errorf("parsing %s: %v", brewDepsYamlFile, err)
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
func validateFormat(name string) error {
	parts := strings.Split(name, "/")
	if len(parts) != 1 && len(parts) != 3 {
		return fmt.Errorf("invalid format %q: must be 'name' or 'tap/repo/name'", name)
	}
	return nil
}

// For use with validateSorting
func compareByBasename(i, j string) bool {
	iBase := path.Base(i)
	jBase := path.Base(j)
	if iBase == jBase {
		return i < j // Use full path as tiebreaker
	}
	return iBase < jBase
}

func validateSorting(items []string) []string {
	// Empty or single item is always sorted
	if len(items) <= 1 {
		return []string{}
	}

	var violations []string
	for i := 1; i < len(items); i++ {
		if !compareByBasename(items[i-1], items[i]) {
			violations = append(violations, fmt.Sprintf("%q should come before %q",
				items[i], items[i-1]))
		}
	}
	return violations
}
