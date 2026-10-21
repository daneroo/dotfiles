package mise_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/daneroo/dotfiles/go/pkg/mise"
	"github.com/daneroo/dotfiles/go/pkg/proc"
	"github.com/daneroo/dotfiles/go/pkg/run"
)

// globalFile is mise's global config, which reconcile sets.
func globalFile() string { return os.Getenv("MISE_GLOBAL_CONFIG_FILE") }

// fakeMise is a controlled mise. Its global config maps each tool to its
// requests; a request resolves to the newest release it names exactly or as
// a prefix, or as given in resolve.
type fakeMise struct {
	missing   bool
	installed map[string][]string
	global    map[string][]string
	// scalar marks the tools whose global value is a string, not a list.
	scalar map[string]bool
	// raw, when set for a tool, is what config get prints for it, such as an
	// options object.
	raw map[string]string
	// foreign maps tools with global requests from another file to it.
	foreign  map[string]string
	releases map[string][]string
	resolve  map[string]string // "tool@request" → version
	fail     string
	// latests counts the resolutions; runs records the changes.
	latests int
	runs    []string
}

func newFake() *fakeMise {
	return &fakeMise{
		installed: map[string][]string{},
		global:    map[string][]string{},
		scalar:    map[string]bool{},
		raw:       map[string]string{},
		foreign:   map[string]string{},
		releases: map[string][]string{
			"bun":    {"1.4.2", "1.4.3"},
			"jq":     {"1.7.1"},
			"node":   {"20.19.1", "21.7.3", "22.3.0"},
			"python": {"3.11.17", "3.12.15"},
		},
		resolve: map[string]string{"node@>=20, <22": "21.7.3"},
	}
}

func (f *fakeMise) LookPath(name string) (string, error) {
	if name != "mise" || f.missing {
		return "", errors.New("not found")
	}
	return "/fake/bin/mise", nil
}

func (f *fakeMise) Output(_ context.Context, c proc.Cmd) ([]byte, error) {
	cmd := c.String()
	if cmd == f.fail {
		return nil, fmt.Errorf("%s failed: exit status 1", cmd)
	}
	type source struct {
		Path string `json:"path"`
	}
	type entry struct {
		Version string `json:"version"`
		Source  source `json:"source"`
	}
	switch {
	case cmd == "mise ls --installed --json":
		out := map[string][]entry{}
		for tool, versions := range f.installed {
			for _, v := range versions {
				out[tool] = append(out[tool], entry{Version: v})
			}
		}
		return json.Marshal(out)
	case cmd == "mise ls --global --json":
		out := map[string][]entry{}
		for tool, requests := range f.global {
			for _, r := range slices.Sorted(slices.Values(requests)) { // mise sorts them
				out[tool] = append(out[tool], entry{Version: r, Source: source{globalFile()}})
			}
		}
		for tool, path := range f.foreign {
			out[tool] = []entry{{Version: "1.0", Source: source{path}}}
		}
		return json.Marshal(out)
	case strings.HasPrefix(cmd, "mise config get --global tools."):
		tool := strings.TrimPrefix(c.Args[3], "tools.")
		if raw, ok := f.raw[tool]; ok {
			return []byte(raw + "\n"), nil
		}
		requests, ok := f.global[tool]
		if !ok {
			return nil, fmt.Errorf("%s failed: exit status 1\nstderr: mise ERROR Key not found: tools.%s in %s", cmd, tool, globalFile())
		}
		if f.scalar[tool] {
			return []byte(requests[0] + "\n"), nil
		}
		out, _ := json.Marshal(requests)
		return append(out, '\n'), nil
	case strings.HasPrefix(cmd, "mise latest "):
		f.latests++
		return []byte(f.latest(c.Args[1]) + "\n"), nil
	}
	return nil, fmt.Errorf("unexpected command: %s", cmd)
}

// latest resolves tool or tool@request; no match prints nothing.
func (f *fakeMise) latest(arg string) string {
	tool, request, _ := strings.Cut(arg, "@")
	if v, ok := f.resolve[arg]; ok {
		return v
	}
	newest := ""
	for _, v := range f.releases[tool] {
		if request == "" || v == request || strings.HasPrefix(v, request+".") {
			newest = v
		}
	}
	return newest
}

func (f *fakeMise) Run(_ context.Context, c proc.Cmd) error {
	cmd := c.String()
	f.runs = append(f.runs, cmd)
	if cmd == f.fail {
		return fmt.Errorf("%s failed: exit status 1", cmd)
	}
	a := c.Args
	if len(a) > 2 && a[0] == "config" && a[1] == "set" {
		// mise reports a missing global config as not trusted
		if _, err := os.Stat(os.Getenv("MISE_GLOBAL_CONFIG_FILE")); err != nil {
			return fmt.Errorf("%s failed: Config files in %s are not trusted", cmd, os.Getenv("MISE_GLOBAL_CONFIG_FILE"))
		}
	}
	switch {
	case len(a) == 2 && a[0] == "install":
		tool, version, _ := strings.Cut(a[1], "@")
		if !slices.Contains(f.releases[tool], version) {
			return fmt.Errorf("no release %s", a[1])
		}
		if !slices.Contains(f.installed[tool], version) {
			f.installed[tool] = append(f.installed[tool], version)
		}
		return nil
	case len(a) == 2 && a[0] == "uninstall":
		tool, version, _ := strings.Cut(a[1], "@")
		f.installed[tool] = slices.DeleteFunc(f.installed[tool], func(v string) bool { return v == version })
		if len(f.installed[tool]) == 0 {
			delete(f.installed, tool)
		}
		return nil
	case len(a) == 7 && slices.Equal(a[:5], []string{"config", "set", "--global", "--type", "string"}):
		tool := strings.TrimPrefix(a[5], "tools.")
		f.global[tool] = []string{a[6]}
		f.scalar[tool] = true
		delete(f.raw, tool)
		return nil
	case len(a) == 8 && slices.Equal(a[:6], []string{"config", "set", "--global", "--append", "--type", "string"}):
		tool := strings.TrimPrefix(a[6], "tools.")
		if !slices.Contains(f.global[tool], a[7]) {
			f.global[tool] = append(f.global[tool], a[7])
		}
		f.scalar[tool] = false
		return nil
	case len(a) == 4 && slices.Equal(a[:3], []string{"unuse", "--global", "--no-prune"}):
		delete(f.global, a[3])
		return nil
	}
	return fmt.Errorf("unexpected command: %s", cmd)
}

var desired = map[string][]string{"bun": {"latest"}, "python": {"3.12", "3.11"}}

// reconcile runs the reconciler. Mise's global config is an existing
// temporary file, unless the test chose a temporary path of its own.
func reconcile(t *testing.T, r run.Runner, f *fakeMise, want map[string][]string) (run.Result, string) {
	t.Helper()
	if !strings.HasPrefix(os.Getenv("MISE_GLOBAL_CONFIG_FILE"), os.TempDir()) {
		path := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(path, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("MISE_GLOBAL_CONFIG_FILE", path)
	}
	var out strings.Builder
	r.Out = &out
	res := r.Run(context.Background(), []run.Reconciler{mise.New(want, f)})
	return res, out.String()
}

// A fresh machine gets the resolved versions installed and its global
// defaults set in config order; a rerun is unchanged.
func TestFreshMachineThenUnchanged(t *testing.T) {
	f := newFake()
	res, out := reconcile(t, run.Runner{Mode: run.Yolo}, f, desired)
	want := `
## Mise Section

✓ - mise is installed

✗ - bun latest → 1.4.3 is not installed
✗ - bun defaults are (none), config.yaml wants latest
✗ - python 3.12 → 3.12.15 is not installed
✗ - python 3.11 → 3.11.17 is not installed
✗ - python defaults are (none), config.yaml wants 3.12, 3.11

✗ - Missing mise versions: (3)
 mise install bun@1.4.3
 mise install python@3.12.15
 mise install python@3.11.17
✗ - Mise defaults to change: (2)
 mise config set --global --type string tools.bun latest
 mise config set --global --type string tools.python 3.12
 mise config set --global --append --type string tools.python 3.11
✓ - No unwanted mise versions
✓ - No strays in mise's global config
✓ - Mise applied
`
	if out != want {
		t.Errorf("output =\n%s\nwant\n%s", out, want)
	}
	if res.ExitCode() != 0 {
		t.Fatalf("exit code %d: %v", res.ExitCode(), res.Err)
	}
	if got := f.global["python"]; !slices.Equal(got, []string{"3.12", "3.11"}) || !f.scalar["bun"] {
		t.Errorf("global python = %q, bun scalar %v; want [3.12 3.11] and a scalar", got, f.scalar["bun"])
	}

	f.runs = nil
	res, out = reconcile(t, run.Runner{Mode: run.Yolo}, f, desired)
	if len(f.runs) != 0 || res.Entries[0].Status != run.Unchanged {
		t.Errorf("rerun ran %q with status %v:\n%s", f.runs, res.Entries[0].Status, out)
	}
}

// An environment variable overriding a desired tool's version hides it from
// mise's global view, so planning stops rather than compare the wrong thing.
func TestEnvironmentOverrideIsAnError(t *testing.T) {
	t.Setenv("MISE_BUN_VERSION", "1.4.2")
	f := newFake()
	f.installed["bun"] = []string{"1.4.2", "1.4.3"}
	res, out := reconcile(t, run.Runner{Mode: run.Yolo}, f, desired)
	if res.ExitCode() != 1 || !strings.Contains(out, "MISE_BUN_VERSION=1.4.2 overrides bun's global versions; unset it") || len(f.runs) != 0 {
		t.Errorf("exit code %d, ran %q, output:\n%s", res.ExitCode(), f.runs, out)
	}
}

// Mise cannot set a default in a missing global config, so the plan creates
// it, empty, first.
func TestMissingGlobalConfigIsCreated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "mise", "config.toml")
	t.Setenv("MISE_GLOBAL_CONFIG_FILE", path)
	f := newFake()
	res, out := reconcile(t, run.Runner{Mode: run.Yolo}, f, map[string][]string{"bun": {"latest"}})
	want := "✗ - Mise defaults to change: (1)\n create empty " + path + "\n mise config set --global --type string tools.bun latest\n"
	if !strings.Contains(out, want) {
		t.Errorf("output lacks %q:\n%s", want, out)
	}
	if res.ExitCode() != 0 {
		t.Fatalf("exit code %d: %v", res.ExitCode(), res.Err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("global config not created: %v", err)
	}
	if got := f.global["bun"]; !slices.Equal(got, []string{"latest"}) {
		t.Errorf("global bun = %q, want [latest]", got)
	}

	// A dry run creates nothing
	path = filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("MISE_GLOBAL_CONFIG_FILE", path)
	_, out = reconcile(t, run.Runner{Mode: run.DryRun}, newFake(), map[string][]string{"bun": {"latest"}})
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) || !strings.Contains(out, " create empty "+path+"\n") {
		t.Errorf("dry run created %s (%v), or did not preview it:\n%s", path, err, out)
	}
}

// A singleton list in config.yaml matches mise's scalar, and a scalar
// matches a one-element list.
func TestSingletonFormsAreEquivalent(t *testing.T) {
	for _, scalar := range []bool{true, false} {
		f := newFake()
		f.installed = map[string][]string{"bun": {"1.4.3"}}
		f.global = map[string][]string{"bun": {"latest"}}
		f.scalar["bun"] = scalar
		res, out := reconcile(t, run.Runner{Mode: run.Yolo}, f, map[string][]string{"bun": {"latest"}})
		if res.Entries[0].Status != run.Unchanged {
			t.Errorf("scalar %v: status %v:\n%s", scalar, res.Entries[0].Status, out)
		}
	}
}

// A changed default order is fixed by editing the line, in config order.
func TestChangedDefaultOrder(t *testing.T) {
	f := newFake()
	f.installed = map[string][]string{"python": {"3.11.17", "3.12.15"}}
	f.global = map[string][]string{"python": {"3.11", "3.12"}}
	_, out := reconcile(t, run.Runner{Mode: run.Yolo}, f, map[string][]string{"python": {"3.12", "3.11"}})
	if !strings.Contains(out, "✗ - python defaults are 3.11, 3.12, config.yaml wants 3.12, 3.11") {
		t.Errorf("output:\n%s", out)
	}
	if got := f.global["python"]; !slices.Equal(got, []string{"3.12", "3.11"}) {
		t.Errorf("global python = %q, want [3.12 3.11]", got)
	}
}

// Installs and default changes come first; then unwanted versions are
// uninstalled and strays unused without pruning.
func TestCleanupOrder(t *testing.T) {
	f := newFake()
	f.installed = map[string][]string{"bun": {"1.4.2"}, "jq": {"1.7.1"}, "python": {"3.11.17", "3.12.15"}}
	f.global = map[string][]string{"bun": {"latest"}, "jq": {"1.7.1"}, "python": {"3.12", "3.11"}}
	f.scalar = map[string]bool{"bun": true, "jq": true}
	res, out := reconcile(t, run.Runner{Mode: run.Yolo}, f, desired)
	wantRuns := []string{
		"mise install bun@1.4.3",
		"mise uninstall bun@1.4.2",
		"mise uninstall jq@1.7.1",
		"mise unuse --global --no-prune jq",
	}
	if !slices.Equal(f.runs, wantRuns) {
		t.Errorf("ran %q, want %q", f.runs, wantRuns)
	}
	if !strings.Contains(out, "✗ - jq is a stray in mise's global config, "+globalFile()+": config.yaml doesn't list it") {
		t.Errorf("output:\n%s", out)
	}
	if res.ExitCode() != 0 || len(f.installed["jq"]) != 0 || f.global["jq"] != nil {
		t.Errorf("exit code %d, jq installed %q, jq global %q", res.ExitCode(), f.installed["jq"], f.global["jq"])
	}
}

// Mise interprets native expressions; their commas stay inside them.
func TestNativeExpressionWithComma(t *testing.T) {
	f := newFake()
	want := map[string][]string{"node": {">=20, <22", "22"}}
	_, out := reconcile(t, run.Runner{Mode: run.Yolo}, f, want)
	if !strings.Contains(out, " mise config set --global --type string tools.node '>=20, <22'\n") {
		t.Errorf("output:\n%s", out)
	}
	if got := f.global["node"]; !slices.Equal(got, []string{">=20, <22", "22"}) {
		t.Errorf("global node = %q, want the two requests intact", got)
	}
	if got := f.installed["node"]; !slices.Equal(got, []string{"21.7.3", "22.3.0"}) {
		t.Errorf("installed node = %q", got)
	}
}

// answerAfter answers yes to the prompt, after a release appears.
type answerAfter struct {
	f    *fakeMise
	done bool
}

func (a *answerAfter) Read(p []byte) (int, error) {
	if a.done {
		return 0, errors.New("read twice")
	}
	a.done = true
	a.f.releases["python"] = append(a.f.releases["python"], "3.12.16")
	return copy(p, "y\n"), nil
}

// A release appearing between planning and applying changes nothing: the
// approved concrete versions are installed, with no second resolution.
func TestReleaseBetweenPlanAndApply(t *testing.T) {
	f := newFake()
	r := run.Runner{Mode: run.Confirm, Interactive: true, In: &answerAfter{f: f}}
	_, out := reconcile(t, r, f, map[string][]string{"python": {"3.12"}})
	if f.latests != 1 {
		t.Errorf("resolved %d times, want once, when planning", f.latests)
	}
	if !slices.Contains(f.runs, "mise install python@3.12.15") || slices.Contains(f.runs, "mise install python@3.12.16") {
		t.Errorf("ran %q, want only the approved 3.12.15:\n%s", f.runs, out)
	}
}

func TestFailuresStopLaterSteps(t *testing.T) {
	tests := []struct {
		fail     string
		wantRuns int
	}{
		{fail: "mise install python@3.12.15", wantRuns: 2},
		{fail: "mise config set --global --type string tools.python 3.12", wantRuns: 5},
	}
	for _, tt := range tests {
		t.Run(tt.fail, func(t *testing.T) {
			f := newFake()
			f.installed = map[string][]string{"bun": {"1.4.2"}}
			f.global = map[string][]string{"jq": {"1.7.1"}}
			f.fail = tt.fail
			res, _ := reconcile(t, run.Runner{Mode: run.Yolo}, f, desired)
			if len(f.runs) != tt.wantRuns || f.runs[len(f.runs)-1] != tt.fail {
				t.Errorf("ran %q, want to stop at %s", f.runs, tt.fail)
			}
			for _, r := range f.runs {
				if strings.HasPrefix(r, "mise uninstall") || strings.HasPrefix(r, "mise unuse") {
					t.Errorf("cleaned up after a failure: %s", r)
				}
			}
			if res.ExitCode() != 1 {
				t.Errorf("exit code %d, want 1", res.ExitCode())
			}
		})
	}
}

// What reconfig cannot faithfully plan is an error, and nothing is written.
func TestPlanningErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f *fakeMise)
		want  map[string][]string
		msg   string
	}{
		{
			name:  "a spec mise resolves to nothing",
			setup: func(f *fakeMise) {},
			want:  map[string][]string{"python": {"9.9"}},
			msg:   `version spec "9.9" for python matches no release`,
		},
		{
			name: "an options object in mise's global config",
			setup: func(f *fakeMise) {
				f.global["python"] = []string{"3.12"}
				f.raw["python"] = `{ version = "3.12", os = ["linux"] }`
			},
			want: desired,
			msg:  "unsupported value",
		},
		{
			name:  "global versions from another file",
			setup: func(f *fakeMise) { f.foreign["node"] = "/home/me/.tool-versions" },
			want:  desired,
			msg:   "node's global versions come from /home/me/.tool-versions, not mise's global config",
		},
		{
			name: "a desired tool's versions from another file too",
			setup: func(f *fakeMise) {
				f.global["bun"] = []string{"latest"}
				f.foreign["bun"] = "/home/me/.config/mise/conf.d/bun.toml"
			},
			want: desired,
			msg:  "bun's global versions come from /home/me/.config/mise/conf.d/bun.toml, not mise's global config",
		},
		{
			name:  "mise ls fails",
			setup: func(f *fakeMise) { f.fail = "mise ls --installed --json" },
			want:  desired,
			msg:   "mise ls --installed --json failed",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFake()
			tt.setup(f)
			res, out := reconcile(t, run.Runner{Mode: run.Yolo}, f, tt.want)
			if res.ExitCode() != 1 || !strings.Contains(out, tt.msg) || len(f.runs) != 0 {
				t.Errorf("exit code %d, ran %q, output lacks %q:\n%s", res.ExitCode(), f.runs, tt.msg, out)
			}
		})
	}
}

func TestDryRunChangesNothing(t *testing.T) {
	f := newFake()
	f.global = map[string][]string{"jq": {"1.7.1"}}
	before := maps.Clone(f.global)
	res, _ := reconcile(t, run.Runner{Mode: run.DryRun}, f, desired)
	if len(f.runs) != 0 || !maps.EqualFunc(f.global, before, slices.Equal[[]string]) || res.Entries[0].Outcome != run.Previewed {
		t.Errorf("dry run ran %q, global %q, outcome %v", f.runs, f.global, res.Entries[0].Outcome)
	}
}

func TestMissingMiseIsUnavailable(t *testing.T) {
	f := newFake()
	f.missing = true
	res, out := reconcile(t, run.Runner{Mode: run.Yolo}, f, desired)
	if res.Entries[0].Status != run.Unavailable || res.ExitCode() != 0 || !strings.Contains(out, "✗ - mise is not installed, skipping") {
		t.Errorf("status %v, exit code %d:\n%s", res.Entries[0].Status, res.ExitCode(), out)
	}
}
