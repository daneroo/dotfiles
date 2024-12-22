package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/brewdeps/actual"
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/config"
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/desired"
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/reconcile"
	"github.com/daneroo/dotfiles/go/pkg/logsetup"
)

func main() {
	flag.BoolVar(&config.Global.Verbose, "verbose", false, "turn on verbose logging")
	flag.BoolVar(&config.Global.Verbose, "v", false, "turn on verbose logging (shorthand)")
	flag.Parse()
	logsetup.SetupFormat()

	fmt.Printf("--- Checks: --- ( ✓ / ✗ ) \n")

	required := desired.GetRequired()
	state, err := actual.GetState()
	if err != nil {
		fmt.Printf("✗ - Sanity failed: installed > keys(deps)\n")
		log.Fatal(err)
	}

	fmt.Printf("✓ - Sanity passed: installed < keys(deps)\n")

	result := reconcile.Reconcile(required, state)

	if len(result.Missing) > 0 {
		fmt.Printf("✗ - Missing casks/formulae:\n")
		for _, pkg := range result.Missing {
			flag := "--formula"
			if pkg.IsCask {
				flag = "--cask"
			}
			fmt.Printf(" brew install %s %s\n", flag, pkg.Name)
		}
		fmt.Printf("\n  or\n\n")
		// Group by type for combined command
		formulas := []string{}
		casks := []string{}
		for _, pkg := range result.Missing {
			if pkg.IsCask {
				casks = append(casks, pkg.Name)
			} else {
				formulas = append(formulas, pkg.Name)
			}
		}
		if len(formulas) > 0 {
			fmt.Printf(" brew install --formula %s\n", strings.Join(formulas, " "))
		}
		if len(casks) > 0 {
			fmt.Printf(" brew install --cask %s\n", strings.Join(casks, " "))
		}
	} else {
		fmt.Printf("✓ - No missing casks/formulae\n")
	}

	if len(result.Extra) > 0 {
		fmt.Printf("✗ - Extraneous casks/formulae:\n")
		for _, e := range result.Extra {
			flag := "--formula"
			if e.IsCask {
				flag = "--cask"
			}
			fmt.Printf(" brew uninstall %s %s\n", flag, e.Name)
		}
		fmt.Printf("\n  or\n\n")
		// Group by type for combined command
		formulas := []string{}
		casks := []string{}
		for _, pkg := range result.Extra {
			if pkg.IsCask {
				casks = append(casks, pkg.Name)
			} else {
				formulas = append(formulas, pkg.Name)
			}
		}
		if len(formulas) > 0 {
			fmt.Printf(" brew uninstall --formula %s\n", strings.Join(formulas, " "))
		}
		if len(casks) > 0 {
			fmt.Printf(" brew uninstall --cask %s\n", strings.Join(casks, " "))
		}
	} else {
		fmt.Printf("✓ - No extraneous casks/formulae\n")
	}
	fmt.Printf("---\n")

}
