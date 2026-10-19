package config

import (
	"slices"
	"strings"
	"testing"
)

func TestParseConfigAccepts(t *testing.T) {
	tests := []struct {
		name string
		yaml string
	}{
		{
			name: "minimal config",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
mise: {}
npm: []`,
		},
		{
			name: "basic sorted config",
			yaml: `
homebrew:
  formulae:
    main:
      - git
      - go
  casks:
    - 1password
mise: {}
npm:
  - typescript`,
		},
		{
			name: "mise version specs can be in any order",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
mise:
  python: ["3.12", "3.11"]
  bun: ["latest"]
  deno: ["2.9.7"]
npm: []`,
		},
		{
			name: "lts is a mise version spec",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
mise:
  node: ["lts"]
npm: []`,
		},
		{
			name: "tap formula sorted by basename",
			yaml: `
homebrew:
  formulae:
    main:
      - git
      - westpoint-io/lazyrsync/lazyrsync
      - wget
  casks: []
mise: {}
npm: []`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseConfig([]byte(tt.yaml)); err != nil {
				t.Errorf("ParseConfig() rejected a valid config: %v", err)
			}
		})
	}
}

func TestParseConfigRejects(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		msgIncludes []string
	}{
		{
			name: "unsorted formulae",
			yaml: `
homebrew:
  formulae:
    main:
      - zsh
      - git
      - bash
  casks: []
mise: {}
npm: []`,
			msgIncludes: []string{`"git" should come before "zsh"`, `"bash" should come before "git"`},
		},
		{
			name: "unsorted casks",
			yaml: `
homebrew:
  formulae:
    main: []
  casks:
    - zoom
    - alfred
    - 1password
mise: {}
npm: []`,
			msgIncludes: []string{"Casks are not sorted"},
		},
		{
			name: "unsorted npm packages",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
mise: {}
npm:
  - zod
  - typescript
  - deno`,
			msgIncludes: []string{"NPM packages are not sorted"},
		},
		{
			name: "unsorted by basename",
			yaml: `
homebrew:
  formulae:
    main:
      - bbb
      - org/repo/aaa
  casks: []
mise: {}
npm: []`,
			msgIncludes: []string{`"org/repo/aaa" should come before "bbb"`},
		},
		{
			name: "same basename sorted by full name",
			yaml: `
homebrew:
  formulae:
    main:
      - org2/repo2/name
      - org1/repo1/name
  casks: []
mise: {}
npm: []`,
			msgIncludes: []string{`"org1/repo1/name" should come before "org2/repo2/name"`},
		},
		{
			name: "unsorted casks by basename",
			yaml: `
homebrew:
  formulae:
    main: []
  casks:
    - bbb
    - org/repo/aaa
mise: {}
npm: []`,
			msgIncludes: []string{`"org/repo/aaa" should come before "bbb"`},
		},
		{
			name: "casks with same basename sorted by full name",
			yaml: `
homebrew:
  formulae:
    main: []
  casks:
    - org2/repo2/name
    - org1/repo1/name
mise: {}
npm: []`,
			msgIncludes: []string{`"org1/repo1/name" should come before "org2/repo2/name"`},
		},
		{
			name: "formulae value should be array not string",
			yaml: `
homebrew:
  formulae:
    main: "not-an-array"
  casks: []
mise: {}
npm: []`,
			msgIncludes: []string{"cannot unmarshal"},
		},
		{
			name: "invalid cask format",
			yaml: `
homebrew:
  formulae:
    main: []
  casks:
    - this/has/way/too/many/parts
mise: {}
npm: []`,
			msgIncludes: []string{"invalid format"},
		},
		{
			name: "missing required homebrew.casks field",
			yaml: `
homebrew:
  formulae:
    main: []
mise: {}
npm: []`,
			msgIncludes: []string{"missing required", "homebrew.casks"},
		},
		{
			name: "missing required homebrew.formulae field",
			yaml: `
homebrew:
  casks: []
mise: {}
npm: []`,
			msgIncludes: []string{"missing required", "homebrew.formulae"},
		},
		{
			name: "asdf section is an unknown key",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
asdf: {}
mise: {}
npm: []`,
			msgIncludes: []string{"field asdf not found"},
		},
		{
			name: "missing required mise section",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
npm: []`,
			msgIncludes: []string{"missing required", "mise"},
		},
		{
			name: "invalid mise version spec",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
mise:
  python: ["3.12.x"]
npm: []`,
			msgIncludes: []string{`Tool "python"`, `invalid version spec "3.12.x"`},
		},
		{
			name: "mise tool with no version specs",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
mise:
  bun: []
npm: []`,
			msgIncludes: []string{`Tool "bun": no version specs`},
		},
		{
			name: "duplicate mise version spec",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
mise:
  python: ["3.12", "3.12"]
npm: []`,
			msgIncludes: []string{`duplicate "3.12"`},
		},
		{
			name: "missing required npm section",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
mise: {}`,
			msgIncludes: []string{"missing required", "npm"},
		},
		{
			name: "unknown key at root level",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
mise: {}
npm: []
homebew: {}`,
			msgIncludes: []string{"homebew"},
		},
		{
			name: "unknown key in homebrew section",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
  cask: []
mise: {}
npm: []`,
			msgIncludes: []string{"cask"},
		},
		{
			name: "duplicate formula",
			yaml: `
homebrew:
  formulae:
    main: [same-package, same-package]
  casks: []
mise: {}
npm: []`,
			msgIncludes: []string{`duplicate "same-package"`},
		},
		{
			name: "duplicate cask",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: [same-package, same-package]
mise: {}
npm: []`,
			msgIncludes: []string{`duplicate "same-package"`},
		},
		{
			name: "duplicate npm package",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
mise: {}
npm: [same-package, same-package]`,
			msgIncludes: []string{`duplicate "same-package"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseConfig([]byte(tt.yaml))
			if err == nil {
				t.Fatal("ParseConfig() accepted an invalid config")
			}
			for _, want := range tt.msgIncludes {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not mention %q", err, want)
				}
			}
		})
	}
}

func TestParseConfigKeepsMiseSpecOrder(t *testing.T) {
	cfg, err := ParseConfig([]byte(`
homebrew:
  formulae:
    main: []
  casks: []
mise:
  python: ["3.12", "3.11"]
npm: []`))
	if err != nil {
		t.Fatalf("ParseConfig() rejected a valid config: %v", err)
	}
	want := []string{"3.12", "3.11"}
	if got := cfg.Mise["python"]; !slices.Equal(got, want) {
		t.Errorf("mise python specs = %v, want %v", got, want)
	}
}

func TestLoadConfigAcceptsRepoConfig(t *testing.T) {
	if _, err := LoadConfig("../../../config.yaml"); err != nil {
		t.Errorf("the repo's config.yaml is invalid: %v", err)
	}
}

func TestLoadConfigAcceptsE2EConfig(t *testing.T) {
	if _, err := LoadConfig("../../../e2e/config.yaml"); err != nil {
		t.Errorf("e2e/config.yaml is invalid: %v", err)
	}
}
