package reconcile

import (
	"github.com/daneroo/dotfiles/go/pkg/brewdeps/types"
)

// Result holds the reconciliation results
type Result struct {
	Missing []types.Package
	Extra   []types.Package
}

// Reconcile compares desired state against actual state
func Reconcile(desired types.DesiredState, actual types.ActualState) Result {
	missing := CheckMissing(desired.Packages, actual.Packages)
	extra := Extraneous(desired.Packages, actual.Packages, actual.DepsMap)

	return Result{
		Missing: missing,
		Extra:   extra,
	}
}
