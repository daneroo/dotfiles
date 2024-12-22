package reconcile

import (
    "github.com/daneroo/dotfiles/go/pkg/brewdeps/actual"
    "github.com/daneroo/dotfiles/go/pkg/brewdeps/types"
)

// Result holds the reconciliation results
type Result struct {
    Missing []types.Package
    Extra   []types.Package
}

// Reconcile compares desired state against actual state
func Reconcile(required []types.Package, state actual.State) Result {
    missing := CheckMissing(required, state.Installed)
    extra := Extraneous(required, state.Installed, state.Deps)
    
    return Result{
        Missing: missing,
        Extra:   extra,
    }
}
