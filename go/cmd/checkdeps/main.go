package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/daneroo/dotfiles/go/pkg/asdf"
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/actual"
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/desired"
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/reconcile"
	"github.com/daneroo/dotfiles/go/pkg/config"
	"github.com/daneroo/dotfiles/go/pkg/npm"
)

func main() {
	f := parseFlags()

	// Set global execution mode
	config.Global.Verbose = f.verbose

	// Desired asdf plugins and versions - TODO: move to config file
	desiredVersions := map[string][]string{
		"nodejs": {"lts", "21", "22"}, // Latest LTS version
		"python": {"3.12", "3.11"},    // Multiple versions, latest patch
		"deno":   {"latest"},          // Latest stable
		"bun":    {"latest"},          // Latest stable
	}

	// Show global flags and config
	fmt.Printf("Global Flags:\n")
	fmt.Printf(" - verbose: %v\n", config.Global.Verbose)
	fmt.Printf("Config: %s\n", f.configFile)

	fmt.Printf("\n## Brew Section\n\n")
	// Check for updates first
	hasUpdates, err := actual.CheckOutdated()
	if err != nil {
		handleError(err)
	}
	if hasUpdates {
		fmt.Printf("\nNote: Must resolve outdated packages before proceeding with brewDeps reconciliation\n")
		fmt.Printf("      because outdated packages can break dependency resolution\n")
		os.Exit(1) // Exit before reconciliation if updates needed
	}

	desiredState := desired.Parse(f.configFile)
	err = reconcile.Reconcile(desiredState)
	if err != nil {
		handleError(err)
	}

	fmt.Printf("\n## ASDF Section\n\n")
	// Handle asdf plugins and versions
	if err := asdf.Reconcile(desiredVersions); err != nil {
		fmt.Printf("✗ - %v\n", err)
		os.Exit(1)
	}

	// Desired npm global packages
	npmGlobalPackages := []string{
		"corepack",
		"eslint",
		"json",
		// "turbo",
		"nx",
		"pino-pretty",
		"serve",
		"standard",
		"typescript",
		"vercel",
		"npm",
	}

	fmt.Printf("\n## NPM Globals Section\n\n")
	// Handle npm global packages
	if err := npm.Reconcile(npmGlobalPackages); err != nil {
		fmt.Printf("✗ - %v\n", err)
		os.Exit(1)
	}
}

// handleError handles both validation errors and unexpected errors
func handleError(err error) {
	if validErr, ok := err.(*actual.ValidationError); ok {
		fmt.Printf("✗ - Dependency map inconsistency\n")
		fmt.Printf(" ...%v\n", validErr)
		// exit instead of log.Fatal to avoid duplicate output
		os.Exit(1)
	}
	log.Fatal(err)
}

type flags struct {
	verbose    bool
	configFile string
	// TODO: Add execution mode flags
	// dryRun bool - Show commands vs Execute them
	// force bool - Skip confirmation
}

func parseFlags() flags {
	f := flags{}
	flag.BoolVar(&f.verbose, "verbose", false, "turn on verbose logging")
	flag.BoolVar(&f.verbose, "v", false, "turn on verbose logging (shorthand)")
	flag.StringVar(&f.configFile, "config", "brewDeps.yaml", "path to brewDeps.yaml config file")
	flag.StringVar(&f.configFile, "c", "brewDeps.yaml", "path to brewDeps.yaml config file (shorthand)")
	flag.Parse()
	return f
}
