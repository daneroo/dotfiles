package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/daneroo/dotfiles/go/pkg/brewdeps/actual"
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/desired"
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/reconcile"
	"github.com/daneroo/dotfiles/go/pkg/config"
)

func main() {
	// Parse flags
	var (
		verbose    bool
		configFile string
	)
	flag.BoolVar(&verbose, "verbose", false, "turn on verbose logging")
	flag.BoolVar(&verbose, "v", false, "turn on verbose logging (shorthand)")
	flag.StringVar(&configFile, "config", "brewDeps.yaml", "path to brewDeps.yaml config file")
	flag.StringVar(&configFile, "c", "brewDeps.yaml", "path to brewDeps.yaml config file (shorthand)")
	flag.Parse()

	// Set global execution mode
	config.Global.Verbose = verbose

	// Show global flags and config
	fmt.Printf("Global Flags:\n")
	fmt.Printf(" - verbose: %v\n", config.Global.Verbose)
	fmt.Printf("Config: %s\n", configFile)
	fmt.Printf("\n")

	desiredState := desired.Parse(configFile)
	err := reconcile.Reconcile(desiredState)
	if err != nil {
		if validErr, ok := err.(*actual.ValidationError); ok {
			fmt.Printf("✗ - Dependency map inconsistency\n")
			fmt.Printf(" ...%v\n", validErr)
			// exit instead of log.Fatal to avoid duplicate output
			os.Exit(1)
		} else {
			log.Fatal(err)
		}
	}

}
