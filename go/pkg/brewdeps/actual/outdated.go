package actual

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type outdatedFormula struct {
	Name              string   `json:"name"`
	InstalledVersions []string `json:"installed_versions"`
	CurrentVersion    string   `json:"current_version"`
}

type outdatedResponse struct {
	Formulae []outdatedFormula `json:"formulae"`
	Casks    []outdatedFormula `json:"casks"`
}

// CheckOutdated returns true if any packages need updating
// First runs brew update to ensure we have latest information
func CheckOutdated() (bool, error) {
	// Run brew update first
	if err := exec.Command("brew", "update").Run(); err != nil {
		return false, fmt.Errorf("brew update failed: %w", err)
	}

	out, err := exec.Command("brew", "outdated", "--json").Output()
	if err != nil {
		return false, err
	}

	var response outdatedResponse
	if err := json.Unmarshal(out, &response); err != nil {
		return false, err
	}

	hasUpdates := len(response.Formulae) > 0 || len(response.Casks) > 0
	if !hasUpdates {
		fmt.Printf("✓ - No updates available\n")
		return false, nil
	}

	fmt.Printf("✗ - Updates available: (%d packages)\n",
		len(response.Formulae)+len(response.Casks))

	// Show individual updates
	for _, f := range response.Formulae {
		fmt.Printf(" - %s: %s -> %s\n",
			f.Name, f.InstalledVersions[0], f.CurrentVersion)
	}
	for _, c := range response.Casks {
		fmt.Printf(" - %s: %s -> %s\n",
			c.Name, c.InstalledVersions[0], c.CurrentVersion)
	}

	fmt.Printf("\nRun:\n")
	fmt.Printf(" brew upgrade && brew cleanup\n")
	return true, nil
}
