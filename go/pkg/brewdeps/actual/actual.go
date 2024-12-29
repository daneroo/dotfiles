package actual

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/brewdeps/types"
	"github.com/daneroo/dotfiles/go/pkg/config"
)

// State represents the current state of the system
// type State struct {
// 	Installed []types.Package
// 	Deps      map[types.Package][]types.Package
// }

// GetActual returns the current state of installed packages and their dependencies
func GetActual() (types.ActualState, error) {
	depsMap := GetDepsMap(config.Global.Verbose)
	installed := GetInstalled(config.Global.Verbose)

	if err := Validate(installed, depsMap); err != nil {
		return types.ActualState{}, err
	}

	return types.ActualState{
		Packages: installed,
		DepsMap:  depsMap,
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

// GetDepsMap returns a map of installed packages to their dependencies by running:
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
func GetDepsMap(verbose bool) map[types.Package][]types.Package {
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

	fmt.Printf("✓ - Got Dependency Map\n")
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
func Validate(installed []types.Package, depsMap map[types.Package][]types.Package) error {
	var missingFromMap []types.Package
	for _, inst := range installed {
		_, ok := depsMap[inst]
		// force an inconsistency to test output and error handling
		// if inst.Name == "git" || inst.Name == "vlc" {
		// 	ok = false
		// }
		if !ok {
			missingFromMap = append(missingFromMap, inst)
		}
	}
	if len(missingFromMap) > 0 {
		return &ValidationError{MissingFromDepsMap: missingFromMap}
	}
	return nil
}

// ValidationError represents an inconsistency between installed packages
// and the dependency map. This indicates that some installed packages
// are not present as keys in the deps map, which is a precondition
// for the extraneous check.
type ValidationError struct {
	MissingFromDepsMap []types.Package
}

func (e *ValidationError) Error() string {
	var msgs []string
	for _, pkg := range e.MissingFromDepsMap {
		msgs = append(msgs, fmt.Sprintf("%q (cask=%v)", pkg.Name, pkg.IsCask))
	}
	return fmt.Sprintf("dependency map inconsistency: installed packages not found in deps map: %s",
		strings.Join(msgs, ", "))
}
