package reconcile

import (
    "fmt"
    "log"
    "github.com/daneroo/dotfiles/go/pkg/brewdeps/types"
)

var verbose bool

// Extraneous returns a list of installed packages that are not required (directly or transitively).
func Extraneous(required, installed []types.Package, deps map[types.Package][]types.Package) []types.Package {
    extra := []types.Package{}
    for _, inst := range installed {
        ok := IsTransitiveDep(inst, required, deps)
        if ok {
            if verbose {
                fmt.Printf(" - %s is required (directly or transitively)\n", inst.Name)
            }
        } else {
            extra = append(extra, inst)
            fmt.Printf(" - %s is not required (transitively)\n", inst.Name)
        }
    }

    minimalExtra := minimizeExtraneous(extra, deps)
    if len(minimalExtra) == 0 && len(extra) > 0 {
        log.Fatal("Impossible: found extraneous packages but no minimal set - circular dependencies?")
    }
    return minimalExtra
}

func minimizeExtraneous(extra []types.Package, deps map[types.Package][]types.Package) []types.Package {
    isDependency := make(map[types.Package]bool)
    for _, pkg := range extra {
        if pkgDeps, ok := deps[pkg]; ok {
            for _, dep := range pkgDeps {
                if ContainsPackage(extra, dep) {
                    isDependency[dep] = true
                }
            }
        }
    }

    var roots []types.Package
    for _, pkg := range extra {
        if !isDependency[pkg] {
            roots = append(roots, pkg)
        } else {
            fmt.Printf(" - %s is a dependency of another extraneous package, so skipping\n", pkg.Name)
        }
    }
    return roots
}
