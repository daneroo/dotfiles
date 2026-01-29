package asdf

import (
	"fmt"
	"os/exec"
	"strings"
)

// getActualPlugins returns the list of currently installed plugins
func getActualPlugins() ([]string, error) {
	out, err := exec.Command("asdf", "plugin", "list").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", err)
	}
	return strings.Fields(string(out)), nil
}

// reconcilePlugins determines which plugins need to be installed/removed
func reconcilePlugins(desired []string, actual []string) (missing, extra []string) {
	// Convert to sets
	desiredSet := make(map[string]bool)
	for _, plugin := range desired {
		desiredSet[plugin] = true
	}
	actualSet := make(map[string]bool)
	for _, plugin := range actual {
		actualSet[plugin] = true
	}

	// Find missing (in desired but not actual)
	for plugin := range desiredSet {
		if !actualSet[plugin] {
			missing = append(missing, plugin)
		}
	}

	// Find extra (in actual but not desired)
	for plugin := range actualSet {
		if !desiredSet[plugin] {
			extra = append(extra, plugin)
		}
	}

	return missing, extra
}

// performPluginActions handles installation, updates, and removal hints for plugins
func performPluginActions(desiredVersions map[string][]string, missing, extra []string) error {
	// Install missing plugins
	for _, plugin := range missing {
		fmt.Printf("✗ - asdf plugin %s is missing. Installing\n", plugin)
		if err := exec.Command("asdf", "plugin", "add", plugin).Run(); err != nil {
			return fmt.Errorf("failed to install plugin %s: %w", plugin, err)
		}
		fmt.Printf("✓ - asdf plugin %s is installed\n", plugin)
	}

	// Update all plugins that should be installed (including newly installed ones)
	// Note: missing plugins have been installed above, so we can update them too
	for plugin := range desiredVersions {
		// There is no way to only "check for updates" for a plugin
		// so we just update it and parse the output for status
		updateOut, err := exec.Command("asdf", "plugin", "update", plugin).CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to update plugin %s: %w", plugin, err)
		}
		output := string(updateOut)
		if strings.Contains(output, "Already on 'master'") ||
			strings.Contains(output, "Your branch is up to date") {
			fmt.Printf("✓ - %s plugin is installed and up to date\n", plugin)
		} else {
			fmt.Printf("✓ - %s plugin is installed and was updated\n", plugin)
			fmt.Printf("%s\n", output)
		}
	}

	// Remove extraneous plugins (hint)
	if len(extra) > 0 {
		fmt.Printf("✗ - Extraneous plugins found:\n")
		for _, plugin := range extra {
			fmt.Printf("- To remove %s plugin (and all installed versions):\n", plugin)
			fmt.Printf(" asdf plugin remove %s\n", plugin)
		}
	}

	return nil
}
