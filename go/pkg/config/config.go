package config

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

// internal type for parsing - holds the sectioned structure from YAML
type packageConfig struct {
	Homebrew struct {
		FormulaeBySection map[string][]string `yaml:"formulae"`
		Casks             []string            `yaml:"casks"`
	} `yaml:"homebrew"`
	Mise map[string]versionSpecs `yaml:"mise"`
	Npm  []string                `yaml:"npm"`
}

// versionSpecs is a versioned tool's version specs in config.yaml: one
// string, or a list of strings whose first is the default. Each is kept
// exactly as written, for mise to interpret: python: 3.10 stays "3.10".
type versionSpecs []string

// UnmarshalYAML accepts a string or a list of strings. Mise's options
// objects, such as { version = "3.12", os = ["linux"] }, are not supported.
func (v *versionSpecs) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		*v = versionSpecs{}
		if n.Tag != "!!null" {
			*v = versionSpecs{n.Value}
		}
		return nil
	case yaml.SequenceNode:
		specs := versionSpecs{}
		for _, item := range n.Content {
			if item.Kind != yaml.ScalarNode || item.Tag == "!!null" {
				return fmt.Errorf("line %d: a version spec must be a string", item.Line)
			}
			specs = append(specs, item.Value)
		}
		*v = specs
		return nil
	}
	return fmt.Errorf("line %d: version specs must be a string or a list of strings; options objects are not supported", n.Line)
}

// LoadConfig loads and validates the configuration from the specified file
func LoadConfig(configFile string) (*Config, error) {
	out, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", configFile, err)
	}

	cfg, err := ParseConfig(out)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", configFile, err)
	}

	fmt.Printf("✓ - Configuration loaded\n")
	return cfg, nil
}

// ParseConfig parses and validates a configuration document
func ParseConfig(data []byte) (*Config, error) {
	var temp packageConfig
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true) // reject unknown keys, e.g. a misspelled section
	if err := decoder.Decode(&temp); err != nil {
		return nil, fmt.Errorf("parsing: %w", err)
	}

	if err := validateConfig(&temp); err != nil {
		return nil, err
	}

	// After validation passes, flatten into final Config
	var cfg Config
	cfg.Mise = make(map[string][]string, len(temp.Mise))
	for tool, specs := range temp.Mise {
		cfg.Mise[tool] = specs
	}
	cfg.Npm = temp.Npm

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

	return &cfg, nil
}

// validateConfig performs validation on the loaded configuration
func validateConfig(cfg *packageConfig) error {
	var violations []string
	var sectionViolations []string

	// Required sections may be empty ([] or {}), but must be present
	required := []struct {
		name    string
		missing bool
	}{
		{"homebrew.formulae", cfg.Homebrew.FormulaeBySection == nil},
		{"homebrew.casks", cfg.Homebrew.Casks == nil},
		{"mise", cfg.Mise == nil},
		{"npm", cfg.Npm == nil},
	}
	for _, r := range required {
		if r.missing {
			violations = append(violations, fmt.Sprintf("✗ - missing required section %q", r.name))
		}
	}

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

	// Validate mise version specs; their order is meaningful, so not sorted.
	// Only their structure is checked: mise interprets them when reconciling.
	for tool, specs := range cfg.Mise {
		if len(specs) == 0 {
			violations = append(violations, fmt.Sprintf("  ✗ - Tool %q: no version specs", tool))
		}
		for _, spec := range specs {
			if strings.TrimSpace(spec) == "" {
				violations = append(violations, fmt.Sprintf("  ✗ - Tool %q: empty version spec", tool))
			}
		}
	}

	// Validate no list declares the same entry twice
	for section, formulae := range cfg.Homebrew.FormulaeBySection {
		violations = append(violations, validateUnique(fmt.Sprintf("Section %q", section), formulae)...)
	}
	violations = append(violations, validateUnique("Casks", cfg.Homebrew.Casks)...)
	violations = append(violations, validateUnique("NPM packages", cfg.Npm)...)
	for tool, specs := range cfg.Mise {
		violations = append(violations, validateUnique(fmt.Sprintf("Tool %q", tool), specs)...)
	}

	if len(violations) > 0 {
		return fmt.Errorf("validation failed:\n%s", strings.Join(violations, "\n"))
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
		// Equal neighbours are duplicates, reported by validateUnique
		if items[i-1] != items[i] && !compareByBasename(items[i-1], items[i]) {
			violations = append(violations, fmt.Sprintf("%q should come before %q",
				items[i], items[i-1]))
		}
	}
	return violations
}

// validateUnique reports each entry that appears more than once in a list
func validateUnique(list string, items []string) []string {
	var violations []string
	seen := make(map[string]bool)
	for _, item := range items {
		if seen[item] {
			violations = append(violations, fmt.Sprintf("  ✗ - %s: duplicate %q", list, item))
		}
		seen[item] = true
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
