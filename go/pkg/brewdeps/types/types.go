package types

import "github.com/daneroo/dotfiles/go/pkg/config"

// Package represents either a formula or cask in Homebrew
type Package = config.Package

// ActualState represents the current system state including dependencies
type ActualState struct {
	Packages []Package
	DepsMap  map[Package][]Package // For determining transitive dependencies
}
