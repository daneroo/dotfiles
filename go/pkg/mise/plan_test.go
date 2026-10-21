package mise

import (
	"slices"
	"testing"
)

func TestPlan(t *testing.T) {
	python := map[string][]string{"python": {"3.12", "3.11"}}
	pythonResolved := map[string]map[string]string{
		"python": {"3.12": "3.12.15", "3.11": "3.11.17"},
	}

	tests := []struct {
		name          string
		desired       map[string][]string
		actual        observed
		wantInstall   []string
		wantUse       []string
		wantUninstall []string
		wantUnuse     []string
	}{
		{
			name:    "a fresh machine",
			desired: map[string][]string{"bun": {"latest"}},
			actual: observed{
				resolved: map[string]map[string]string{"bun": {"latest": "1.4.3"}},
			},
			wantInstall: []string{"mise install bun@1.4.3"},
			wantUse:     []string{"mise use -g bun@latest"},
		},
		{
			name:    "actual state matches desired state",
			desired: python,
			actual: observed{
				installed: map[string][]string{"python": {"3.11.17", "3.12.15"}},
				defaults:  map[string][]string{"python": {"3.12", "3.11"}},
				resolved:  pythonResolved,
			},
		},
		{
			name:    "a newer patch release available",
			desired: python,
			actual: observed{
				installed: map[string][]string{"python": {"3.11.17", "3.12.15"}},
				defaults:  map[string][]string{"python": {"3.12", "3.11"}},
				resolved: map[string]map[string]string{
					"python": {"3.12": "3.12.16", "3.11": "3.11.17"},
				},
			},
			wantInstall:   []string{"mise install python@3.12.16"},
			wantUninstall: []string{"mise uninstall python@3.12.15"},
		},
		{
			name:    "an old patch left after an upgrade",
			desired: python,
			actual: observed{
				installed: map[string][]string{"python": {"3.11.17", "3.12.15", "3.12.16"}},
				defaults:  map[string][]string{"python": {"3.12", "3.11"}},
				resolved: map[string]map[string]string{
					"python": {"3.12": "3.12.16", "3.11": "3.11.17"},
				},
			},
			wantUninstall: []string{"mise uninstall python@3.12.15"},
		},
		{
			name:    "two Python versions",
			desired: python,
			actual: observed{
				resolved: pythonResolved,
			},
			wantInstall: []string{"mise install python@3.12.15", "mise install python@3.11.17"},
			wantUse:     []string{"mise use -g python@3.12 python@3.11"},
		},
		{
			name:    "the default list in the wrong order",
			desired: python,
			actual: observed{
				installed: map[string][]string{"python": {"3.11.17", "3.12.15"}},
				defaults:  map[string][]string{"python": {"3.11", "3.12"}},
				resolved:  pythonResolved,
			},
			wantUse: []string{"mise use -g python@3.12 python@3.11"},
		},
		{
			name:    "a tool installed but not in the desired state",
			desired: python,
			actual: observed{
				installed: map[string][]string{
					"deno":   {"2.9.7"},
					"python": {"3.11.17", "3.12.15"},
				},
				defaults: map[string][]string{"python": {"3.12", "3.11"}},
				resolved: pythonResolved,
			},
			wantUninstall: []string{"mise uninstall deno@2.9.7"},
		},
		{
			name:    "latest",
			desired: map[string][]string{"bun": {"latest"}},
			actual: observed{
				installed: map[string][]string{"bun": {"1.4.2"}},
				defaults:  map[string][]string{"bun": {"latest"}},
				resolved:  map[string]map[string]string{"bun": {"latest": "1.4.3"}},
			},
			wantInstall:   []string{"mise install bun@1.4.3"},
			wantUninstall: []string{"mise uninstall bun@1.4.2"},
		},
		{
			name:    "a stray in mise's global config",
			desired: python,
			actual: observed{
				installed: map[string][]string{
					"bun":    {"1.4.3"},
					"python": {"3.11.17", "3.12.15"},
				},
				defaults: map[string][]string{
					"bun":    {"latest"},
					"python": {"3.12", "3.11"},
				},
				resolved: pythonResolved,
			},
			wantUninstall: []string{"mise uninstall bun@1.4.3"},
			wantUnuse:     []string{"mise unuse -g bun"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			install, use, uninstall, unuse := printedCommands(plan(tt.desired, tt.actual))
			assertCommands(t, "install", install, tt.wantInstall)
			assertCommands(t, "use", use, tt.wantUse)
			assertCommands(t, "uninstall", uninstall, tt.wantUninstall)
			assertCommands(t, "unuse", unuse, tt.wantUnuse)
		})
	}
}

func TestPlanReportsEachSpec(t *testing.T) {
	got := plan(map[string][]string{"python": {"3.12", "3.11"}}, observed{
		installed: map[string][]string{"python": {"3.12.15"}},
		defaults:  map[string][]string{"python": {"3.12", "3.11"}},
		resolved: map[string]map[string]string{
			"python": {"3.12": "3.12.15", "3.11": "3.11.17"},
		},
	})
	want := []string{
		"✓ - python 3.12 → 3.12.15",
		"✗ - python 3.11 → 3.11.17 is not installed",
		"✓ - python defaults match config.yaml: 3.12, 3.11",
	}
	if lines := reportLines(got); !slices.Equal(lines, want) {
		t.Errorf("report:\n got %q\nwant %q", lines, want)
	}
}

func TestPlanReportsAStray(t *testing.T) {
	got := plan(map[string][]string{}, observed{
		defaults:     map[string][]string{"bun": {"latest"}},
		globalConfig: "/Users/daniel/.config/mise/config.toml",
	})
	want := []string{
		"✗ - bun is a stray in mise's global config, /Users/daniel/.config/mise/config.toml: config.yaml doesn't list it",
	}
	if lines := reportLines(got); !slices.Equal(lines, want) {
		t.Errorf("report:\n got %q\nwant %q", lines, want)
	}
}

// reportLines renders the plan's report lines as Reconcile prints them.
func reportLines(p planned) []string {
	var lines []string
	for _, n := range p.report {
		lines = append(lines, string(n.Mark)+" - "+n.Text)
	}
	return lines
}

func assertCommands(t *testing.T, kind string, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("%s commands = %q, want %q", kind, got, want)
	}
}
