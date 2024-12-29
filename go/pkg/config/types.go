package config

// BrewPackage represents either a formula or cask in Homebrew
type BrewPackage struct {
	Name   string
	IsCask bool
}

// Config represents the complete configuration for all package managers
type Config struct {
	Homebrew []BrewPackage
	Asdf     map[string][]string
	Npm      []string
}
