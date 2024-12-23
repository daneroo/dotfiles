package reconcile

import (
	"fmt"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/brewdeps/actual"
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/desired"
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/types"
)

// Reconcile performs a complete reconciliation cycle:
// 1. Takes desired state as input
// 2. Gets actual state from system
// 3. Validates dependency consistency
// 4. Shows actions needed
func Reconcile(desiredState types.DesiredState) error {
	desiredState = desired.GetDesired(desiredState)
	actualState, err := actual.GetActual()
	if err != nil {
		return err
	}
	fmt.Printf("✓ - Dependency map is consistent\n")

	missing := CheckMissing(desiredState.Packages, actualState.Packages)
	extra := Extraneous(desiredState.Packages, actualState.Packages, actualState.DepsMap)

	// TODO: This will evolve into proper Actions that can be shown/executed
	showActions(missing, installAction)
	showActions(extra, uninstallAction)
	fmt.Printf("---\n") // Separator after all actions

	return nil
}

// Internal implementation details below
// ===================================

// actionType combines the verb (install/uninstall) with its state description (Missing/Extraneous)
// This allows us to handle both installation and removal with the same code path,
// while maintaining clear output messages.
type actionType struct {
	verb  string // "install" or "uninstall"
	state string // "Missing" or "Extraneous"
}

var (
	installAction = actionType{
		verb:  "install",
		state: "Missing",
	}
	uninstallAction = actionType{
		verb:  "uninstall",
		state: "Extraneous",
	}
)

// commandOptions configures how commands are displayed
type commandOptions struct {
	isCask       bool
	groupCommand bool // When true, shows single command with all packages
}

// showActions formats and displays the actions needed to reconcile the system.
// It handles both installation and removal of packages in a consistent way:
// 1. Shows a header with the state (Missing/Extraneous)
// 2. Shows individual commands first (one package per line)
// 3. Shows combined commands (all packages in one command)
//
// Example output for missing packages:
//
//	✗ - Missing casks/formulae:
//	 brew install --formula wget
//	 brew install --formula yq
//	 brew install --cask vlc
//
//	or all together:
//
//	 brew install --formula wget yq
//	 brew install --cask vlc
func showActions(pkgs []types.Package, action actionType) {
	if len(pkgs) > 0 {
		fmt.Printf("✗ - %s casks/formulae:\n", action.state)
		// Show individual commands first
		showCommands(pkgs, action.verb, commandOptions{isCask: false, groupCommand: false})
		showCommands(pkgs, action.verb, commandOptions{isCask: true, groupCommand: false})
		// Then show grouped commands
		fmt.Printf("\n  or all together:\n\n")
		showCommands(pkgs, action.verb, commandOptions{isCask: false, groupCommand: true})
		showCommands(pkgs, action.verb, commandOptions{isCask: true, groupCommand: true})
	} else {
		fmt.Printf("✓ - No %s casks/formulae\n", strings.ToLower(action.state))
	}
}

// showCommands shows commands for a filtered set of packages.
// It handles both individual and grouped command display:
//
// Individual mode (groupCommand=false):
//
//	brew install --formula wget
//	brew install --formula yq
//
// Group mode (groupCommand=true):
//
//	brew install --formula wget yq
//
// This separation allows for:
// 1. Clear display of each action for review
// 2. Efficient execution with combined commands
// 3. Proper handling of formula/cask separation
func showCommands(pkgs []types.Package, action string, opts commandOptions) {
	// Filter and collect names
	var names []string
	for _, pkg := range pkgs {
		if pkg.IsCask == opts.isCask {
			names = append(names, pkg.Name)
		}
	}

	// Show commands if we have any matching packages
	if len(names) > 0 {
		flag := "--formula"
		if opts.isCask {
			flag = "--cask"
		}
		if !opts.groupCommand {
			// Individual commands
			for _, name := range names {
				fmt.Printf(" brew %s %s %s\n", action, flag, name)
			}
		} else {
			// Group command
			fmt.Printf(" brew %s %s %s\n", action, flag, strings.Join(names, " "))
		}
	}
}
