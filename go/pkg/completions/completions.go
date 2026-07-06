package completions

import (
	"bytes"
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
//
// Not migrated to v2 on-demand loading: npm/pnpm's shipped completions live
// under versioned Cellar paths that move on every brew upgrade. Caching
// generated content here avoids that.
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

// UpdateGlobalBunCompletionAsASpecialSnowflake fixes bun's global bash
// completion. Unlike pnpm/npm/docker, this can't go through the generic
// CompletionSpec pipeline — bun writes its own fixed output path, and it
// has 2 upstream bugs we patch on every run:
//   - bug 1: wrong filename (bun.completion.bash instead of "bun") breaks
//     bash-completion v2's on-demand loader. https://github.com/oven-sh/bun/issues/671
//   - bug 2: broken regex spams "invalid regular expression" on every
//     bun run <Tab>. https://github.com/oven-sh/bun/issues/24847
//
// We leave bun's own file untouched and write a separate corrected "bun"
// file, so the original stays as proof both bugs are still unfixed.
func UpdateGlobalBunCompletionAsASpecialSnowflake() error {
	homebrewPrefix := os.Getenv("HOMEBREW_PREFIX")
	if homebrewPrefix == "" {
		homebrewPrefix = "/opt/homebrew"
	}

	completionsDir := filepath.Join(homebrewPrefix, "share", "bash-completion", "completions")
	upstreamFile := filepath.Join(completionsDir, "bun.completion.bash") // bug 1: untouched, wrong name
	correctedFile := filepath.Join(completionsDir, "bun")                // fixed, correctly-named copy

	if output, err := exec.Command("bun", "completions").CombinedOutput(); err != nil {
		return fmt.Errorf("bun completions failed: %w (output: %s)", err, output)
	}

	content, err := os.ReadFile(upstreamFile)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", upstreamFile, err)
	}

	// bug 2 fix
	const buggyLine = "local re_prev_script="
	const patchedLine = "return; local re_prev_script="
	if bytes.Contains(content, []byte(buggyLine)) {
		content = bytes.Replace(content, []byte(buggyLine), []byte(patchedLine), 1)
	}

	if err := os.WriteFile(correctedFile, content, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", correctedFile, err)
	}

	fmt.Printf("✓ - bun global completions patched: %s\n", correctedFile)
	return nil
}
