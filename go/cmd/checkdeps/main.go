package main

// 1-Compares requested dependencies: `brewDeps`
// to make sure that are all installed
// 2- makes sure any other installed casks are dependants of the requested ones.

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/logsetup"
	"gopkg.in/yaml.v3"
)

type BrewDeps struct {
	FormulaeBySection map[string][]string `yaml:"formulae"`
	Casks             []string            `yaml:"casks"`
}

var verbose = false

const (
	brewDepsFile     = "brewDeps"
	brewDepsYamlFile = "brewDeps.yaml"
)

func main() {
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

func extraneous(required, installed []string, deps map[string][]string) []string {
	extra := []string{}
	for _, inst := range installed {
		ok := false
		if contains(required, inst) {
			ok = true
			// fmt.Printf(" - %s is required\n", inst)
		} else {
			// if cask is required, then it's deps are OK
			for cask, deps := range deps {
				if contains(required, cask) {
					if contains(deps, inst) {
						ok = true
						// fmt.Printf(" - %s is required transitively by %s\n", inst, cask)
					}
				}
			}
		}
		if !ok {
			extra = append(extra, inst)
			fmt.Printf(" - %s is not required (transitevely)\n", inst)
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
			fmt.Printf("(In)Sanity: Installed package %s not present in dependancies\n", inst)
		}
	}
	return !insane
}

func getRequiredOld() []string {
	out, err := os.ReadFile("brewDeps")
	if err != nil {
		log.Fatal(err)
	}
	// remove empty lines, and lines starting with # (comment)
	required := filter(
		strings.Split(string(out), "\n"),
		func(s string) bool {
			return len(s) > 0 && !strings.HasPrefix(strings.TrimSpace(s), "#")
		})

	// Trim entries
	for i := 0; i < len(required); i++ {
		required[i] = strings.TrimSpace(required[i])
	}

	fmt.Printf("✓ - Got Required (old)\n")
	if verbose {
		fmt.Printf("Required: (./brewDeps)\n %s\n\n", strings.Join(required, ", "))
	}
	return required
}

func getRequiredNew() []string {
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

	fmt.Printf("✓ - Got Required (new)\n")
	if verbose {
		fmt.Printf("Required: (./brewDeps.yaml)\n %s\n\n", strings.Join(required, ", "))
	}
	return required
}

func getRequired() []string {
	oldReq := getRequiredOld()
	newReq := getRequiredNew()

	// Sort both lists before comparing
	sort.Strings(oldReq)
	sort.Strings(newReq)

	// Compare sorted slices
	if !slicesEqual(oldReq, newReq) {
		log.Fatalf("Mismatch between brewDeps and brewDeps.yaml:\nOld: %v\nNew: %v", oldReq, newReq)
	}
	fmt.Printf("✓ - Required old and new are consistent\n")

	// During transition, return old version (unsorted)
	return getRequiredOld() // Return fresh copy of old version
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

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	// Make copies to sort
	aCopy := make([]string, len(a))
	bCopy := make([]string, len(b))
	copy(aCopy, a)
	copy(bCopy, b)
	sort.Strings(aCopy)
	sort.Strings(bCopy)

	for i := range aCopy {
		if aCopy[i] != bCopy[i] {
			return false
		}
	}
	return true
}

func validateSorting(items []string) []string {
	sorted := make([]string, len(items))
	copy(sorted, items)
	sort.Strings(sorted)

	var violations []string
	for i := range items {
		if items[i] != sorted[i] {
			if sorted[i] < items[i] {
				violations = append(violations, fmt.Sprintf("%q should come before %q",
					sorted[i], items[i]))
			}
		}
	}
	return violations
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
