// go/pkg/asdf/reconcile.go
package asdf

import (
	"fmt"
	"os/exec"
)

// Reconcile performs a complete reconciliation cycle for asdf:
// 1. Check if asdf is installed
// 2. Get actual state (installed plugins)
// 3. Compare with desired state
// 4. Take actions to reconcile differences
func Reconcile(desiredVersions map[string][]string) error {
	// Check if asdf is installed
	if err := exec.Command("command", "-v", "asdf").Run(); err != nil {
		return fmt.Errorf("asdf is not installed")
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
