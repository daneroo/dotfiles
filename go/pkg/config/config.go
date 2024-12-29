package config

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// GlobalMutableState holds the ONLY piece of global state we allow in the entire codebase.
// We made exactly ONE exception to the "no global state" rule:
// - Verbose: because passing it through every single function would be impractical
// NEVER add anything else here. No, really. Don't even think about it.
type GlobalMutableState struct {
	Verbose bool
}

// Global is the ONLY global variable in the entire codebase
var Global = GlobalMutableState{
	Verbose: false,
}

// internal type for parsing - holds the sectioned structure from YAML
type packageConfig struct {
	Homebrew struct {
		FormulaeBySection map[string][]string `yaml:"formulae"`
		Casks             []string            `yaml:"casks"`
	} `yaml:"homebrew"`
	Asdf map[string][]string `yaml:"asdf"`
	Npm  []string            `yaml:"npm"`
}

// LoadConfig loads and validates the configuration from the specified file
func LoadConfig(configFile string) (*Config, error) {
	out, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", configFile, err)
	}

	var temp packageConfig
	if err := yaml.Unmarshal(out, &temp); err != nil {
		return nil, fmt.Errorf("parsing %s: %v", configFile, err)
	}

	if err := validateConfig(&temp); err != nil {
		return nil, err
	}

	// After validation passes, flatten into final Config
	var cfg Config
	cfg.Npm = temp.Npm
	cfg.Asdf = temp.Asdf

	// Convert and flatten formulae sections and casks into []Package
	cfg.Homebrew = make([]BrewPackage, 0)
	for _, formulae := range temp.Homebrew.FormulaeBySection {
		for _, f := range formulae {
			cfg.Homebrew = append(cfg.Homebrew, BrewPackage{Name: f, IsCask: false})
		}
	}
	for _, c := range temp.Homebrew.Casks {
		cfg.Homebrew = append(cfg.Homebrew, BrewPackage{Name: c, IsCask: true})
	}

	fmt.Printf("✓ - Configuration loaded\n")
	return &cfg, nil
}

// validateConfig performs validation on the loaded configuration
func validateConfig(cfg *packageConfig) error {
	var violations []string
	var sectionViolations []string

	// Validate format for all sections
	for section, formulae := range cfg.Homebrew.FormulaeBySection {
		for _, f := range formulae {
			if err := validateBrewPackageFormat(f); err != nil {
				violations = append(violations, fmt.Sprintf("  ✗ - Section %q: %v", section, err))
			}
		}
	}

	// Validate format for casks
	for _, c := range cfg.Homebrew.Casks {
		if err := validateBrewPackageFormat(c); err != nil {
			violations = append(violations, fmt.Sprintf("  ✗ - Casks: %v", err))
		}
	}

	// Validate all sections
	for section, formulae := range cfg.Homebrew.FormulaeBySection {
		if sortViolations := validateSorting(formulae); len(sortViolations) > 0 {
			sectionViolations = append(sectionViolations, fmt.Sprintf("  ✗ - Section %q is not sorted", section))
			for _, v := range sortViolations {
				sectionViolations = append(violations, fmt.Sprintf("    ✗ - %s", v))
			}
		}
	}

	// If we have any section violations, add the header
	if len(sectionViolations) > 0 {
		violations = append(violations, "✗ - Formulae are not sorted")
		violations = append(violations, sectionViolations...)
	}

	// Validate casks
	if sortViolations := validateSorting(cfg.Homebrew.Casks); len(sortViolations) > 0 {
		violations = append(violations, "✗ - Casks are not sorted")
		for _, v := range sortViolations {
			violations = append(violations, fmt.Sprintf("  ✗ - %s", v))
		}
	}

	// Validate npm packages are sorted
	if sortViolations := validateSorting(cfg.Npm); len(sortViolations) > 0 {
		violations = append(violations, "✗ - NPM packages are not sorted")
		for _, v := range sortViolations {
			violations = append(violations, fmt.Sprintf("  ✗ - %s", v))
		}
	}

	// Validate asdf plugin versions
	for plugin, versions := range cfg.Asdf {
		for _, version := range versions {
			if err := validateAsdfVersion(version, plugin); err != nil {
				violations = append(violations, fmt.Sprintf("  ✗ - Plugin %q: %v", plugin, err))
			}
		}
	}

	if len(violations) > 0 {
		fmt.Println(strings.Join(violations, "\n"))
		return fmt.Errorf("validation failed for config")
	}

	return nil
}

func validateBrewPackageFormat(pkg string) error {
	parts := strings.Split(pkg, "/")
	if len(parts) != 1 && len(parts) != 3 {
		return fmt.Errorf("invalid format %q: must be 'name' or 'tap/repo/name'", pkg)
	}
	return nil
}

func validateSorting(items []string) []string {
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

// compareByBasename compares two package names by their basename
func compareByBasename(i, j string) bool {
	iBase := path.Base(i)
	jBase := path.Base(j)
	if iBase == jBase {
		return i < j // Use full path as tiebreaker
	}
	return iBase < jBase
}

// validateAsdfVersion validates version format for asdf plugins
// Supported formats:
// - "latest": resolves to the latest stable version (using asdf latest <plugin>)
// - "lts": for nodejs only, resolves to the latest LTS version from nodejs.org
// - "X[.Y[.Z]]": resolves to the latest version matching the prefix:
//   - "3" -> latest 3.x.x
//   - "3.12" -> latest 3.12.x
//   - "3.12.0" -> exact version
func validateAsdfVersion(version string, plugin string) error {
	if version == "latest" {
		return nil
	}
	if version == "lts" {
		if plugin != "nodejs" {
			return fmt.Errorf("version %q is only valid for nodejs, not for %q", version, plugin)
		}
		return nil
	}

	pattern := `^\d+(\.\d+){0,2}$`
	matched, _ := regexp.MatchString(pattern, version)
	if !matched {
		if plugin == "nodejs" {
			return fmt.Errorf("invalid version format %q: must be 'latest', 'lts', or 'X[.Y[.Z]]'", version)
		}
		return fmt.Errorf("invalid version format %q: must be 'latest' or 'X[.Y[.Z]]'", version)
	}
	return nil
}
