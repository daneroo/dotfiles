package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/daneroo/dotfiles/go/pkg/asdf"
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/actual"
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/reconcile"
	"github.com/daneroo/dotfiles/go/pkg/config"
	"github.com/daneroo/dotfiles/go/pkg/npm"
)

func main() {
	f := parseFlags()

	// Set global execution mode
	config.Global.Verbose = f.verbose

	// Show global flags and config
	fmt.Printf("Global Flags:\n")
	fmt.Printf(" - verbose: %v\n", config.Global.Verbose)
	fmt.Printf("Config: %s\n", f.configFile)

	// Load configuration
	fmt.Printf("\n## Loading Configuration\n\n")
	cfg, err := config.LoadConfig(f.configFile)
	if err != nil {
		log.Fatal(err)
	}

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

	err = reconcile.Reconcile(cfg.Homebrew)
	if err != nil {
		handleError(err)
	}

	fmt.Printf("\n## ASDF Section\n\n")
	// Handle asdf plugins and versions
	if err := asdf.Reconcile(cfg.Asdf); err != nil {
		fmt.Printf("✗ - %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n## NPM Globals Section\n\n")
	// Handle npm global packages
	if err := npm.Reconcile(cfg.Npm); err != nil {
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
	flag.StringVar(&f.configFile, "config", "config.yaml", "path to config file")
	flag.StringVar(&f.configFile, "c", "config.yaml", "path to config file (shorthand)")
	flag.Parse()
	return f
}
