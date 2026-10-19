// Package mise reconciles versioned tools through mise.
package mise

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os/exec"
	"slices"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/tool"
)

// Reconcile compares the desired version specs of each versioned tool with
// what mise has installed and with mise's global default versions, and prints
// the mise commands that would close the gap. It never installs, uninstalls or
// sets defaults itself:
// 1. Check if mise is installed (a missing tool is skipped, not an error)
// 2. Observe the actual state: installed versions, global defaults, and the
// newest release matching each version spec
// 3. Plan: compare with the desired state
// 4. Print the report and the commands: installs before uninstalls
func Reconcile(desired map[string][]string) error {
	if !tool.Installed("mise") {
		return nil
	}
	fmt.Printf("✓ - mise is installed\n")

	actual, err := observe(desired)
	if err != nil {
		return err
	}
	p := plan(desired, actual)

	fmt.Printf("\n") // separator
	for _, line := range p.report {
		fmt.Println(line)
	}

	fmt.Printf("\n") // separator
	showCommands(p.install, "missing mise versions")
	showCommands(p.use, "mise defaults to change")
	showCommands(p.uninstall, "unwanted mise versions")
	return nil
}

// observe queries mise for the actual state of the versioned tools.
func observe(desired map[string][]string) (observed, error) {
	installed, err := getInstalledVersions()
	if err != nil {
		return observed{}, err
	}
	defaults, err := getGlobalDefaults()
	if err != nil {
		return observed{}, err
	}
	resolved := map[string]map[string]string{}
	for _, tool := range slices.Sorted(maps.Keys(desired)) {
		resolved[tool] = map[string]string{}
		for _, spec := range desired[tool] {
			version, err := resolveSpec(tool, spec)
			if err != nil {
				return observed{}, err
			}
			resolved[tool][spec] = version
		}
	}
	return observed{installed: installed, defaults: defaults, resolved: resolved}, nil
}

// lsEntry is one version in mise ls --json.
type lsEntry struct {
	Version string `json:"version"`
	Source  struct {
		Path string `json:"path"`
	} `json:"source"`
}

// getInstalledVersions returns each tool's installed versions, by running
// mise ls --installed --json
func getInstalledVersions() (map[string][]string, error) {
	entries, err := miseLs("--installed")
	if err != nil {
		return nil, err
	}
	installed := map[string][]string{}
	for tool, versions := range entries {
		for _, v := range versions {
			installed[tool] = append(installed[tool], v.Version)
		}
	}
	return installed, nil
}

// getGlobalDefaults returns, for each tool in mise's global config, its
// version specs in config order. mise ls --global says which tools the global
// config lists and where it is, but sorts their versions; so the ordered list
// comes from mise config get on that file.
func getGlobalDefaults() (map[string][]string, error) {
	entries, err := miseLs("--global")
	if err != nil {
		return nil, err
	}
	defaults := map[string][]string{}
	for tool, versions := range entries {
		if len(versions) == 0 || versions[0].Source.Path == "" {
			continue
		}
		specs, err := getConfiguredSpecs(versions[0].Source.Path, tool)
		if err != nil {
			return nil, err
		}
		defaults[tool] = specs
	}
	return defaults, nil
}

// miseLs runs mise ls --json with a filter flag and parses the output
func miseLs(filter string) (map[string][]lsEntry, error) {
	out, err := runMise("ls", filter, "--json")
	if err != nil {
		return nil, err
	}
	var entries map[string][]lsEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse mise ls %s output: %w", filter, err)
	}
	return entries, nil
}

// getConfiguredSpecs returns a tool's version specs in one mise config file,
// in order, by running mise config get
func getConfiguredSpecs(configFile, tool string) ([]string, error) {
	out, err := runMise("config", "get", "-f", configFile, "tools."+tool)
	if err != nil {
		return nil, err
	}
	specs, err := parseConfiguredSpecs(string(out))
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s versions in %s: %w", tool, configFile, err)
	}
	return specs, nil
}

// parseConfiguredSpecs parses what mise config get prints for a tool: either
// one spec (latest) or an array (["3.12", "3.11"])
func parseConfiguredSpecs(out string) ([]string, error) {
	value := strings.TrimSpace(out)
	if !strings.HasPrefix(value, "[") {
		return []string{value}, nil
	}
	var specs []string
	if err := json.Unmarshal([]byte(value), &specs); err != nil {
		return nil, fmt.Errorf("%q: %w", value, err)
	}
	return specs, nil
}

// resolveSpec returns the newest release matching a version spec, by running
// mise latest <tool>[@<spec>]. A spec that matches no release is an error.
func resolveSpec(tool, spec string) (string, error) {
	arg := tool + "@" + spec
	if spec == "latest" {
		arg = tool
	}
	out, err := runMise("latest", arg)
	if err != nil {
		return "", err
	}
	// mise latest prints nothing, and exits 0, when no release matches
	version := strings.TrimSpace(string(out))
	if version == "" {
		return "", fmt.Errorf("version spec %q for %s matches no release\nCommand:\nmise latest %s", spec, tool, arg)
	}
	return version, nil
}

// runMise runs mise with the arguments and returns its standard output; a
// failure reports the command and mise's standard error.
func runMise(args ...string) ([]byte, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("mise", args...)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("mise %s failed: %w\nstderr: %s", strings.Join(args, " "), err, stderr.String())
	}
	return out, nil
}

// showCommands prints one kind of mise command, one per line, or a ✓ line
// when there are none.
//
// Example output:
//
//	✗ - Missing mise versions: (2)
//	 mise install python@3.12.15
//	 mise install python@3.11.17
func showCommands(cmds []string, label string) {
	if len(cmds) == 0 {
		fmt.Printf("✓ - No %s\n", label)
		return
	}
	fmt.Printf("✗ - %s%s: (%d)\n", strings.ToUpper(label[:1]), label[1:], len(cmds))
	for _, cmd := range cmds {
		fmt.Printf(" %s\n", cmd)
	}
}
