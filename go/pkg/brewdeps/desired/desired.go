package desired

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/brewdeps/types"
	"github.com/daneroo/dotfiles/go/pkg/config"
	"gopkg.in/yaml.v3"
)

type BrewDeps struct {
	FormulaeBySection map[string][]types.Package `yaml:"formulae"`
	Casks             []types.Package            `yaml:"casks"`
}

// simply a passthrough for now (validation happens in Parse)
// Later we will externalize the parsing, and may retain some validation here!
func GetDesired(desired types.DesiredState) types.DesiredState {
	fmt.Printf("✓ - Got Desired\n")
	if config.Global.Verbose {
		fmt.Printf("Desired: (%s)\n %v\n\n", config.Global.ConfigFile, desired)
	}
	return desired
}

// Parse returns the list of required packages from brewDeps.yaml
func Parse(configFile string) types.DesiredState {
	brewDeps, err := parseDeps(configFile)
	if err != nil {
		log.Fatal(err)
	}

	required := make([]types.Package, 0)
	for _, formulae := range brewDeps.FormulaeBySection {
		required = append(required, formulae...)
	}
	required = append(required, brewDeps.Casks...)

	fmt.Printf("✓ - Parsed Desired\n")
	if config.Global.Verbose {
		fmt.Printf("Desired: (%s)\n %v\n\n", configFile, required)
	}
	return types.DesiredState{Packages: required}
}

func validateFormat(pkg types.Package) error {
	parts := strings.Split(pkg.Name, "/")
	if len(parts) != 1 && len(parts) != 3 {
		return fmt.Errorf("invalid format %q: must be 'name' or 'tap/repo/name'", pkg.Name)
	}
	return nil
}

func compareByBasename(i, j types.Package) bool {
	iBase := path.Base(i.Name)
	jBase := path.Base(j.Name)
	if iBase == jBase {
		return i.Name < j.Name // Use full path as tiebreaker
	}
	return iBase < jBase
}

func validateSorting(items []types.Package) []string {
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

func parseDeps(configFile string) (BrewDeps, error) {
	out, err := os.ReadFile(configFile)
	if err != nil {
		return BrewDeps{}, fmt.Errorf("reading %s: %w", configFile, err)
	}

	var temp struct {
		FormulaeBySection map[string][]string `yaml:"formulae"`
		Casks             []string            `yaml:"casks"`
	}
	if err := yaml.Unmarshal(out, &temp); err != nil {
		return BrewDeps{}, fmt.Errorf("parsing %s: %v", configFile, err)
	}

	brewDeps := BrewDeps{
		FormulaeBySection: make(map[string][]types.Package),
		Casks:             make([]types.Package, 0),
	}

	// Convert formulae
	for section, formulae := range temp.FormulaeBySection {
		brewDeps.FormulaeBySection[section] = make([]types.Package, 0)
		for _, f := range formulae {
			brewDeps.FormulaeBySection[section] = append(
				brewDeps.FormulaeBySection[section],
				types.Package{Name: f, IsCask: false},
			)
		}
	}

	// Convert casks
	for _, c := range temp.Casks {
		brewDeps.Casks = append(brewDeps.Casks, types.Package{Name: c, IsCask: true})
	}

	// Validate format and sorting
	var violations []string
	var sectionViolations []string

	// Validate format for all sections
	for section, formulae := range brewDeps.FormulaeBySection {
		for _, f := range formulae {
			if err := validateFormat(f); err != nil {
				violations = append(violations, fmt.Sprintf("  ✗ - Section %q: %v", section, err))
			}
		}
	}

	// Validate format for casks
	for _, c := range brewDeps.Casks {
		if err := validateFormat(c); err != nil {
			violations = append(violations, fmt.Sprintf("  ✗ - Casks: %v", err))
		}
	}

	// Validate all sections
	for section, formulae := range brewDeps.FormulaeBySection {
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
	if sortViolations := validateSorting(brewDeps.Casks); len(sortViolations) > 0 {
		violations = append(violations, "✗ - Casks are not sorted")
		for _, v := range sortViolations {
			violations = append(violations, fmt.Sprintf("  ✗ - %s", v))
		}
	}

	// Print violations and return validation error
	if len(violations) > 0 {
		fmt.Println(strings.Join(violations, "\n"))
		return brewDeps, fmt.Errorf("validation failed for %s", configFile)
	}

	return brewDeps, nil
}