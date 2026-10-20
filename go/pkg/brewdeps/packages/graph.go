package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"slices"
	"strings"

	"github.com/daneroo/dotfiles/go/pkg/brewdeps/types"
	"github.com/daneroo/dotfiles/go/pkg/proc"
)

// graph is the dependency graph of the installed packages and of the desired
// ones, installed or not, with every dependency described.
type graph struct {
	// nodes is keyed by full name: tap-qualified for packages from other taps.
	nodes map[types.Package]*node
	// names maps every name a package is known by, such as its short name,
	// aliases and old names, to its full-name key.
	names map[types.Package]types.Package
}

// node is one formula or cask.
type node struct {
	pkg       types.Package
	installed bool
	// deps are the direct dependencies, as named by Homebrew.
	deps []types.Package
}

// observe builds the graph: first the installed packages, then, until every
// name is described, the desired packages and dependencies Homebrew has not
// yet described. A name Homebrew cannot describe is an error: the graph
// would be incomplete.
func observe(ctx context.Context, x proc.Exec, desired []types.Package) (*graph, error) {
	g := &graph{nodes: map[types.Package]*node{}, names: map[types.Package]types.Package{}}
	out, err := x.Output(ctx, proc.Command("brew", "info", "--json=v2", "--installed"))
	if err != nil {
		return nil, err
	}
	if err := g.add(out); err != nil {
		return nil, err
	}

	for pending := g.undescribed(desired); len(pending) > 0; pending = g.undescribed(desired) {
		for _, kind := range []struct {
			flag   string
			isCask bool
		}{{"--formula", false}, {"--cask", true}} {
			var names []string
			for _, p := range pending {
				if p.IsCask == kind.isCask {
					names = append(names, p.Name)
				}
			}
			if len(names) == 0 {
				continue
			}
			out, err := x.Output(ctx, proc.Command("brew", append([]string{"info", "--json=v2", kind.flag}, names...)...))
			if err != nil {
				return nil, err
			}
			if err := g.add(out); err != nil {
				return nil, err
			}
		}
		for _, p := range pending {
			if _, ok := g.resolve(p); !ok {
				return nil, fmt.Errorf("incomplete dependency graph: brew info did not describe %s", describe(p))
			}
		}
	}
	return g, nil
}

// infoResponse is the part of brew info --json=v2 the graph uses.
type infoResponse struct {
	Formulae []struct {
		Name                    string            `json:"name"`
		FullName                string            `json:"full_name"`
		Aliases                 []string          `json:"aliases"`
		Oldnames                []string          `json:"oldnames"`
		Dependencies            []string          `json:"dependencies"`
		RecommendedDependencies []string          `json:"recommended_dependencies"`
		UsesFromMacos           []json.RawMessage `json:"uses_from_macos"`
		Installed               []struct {
			RuntimeDependencies []struct {
				FullName string `json:"full_name"`
			} `json:"runtime_dependencies"`
		} `json:"installed"`
	} `json:"formulae"`
	Casks []struct {
		Token     string   `json:"token"`
		FullToken string   `json:"full_token"`
		OldTokens []string `json:"old_tokens"`
		DependsOn struct {
			Formula []string `json:"formula"`
			Cask    []string `json:"cask"`
		} `json:"depends_on"`
		Installed *string `json:"installed"`
	} `json:"casks"`
}

// add parses brew info --json=v2 output into the graph. An installed
// formula's dependencies include those recorded when it was installed.
func (g *graph) add(out []byte) error {
	var info infoResponse
	if err := json.Unmarshal(out, &info); err != nil {
		return fmt.Errorf("failed to parse brew info output: %w", err)
	}
	for _, f := range info.Formulae {
		if f.FullName == "" {
			return fmt.Errorf("incomplete brew info entry for formula %q", f.Name)
		}
		n := &node{pkg: types.Package{Name: f.FullName}, installed: len(f.Installed) > 0}
		for _, d := range slices.Concat(f.Dependencies, f.RecommendedDependencies, linuxDependencies(f.UsesFromMacos)) {
			n.deps = append(n.deps, types.Package{Name: d})
		}
		for _, i := range f.Installed {
			for _, d := range i.RuntimeDependencies {
				n.deps = append(n.deps, types.Package{Name: d.FullName})
			}
		}
		g.insert(n, slices.Concat([]string{f.Name}, f.Aliases, f.Oldnames))
	}
	for _, c := range info.Casks {
		if c.FullToken == "" {
			return fmt.Errorf("incomplete brew info entry for cask %q", c.Token)
		}
		n := &node{pkg: types.Package{Name: c.FullToken, IsCask: true}, installed: c.Installed != nil}
		for _, d := range c.DependsOn.Formula {
			n.deps = append(n.deps, types.Package{Name: d})
		}
		for _, d := range c.DependsOn.Cask {
			n.deps = append(n.deps, types.Package{Name: d, IsCask: true})
		}
		g.insert(n, slices.Concat([]string{c.Token}, c.OldTokens))
	}
	return nil
}

// linuxDependencies returns the names of the uses_from_macos dependencies,
// which are real dependencies everywhere but macOS. Each entry is a name or
// an object keyed by the name.
func linuxDependencies(uses []json.RawMessage) []string {
	if runtime.GOOS == "darwin" {
		return nil
	}
	var names []string
	for _, raw := range uses {
		var name string
		if json.Unmarshal(raw, &name) == nil {
			names = append(names, name)
			continue
		}
		var bounded map[string]json.RawMessage
		if json.Unmarshal(raw, &bounded) == nil {
			for name := range bounded {
				names = append(names, name)
			}
		}
	}
	return names
}

// insert adds the node under its full name and its other names. A name
// already taken by a full name keeps it.
func (g *graph) insert(n *node, names []string) {
	if old, ok := g.nodes[n.pkg]; ok {
		old.installed = old.installed || n.installed
		old.deps = append(old.deps, n.deps...)
		n = old
	} else {
		g.nodes[n.pkg] = n
	}
	g.names[n.pkg] = n.pkg
	for _, name := range names {
		alias := types.Package{Name: name, IsCask: n.pkg.IsCask}
		if _, ok := g.nodes[alias]; !ok {
			g.names[alias] = n.pkg
		}
	}
}

// resolve returns the node a name refers to.
func (g *graph) resolve(p types.Package) (*node, bool) {
	key, ok := g.names[p]
	if !ok {
		return nil, false
	}
	return g.nodes[key], true
}

// undescribed returns the desired packages and dependencies the graph has
// no node for, sorted.
func (g *graph) undescribed(desired []types.Package) []types.Package {
	seen := map[types.Package]bool{}
	var missing []types.Package
	check := func(p types.Package) {
		if _, ok := g.resolve(p); !ok && !seen[p] {
			seen[p] = true
			missing = append(missing, p)
		}
	}
	for _, p := range desired {
		check(p)
	}
	for _, n := range g.nodes {
		for _, d := range n.deps {
			check(d)
		}
	}
	sortPackages(missing)
	return missing
}

// protected returns the desired packages and everything they depend on,
// directly or transitively, by full name.
func (g *graph) protected(desired []types.Package) map[types.Package]bool {
	keep := map[types.Package]bool{}
	var visit func(p types.Package)
	visit = func(p types.Package) {
		n, ok := g.resolve(p)
		if !ok || keep[n.pkg] {
			return
		}
		keep[n.pkg] = true
		for _, d := range n.deps {
			visit(d)
		}
	}
	for _, p := range desired {
		visit(p)
	}
	return keep
}

// compare returns the desired packages that aren't installed, as named in
// the desired state, and the installed packages nothing desired needs.
func (g *graph) compare(desired []types.Package) (missing, unwanted []types.Package) {
	for _, p := range desired {
		if n, ok := g.resolve(p); !ok || !n.installed {
			missing = append(missing, p)
		}
	}
	keep := g.protected(desired)
	for key, n := range g.nodes {
		if n.installed && !keep[key] {
			unwanted = append(unwanted, key)
		}
	}
	sortPackages(missing)
	sortPackages(unwanted)
	return missing, unwanted
}

// neededBy reports whether another package in the set depends on p, so p
// would not be removed on its own.
func (g *graph) neededBy(set []types.Package, p types.Package) bool {
	for _, other := range set {
		n, ok := g.resolve(other)
		if !ok || n.pkg == p {
			continue
		}
		for _, d := range n.deps {
			if dn, ok := g.resolve(d); ok && dn.pkg == p {
				return true
			}
		}
	}
	return false
}

// sortPackages sorts formulae before casks, each by name.
func sortPackages(pkgs []types.Package) {
	slices.SortFunc(pkgs, func(a, b types.Package) int {
		if a.IsCask != b.IsCask {
			if a.IsCask {
				return 1
			}
			return -1
		}
		return strings.Compare(a.Name, b.Name)
	})
}

// describe names a package for the report, marking casks.
func describe(p types.Package) string {
	if p.IsCask {
		return p.Name + " (cask)"
	}
	return p.Name
}
