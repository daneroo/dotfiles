package config

// BrewPackage represents either a formula or cask in Homebrew
type BrewPackage struct {
	Name   string
	IsCask bool
}

// Config represents the complete configuration for all package managers
type Config struct {
	Homebrew []BrewPackage
	// Mise maps each versioned tool to its version specs; the first is the default
	Mise map[string][]string
	Npm  []string
}
