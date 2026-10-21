// Package mise reconciles versioned tools through mise.
package mise

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/proc"
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

	actual, err := observe(context.Background(), proc.OS{}, desired)
	if err != nil {
		return err
	}
	p := plan(desired, actual)

	fmt.Printf("\n") // separator
	for _, n := range p.report {
		fmt.Printf("%s - %s\n", n.Mark, n.Text)
	}

	install, use, uninstall, unuse := printedCommands(p)
	fmt.Printf("\n") // separator
	showCommands(install, "missing mise versions")
	showCommands(use, "mise defaults to change")
	showCommands(uninstall, "unwanted mise versions")
	showCommands(unuse, "strays in mise's global config")
	return nil
}

// printedCommands returns the commands Reconcile prints for a plan.
func printedCommands(p planned) (install, use, uninstall, unuse []string) {
	for _, v := range p.install {
		install = append(install, "mise install "+v.String())
	}
	for _, a := range p.align {
		args := make([]string, len(a.specs))
		for i, spec := range a.specs {
			args[i] = a.tool + "@" + spec
		}
		use = append(use, "mise use -g "+strings.Join(args, " "))
	}
	for _, v := range p.uninstall {
		uninstall = append(uninstall, "mise uninstall "+v.String())
	}
	for _, t := range p.unuse {
		unuse = append(unuse, "mise unuse -g "+t)
	}
	return install, use, uninstall, unuse
}

// observe queries mise for the actual state of the versioned tools.
func observe(ctx context.Context, x proc.Exec, desired map[string][]string) (observed, error) {
	installed, err := getInstalledVersions(ctx, x)
	if err != nil {
		return observed{}, err
	}
	defaults, globalConfig, err := getGlobalDefaults(ctx, x)
	if err != nil {
		return observed{}, err
	}
	resolved := map[string]map[string]string{}
	for _, tool := range slices.Sorted(maps.Keys(desired)) {
		resolved[tool] = map[string]string{}
		for _, spec := range desired[tool] {
			version, err := resolveSpec(ctx, x, tool, spec)
			if err != nil {
				return observed{}, err
			}
			resolved[tool][spec] = version
		}
	}
	return observed{installed: installed, defaults: defaults, resolved: resolved, globalConfig: globalConfig}, nil
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
func getInstalledVersions(ctx context.Context, x proc.Exec) (map[string][]string, error) {
	entries, err := miseLs(ctx, x, "--installed")
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
// version specs in config order, and the config's path. mise ls --global says
// which tools the global config lists and where it is, but sorts their
// versions; so the ordered list comes from mise config get --global, which
// reads the file reconfig edits. A global request from another file, such
// as a .tool-versions, is an error: reconfig would not edit it.
func getGlobalDefaults(ctx context.Context, x proc.Exec) (map[string][]string, string, error) {
	entries, err := miseLs(ctx, x, "--global")
	if err != nil {
		return nil, "", err
	}
	defaults := map[string][]string{}
	var globalConfig string
	for _, tool := range slices.Sorted(maps.Keys(entries)) {
		versions := entries[tool]
		if len(versions) == 0 || versions[0].Source.Path == "" {
			continue
		}
		source := versions[0].Source.Path
		specs, err := getConfiguredSpecs(ctx, x, tool)
		if err != nil {
			return nil, "", fmt.Errorf("%s's global versions come from %s, which reconfig cannot read as mise's global config file: %w", tool, source, err)
		}
		globalConfig = source
		defaults[tool] = specs
	}
	return defaults, globalConfig, nil
}

// miseLs runs mise ls --json with a filter flag and parses the output
func miseLs(ctx context.Context, x proc.Exec, filter string) (map[string][]lsEntry, error) {
	out, err := x.Output(ctx, proc.Command("mise", "ls", filter, "--json"))
	if err != nil {
		return nil, err
	}
	var entries map[string][]lsEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse mise ls %s output: %w", filter, err)
	}
	return entries, nil
}

// getConfiguredSpecs returns a tool's version specs in mise's global config,
// in order, by running mise config get --global
func getConfiguredSpecs(ctx context.Context, x proc.Exec, tool string) ([]string, error) {
	out, err := x.Output(ctx, proc.Command("mise", "config", "get", "--global", "tools."+tool))
	if err != nil {
		return nil, err
	}
	specs, err := parseConfiguredSpecs(string(out))
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s versions in mise's global config: %w", tool, err)
	}
	return specs, nil
}

// parseConfiguredSpecs parses what mise config get prints for a tool: either
// one spec (latest) or an array (["3.12", "3.11"]). An options object, such
// as { version = "3.12" }, is not supported: aligning it would lose options.
func parseConfiguredSpecs(out string) ([]string, error) {
	value := strings.TrimSpace(out)
	if value == "" || strings.HasPrefix(value, "{") {
		return nil, fmt.Errorf("unsupported value %q: reconfig handles a string or a list of strings", value)
	}
	if !strings.HasPrefix(value, "[") {
		return []string{value}, nil
	}
	var specs []string
	if err := json.Unmarshal([]byte(value), &specs); err != nil {
		return nil, fmt.Errorf("unsupported value %q: reconfig handles a string or a list of strings: %w", value, err)
	}
	return specs, nil
}

// resolveSpec returns the newest release matching a version spec, by running
// mise latest <tool>[@<spec>]. A spec that matches no release is an error.
func resolveSpec(ctx context.Context, x proc.Exec, tool, spec string) (string, error) {
	arg := tool + "@" + spec
	if spec == "latest" {
		arg = tool
	}
	out, err := x.Output(ctx, proc.Command("mise", "latest", arg))
	if err != nil {
		return "", err
	}
	// mise latest prints nothing, and exits 0, when no release matches
	version := strings.TrimSpace(string(out))
	if version == "" {
		return "", fmt.Errorf("version spec %q for %s matches no release\nCommand:\n%s", spec, tool, proc.Format("mise", "latest", arg))
	}
	return version, nil
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
