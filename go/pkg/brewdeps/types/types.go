package types

// Package represents either a formula or cask in Homebrew
type Package struct {
    Name   string
    IsCask bool
}
