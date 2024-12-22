package reconcile

import (
    "github.com/daneroo/dotfiles/go/pkg/brewdeps/types"
)

// CheckMissing returns a list of packages that are required but not installed.
func CheckMissing(required, installed []types.Package) []types.Package {
    missing := []types.Package{}
    for _, req := range required {
        if !ContainsPackage(installed, req) {
            missing = append(missing, req)
        }
    }
    return missing
}
