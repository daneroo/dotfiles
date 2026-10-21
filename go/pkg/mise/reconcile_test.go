package mise

import (
	"slices"
	"testing"
)

// The outputs are what mise config get printed for bun = "latest" and
// python = ["3.12", "3.11"], in mise 2026.10.15.
func TestParseConfiguredSpecs(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want []string
	}{
		{name: "one spec", out: "latest\n", want: []string{"latest"}},
		{name: "specs in config order", out: `["3.12", "3.11"]` + "\n", want: []string{"3.12", "3.11"}},
		{name: "a comma inside a spec", out: `[">=3.10, <3.12", "3.12"]` + "\n", want: []string{">=3.10, <3.12", "3.12"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseConfiguredSpecs(tt.out)
			if err != nil {
				t.Fatalf("parseConfiguredSpecs(%q) failed: %v", tt.out, err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("parseConfiguredSpecs(%q) = %q, want %q", tt.out, got, tt.want)
			}
		})
	}
}

// Options objects are not supported: aligning one would lose its options.
func TestParseConfiguredSpecsRejectsOptions(t *testing.T) {
	for _, out := range []string{`{ version = "3.12", os = ["linux"] }`, `[{ version = "3.12" }]`, ""} {
		if specs, err := parseConfiguredSpecs(out + "\n"); err == nil {
			t.Errorf("parseConfiguredSpecs(%q) = %q, want an error", out, specs)
		}
	}
}
