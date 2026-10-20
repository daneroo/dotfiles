package packages_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/brewdeps/upgrade"
	"github.com/daneroo/dotfiles/go/pkg/proc"
)

// fpkg is one formula or cask a controlled brew knows.
type fpkg struct {
	installed bool
	// deps are the declared formula dependencies; caskDeps a cask's
	// required casks.
	deps, caskDeps []string
	// recorded are dependencies brew info does not declare but records once
	// the formula is installed.
	recorded []string
	// hidden are dependencies only the native uninstall guard knows.
	hidden []string
	// sideEffects are formulae installing this one also installs, unasked.
	sideEffects []string
	aliases     []string
	outdated    bool
}

// fakeBrew is a controlled brew: install installs a package and its
// dependencies, and uninstall refuses to remove a package that another
// installed package outside the removal set depends on.
type fakeBrew struct {
	missing         bool
	formulae, casks map[string]*fpkg
	// fail makes the command with this display form fail.
	fail string
	// outputs and runs record the observations and the changes.
	outputs, runs []string
}

func (f *fakeBrew) LookPath(name string) (string, error) {
	if name != "brew" || f.missing {
		return "", errors.New("not found")
	}
	return "/fake/bin/brew", nil
}

func (f *fakeBrew) Output(_ context.Context, c proc.Cmd) ([]byte, error) {
	cmd := c.String()
	f.outputs = append(f.outputs, cmd)
	if cmd == f.fail {
		return nil, fmt.Errorf("%s failed: exit status 1", cmd)
	}
	switch {
	case cmd == "brew outdated --json=v2":
		var outdated []map[string]any
		for _, name := range slices.Sorted(maps.Keys(f.formulae)) {
			if p := f.formulae[name]; p.installed && p.outdated {
				outdated = append(outdated, map[string]any{"name": name, "installed_versions": []string{"1"}, "current_version": "2"})
			}
		}
		return json.Marshal(map[string]any{"formulae": outdated, "casks": []any{}})
	case cmd == "brew info --json=v2 --installed":
		return f.info(f.installedNames(f.formulae), f.installedNames(f.casks))
	case strings.HasPrefix(cmd, "brew info --json=v2 --formula "):
		return f.info(c.Args[3:], nil)
	case strings.HasPrefix(cmd, "brew info --json=v2 --cask "):
		return f.info(nil, c.Args[3:])
	}
	return nil, fmt.Errorf("unexpected command: %s", cmd)
}

func (f *fakeBrew) installedNames(pkgs map[string]*fpkg) []string {
	var names []string
	for _, name := range slices.Sorted(maps.Keys(pkgs)) {
		if pkgs[name].installed {
			names = append(names, name)
		}
	}
	return names
}

// info renders brew info --json=v2 for the named packages, failing as brew
// does for an unknown name.
func (f *fakeBrew) info(formulae, casks []string) ([]byte, error) {
	resp := map[string][]map[string]any{"formulae": {}, "casks": {}}
	for _, name := range formulae {
		p, ok := f.formulae[name]
		if !ok {
			return nil, fmt.Errorf("brew info failed: No available formula with the name %q", name)
		}
		var installed []map[string]any
		if p.installed {
			var runtime []map[string]string
			for _, d := range slices.Concat(p.deps, p.recorded) {
				runtime = append(runtime, map[string]string{"full_name": d})
			}
			installed = append(installed, map[string]any{"runtime_dependencies": runtime})
		}
		resp["formulae"] = append(resp["formulae"], map[string]any{
			"name": name, "full_name": name, "aliases": p.aliases, "oldnames": []string{},
			"dependencies": p.deps, "recommended_dependencies": []string{}, "uses_from_macos": []any{},
			"installed": installed,
		})
	}
	for _, name := range casks {
		p, ok := f.casks[name]
		if !ok {
			return nil, fmt.Errorf("brew info failed: No available cask with the name %q", name)
		}
		var installed any
		if p.installed {
			installed = "1.0"
		}
		resp["casks"] = append(resp["casks"], map[string]any{
			"token": name, "full_token": name, "old_tokens": []string{},
			"depends_on": map[string]any{"formula": p.deps, "cask": p.caskDeps},
			"installed":  installed,
		})
	}
	return json.Marshal(resp)
}

func (f *fakeBrew) Run(_ context.Context, c proc.Cmd) error {
	cmd := c.String()
	f.runs = append(f.runs, cmd)
	if !slices.Contains(c.Env, upgrade.NoAutoremove) {
		return fmt.Errorf("ran without %s: %s", upgrade.NoAutoremove, cmd)
	}
	if slices.Contains(c.Args, "--ignore-dependencies") {
		return fmt.Errorf("forced removal: %s", cmd)
	}
	if cmd == f.fail {
		return fmt.Errorf("%s failed: exit status 1", cmd)
	}
	if len(c.Args) < 3 {
		return fmt.Errorf("unexpected command: %s", cmd)
	}
	pkgs := f.formulae
	if c.Args[1] == "--cask" {
		pkgs = f.casks
	}
	names := c.Args[2:]
	switch c.Args[0] {
	case "install":
		for _, name := range names {
			if err := f.install(pkgs, name); err != nil {
				return err
			}
		}
		return nil
	case "uninstall":
		return f.uninstall(pkgs, c.Args[1] == "--cask", names)
	}
	return fmt.Errorf("unexpected command: %s", cmd)
}

func (f *fakeBrew) install(pkgs map[string]*fpkg, name string) error {
	p, ok := pkgs[name]
	if !ok {
		return fmt.Errorf("no such package: %s", name)
	}
	p.installed = true
	for _, d := range slices.Concat(p.deps, p.recorded, p.sideEffects) {
		if err := f.install(f.formulae, d); err != nil {
			return err
		}
	}
	for _, d := range p.caskDeps {
		if err := f.install(f.casks, d); err != nil {
			return err
		}
	}
	return nil
}

// uninstall applies brew's guard: no package outside the removal set may
// still need one inside it.
func (f *fakeBrew) uninstall(pkgs map[string]*fpkg, casks bool, names []string) error {
	for _, name := range names {
		if p, ok := pkgs[name]; !ok || !p.installed {
			return fmt.Errorf("%s is not installed", name)
		}
	}
	for _, kind := range []struct {
		pkgs  map[string]*fpkg
		casks bool
	}{{f.formulae, false}, {f.casks, true}} {
		for _, other := range slices.Sorted(maps.Keys(kind.pkgs)) {
			o := kind.pkgs[other]
			if !o.installed || (kind.casks == casks && slices.Contains(names, other)) {
				continue
			}
			needs := slices.Concat(o.deps, o.recorded, o.hidden)
			if casks {
				needs = o.caskDeps
			}
			for _, name := range names {
				if slices.Contains(needs, name) {
					return fmt.Errorf("Refusing to uninstall %s because it is required by %s", name, other)
				}
			}
		}
	}
	for _, name := range names {
		pkgs[name].installed = false
	}
	return nil
}
