// Package tool detects missing tools: package managers and completion tools
// whose part of the desired state is reported and skipped when absent.
package tool

import (
	"fmt"
	"os/exec"
)

// Installed reports whether the named tool is on PATH. When it is not, it
// prints that the tool is skipped, so the caller can return early.
func Installed(name string) bool {
	// A PATH lookup, not `command -v`: that is a shell builtin with no
	// executable on Linux, so running it always fails there.
	if _, err := exec.LookPath(name); err != nil {
		fmt.Printf("✗ - %s is not installed, skipping\n", name)
		return false
	}
	return true
}
