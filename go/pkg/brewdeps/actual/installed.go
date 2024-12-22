package actual

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/daneroo/dotfiles/go/pkg/brewdeps/types"
)

// GetInstalled returns a list of all installed packages (both formulae and casks) by running:
//   - brew ls --full-name --formula
//   - brew ls --full-name --cask
//
// The returned slice is the typed union of both commands, with each package marked
// as either formula or cask. We use --full-name to get tap-qualified names where
// appropriate (unlike dependencies which are always simple names).
//
// This list represents the current state of the system and will be compared against:
//   - Required packages from brewDeps.yaml
//   - Dependencies from brew deps --installed (--formula|--cask)
func GetInstalled(verbose bool) []types.Package {
	var pkgs []types.Package

	configs := []struct {
		arg    string
		isCask bool
	}{
		{"--formula", false},
		{"--cask", true},
	}

	for _, cfg := range configs {
		out, err := exec.Command("brew", "ls", "--full-name", cfg.arg).Output()
		if err != nil {
			log.Fatal(err)
		}
		for _, name := range splitByLineNoEmpty(string(out)) {
			pkgs = append(pkgs, types.Package{Name: name, IsCask: cfg.isCask})
		}
	}

	fmt.Printf("✓ - Got Installed\n")
	if verbose {
		fmt.Printf("Installed: (brew ls --full-name)\n %v\n\n", pkgs)
	}
	return pkgs
}
