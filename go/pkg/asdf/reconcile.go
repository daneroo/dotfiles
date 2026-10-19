// go/pkg/asdf/reconcile.go
package asdf

import (
	"fmt"

	"github.com/daneroo/dotfiles/go/pkg/tool"
)

// Reconcile performs a complete reconciliation cycle for asdf:
// 1. Check if asdf is installed (a missing tool is skipped, not an error)
// 2. Get actual state (installed plugins)
// 3. Compare with desired state
// 4. Take actions to reconcile differences
func Reconcile(desiredVersions map[string][]string) error {
	if !tool.Installed("asdf") {
		return nil
	}
	fmt.Printf("✓ - asdf is installed\n")

	// Get actual state of installed plugins
	actualPlugins, err := getActualPlugins()
	if err != nil {
		return err
	}

	// Get list of desired plugins the keys of the desiredVersions map
	var desiredPlugins []string
	for plugin := range desiredVersions {
		desiredPlugins = append(desiredPlugins, plugin)
	}

	// Determine required actions
	missing, extra := reconcilePlugins(desiredPlugins, actualPlugins)

	// Perform all plugin actions
	if err := performPluginActions(desiredVersions, missing, extra); err != nil {
		return err
	}

	// Show version resolution
	for plugin, specs := range desiredVersions {
		if err := reconcileVersionsForPlugin(plugin, specs); err != nil {
			return err
		}
	}

	return nil
}
