package types

// Package represents either a formula or cask in Homebrew
type Package struct {
	Name   string
	IsCask bool
}

// DesiredState represents the packages we want installed
type DesiredState struct {
	Packages []Package
}

// ActualState represents the current system state including dependencies
type ActualState struct {
	Packages []Package
	DepsMap  map[Package][]Package // For determining transitive dependencies
}
