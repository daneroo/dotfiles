package desired

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/brewdeps/config"
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/types"
	"gopkg.in/yaml.v3"
)

type BrewDeps struct {
	FormulaeBySection map[string][]types.Package `yaml:"formulae"`
	Casks             []types.Package            `yaml:"casks"`
}

const (
	brewDepsFile     = "brewDeps"
	brewDepsYamlFile = "brewDeps.yaml"
)

// GetRequired returns the list of required packages from brewDeps.yaml
func GetRequired() []types.Package {
	brewDeps, err := parseDeps()
	if err != nil {
		log.Fatal(err)
	}

	required := make([]types.Package, 0)
	for _, formulae := range brewDeps.FormulaeBySection {
		required = append(required, formulae...)
	}
	required = append(required, brewDeps.Casks...)

	fmt.Printf("✓ - Got Required\n")
	if config.Global.Verbose {
		fmt.Printf("Required: (%s)\n %v\n\n", brewDepsYamlFile, required)
	}
	return required
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

func parseDeps() (BrewDeps, error) {
	out, err := os.ReadFile(brewDepsYamlFile)
	if err != nil {
		return BrewDeps{}, fmt.Errorf("reading %s: %w", brewDepsYamlFile, err)
	}

	var temp struct {
		FormulaeBySection map[string][]string `yaml:"formulae"`
		Casks             []string            `yaml:"casks"`
	}
	if err := yaml.Unmarshal(out, &temp); err != nil {
		return BrewDeps{}, fmt.Errorf("parsing %s: %v", brewDepsYamlFile, err)
	}

	config := BrewDeps{
		FormulaeBySection: make(map[string][]types.Package),
		Casks:             make([]types.Package, 0),
	}

	// Convert formulae
	for section, formulae := range temp.FormulaeBySection {
		config.FormulaeBySection[section] = make([]types.Package, 0)
		for _, f := range formulae {
			config.FormulaeBySection[section] = append(
				config.FormulaeBySection[section],
				types.Package{Name: f, IsCask: false},
			)
		}
	}

	// Convert casks
	for _, c := range temp.Casks {
		config.Casks = append(config.Casks, types.Package{Name: c, IsCask: true})
	}

	// Validate format and sorting
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
