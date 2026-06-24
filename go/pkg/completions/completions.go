package completions

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// CompletionSpec defines a CLI tool whose bash completion should be cached.
type CompletionSpec struct {
	// Name is the human-readable name (e.g., "docker")
	Name string
	// Command is the executable to run (e.g., "docker")
	Command string
	// Args are the arguments to produce bash completion output (e.g., ["completion", "bash"])
	Args []string
	// OutputFile is the path to the cached completion file,
	// relative to the dotfiles root (e.g., "./core/.config/bash_includes/docker_completion.bash")
	OutputFile string
}

// UpdateCachedCompletion generates a completion script and caches it to disk.
// It only writes the file if the content has changed.
// If the command is not installed, it skips gracefully.
func UpdateCachedCompletion(spec CompletionSpec) error {
	// Check if the command is available
	if _, err := exec.LookPath(spec.Command); err != nil {
		fmt.Printf("△ - %s is not installed, skipping completion cache\n", spec.Name)
		return nil
	}

	// Get current completion text
	out, err := exec.Command(spec.Command, spec.Args...).Output()
	if err != nil {
		return fmt.Errorf("failed to get %s completion: %w", spec.Name, err)
	}
	completionText := string(out)

	// Create parent directory if it doesn't exist
	dir := filepath.Dir(spec.OutputFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Check if file exists and compare content
	current, err := os.ReadFile(spec.OutputFile)
	if err == nil && string(current) == completionText {
		fmt.Printf("✓ - %s completions are up to date (%s)\n", spec.Name, spec.OutputFile)
		return nil
	}

	// Write new completion file
	if err := os.WriteFile(spec.OutputFile, []byte(completionText), 0644); err != nil {
		return fmt.Errorf("failed to write %s completion file: %w", spec.Name, err)
	}
	fmt.Printf("✓ - %s completions were updated (%s)\n", spec.Name, spec.OutputFile)

	return nil
}
