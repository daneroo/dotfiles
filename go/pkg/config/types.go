package config

// Package represents either a formula or cask in Homebrew
type Package struct {
	Name   string
	IsCask bool
}

// Config represents the complete configuration for all package managers
type Config struct {
	Homebrew []Package
	Asdf     map[string][]string
	Npm      []string
}
