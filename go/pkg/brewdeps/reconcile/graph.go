package reconcile

import (
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/types"
)

// ContainsPackage checks if a Package is in a slice, matching both Name and IsCask.
func ContainsPackage(s []types.Package, e types.Package) bool {
	for _, a := range s {
		if a.Name == e.Name && a.IsCask == e.IsCask {
			return true
		}
	}
	return false
}

// IsTransitiveDep checks if pkg is required (directly or transitively).
func IsTransitiveDep(pkg types.Package, required []types.Package, depsMap map[types.Package][]types.Package) bool {
	if ContainsPackage(required, pkg) {
		return true
	}

	for _, req := range required {
		if reqDeps, ok := depsMap[req]; ok {
			if ContainsPackage(reqDeps, pkg) {
				return true
			}
			if IsTransitiveDep(pkg, reqDeps, depsMap) {
				return true
			}
		}
	}
	return false
}
