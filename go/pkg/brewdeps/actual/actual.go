package actual

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/brewdeps/config"
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/types"
)

// State represents the current state of the system
// type State struct {
// 	Installed []types.Package
// 	Deps      map[types.Package][]types.Package
// }

// GetActual returns the current state of installed packages and their dependencies
func GetActual() (types.ActualState, error) {
	deps := GetDeps(config.Global.Verbose)
	installed := GetInstalled(config.Global.Verbose)

	if !Validate(installed, deps) {
		return types.ActualState{}, fmt.Errorf("validation failed: some installed packages missing from deps")
	}

	return types.ActualState{
		Packages: installed,
		DepsMap:  deps,
	}, nil
}

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

// GetDeps returns a map of installed packages to their dependencies by running:
//   - brew deps --installed --formula
//   - brew deps --installed --cask
//
// Note: We don't use --full-name for deps because:
//   - Dependencies are always formulae (not casks)
//   - Simple names are sufficient for dependencies
//   - Tap qualification is only needed for explicitly required packages
//
// The returned map includes both formulae and casks as keys, but all dependencies
// (values) are formulae, following Homebrew's (ASSUMED) rules:
//   - Formulae can depend on other formulae
//   - Casks can depend on formulae
//   - Neither can depend on casks
func GetDeps(verbose bool) map[types.Package][]types.Package {
	deps := make(map[types.Package][]types.Package)

	configs := []struct {
		arg    string
		isCask bool
	}{
		{"--formula", false},
		{"--cask", true},
	}

	for _, cfg := range configs {
		out, err := exec.Command("brew", "deps", "--installed", cfg.arg).Output()
		if err != nil {
			log.Fatal(err)
		}
		for _, line := range splitByLineNoEmpty(string(out)) {
			ss := strings.SplitN(line, ":", 2)
			if len(ss) != 2 {
				log.Fatalf("Cannot split(:) %q \n", line)
			}
			pkg := types.Package{Name: ss[0], IsCask: cfg.isCask}
			// Note: dependencies are always formulae
			deps[pkg] = parseDepsAsPackages(ss[1])
		}
	}

	fmt.Printf("✓ - Got Deps\n")
	if verbose {
		fmt.Printf("Deps: (brew deps --installed)\n %v\n\n", deps)
	}
	return deps
}

// parseDepsAsPackages converts a space-separated string of package names into Package objects.
// Note: All dependencies (the right-hand side of `brew deps` output) are assumed to be formulae,
// not casks. This matches Homebrew's (ASSUMED) rules where:
//   - Formulae can depend on other formulae
//   - Casks can depend on formulae
//   - Neither can depend on casks
func parseDepsAsPackages(s string) []types.Package {
	var deps []types.Package
	for _, name := range strings.Fields(s) {
		// All dependencies are formulae
		deps = append(deps, types.Package{Name: name, IsCask: false})
	}
	return deps
}

func splitByLineNoEmpty(s string) []string {
	return filter(
		strings.Split(s, "\n"),
		nonEmptyString,
	)
}

func nonEmptyString(s string) bool {
	return len(s) > 0
}

func filter(vs []string, f func(string) bool) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

// Validate checks if all installed packages appear as keys in the deps map.
// This verifies that `brew deps --installed` returns dependency information
// for every installed package.
//
// Note: This is a precondition for the extraneous check, which assumes
// we can look up dependencies for any installed package.
func Validate(installed []types.Package, deps map[types.Package][]types.Package) bool {
	insane := false
	for _, inst := range installed {
		_, ok := deps[inst]
		if !ok {
			insane = true
			fmt.Printf("(In)Sanity: Installed package %q (cask=%v) not present in dependencies\n",
				inst.Name, inst.IsCask)
		}
	}
	return !insane
}
