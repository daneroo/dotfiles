package config

import (
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
asdf: {}
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
asdf:
  nodejs: ["lts"]
npm:
  - typescript`,
		},
		{
			name: "asdf versions can be in any order",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
asdf:
  nodejs: ["20.0.0", "18.0.0", "lts"]
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
asdf: {}
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
asdf: {}
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
asdf: {}
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
asdf: {}
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
asdf: {}
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
asdf: {}
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
asdf: {}
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
asdf: {}
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
asdf: {}
npm: []`,
			msgIncludes: []string{"cannot unmarshal"},
		},
		{
			name: "invalid asdf version format",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
asdf:
  nodejs: ["invalid-version"]
npm: []`,
			msgIncludes: []string{"invalid version format"},
		},
		{
			name: "invalid cask format",
			yaml: `
homebrew:
  formulae:
    main: []
  casks:
    - this/has/way/too/many/parts
asdf: {}
npm: []`,
			msgIncludes: []string{"invalid format"},
		},
		{
			name: "missing required homebrew.casks field",
			yaml: `
homebrew:
  formulae:
    main: []
asdf: {}
npm: []`,
			msgIncludes: []string{"missing required", "homebrew.casks"},
		},
		{
			name: "missing required homebrew.formulae field",
			yaml: `
homebrew:
  casks: []
asdf: {}
npm: []`,
			msgIncludes: []string{"missing required", "homebrew.formulae"},
		},
		{
			name: "missing required asdf section",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
npm: []`,
			msgIncludes: []string{"missing required", "asdf"},
		},
		{
			name: "missing required npm section",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
asdf: {}`,
			msgIncludes: []string{"missing required", "npm"},
		},
		{
			name: "unknown key at root level",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
asdf: {}
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
asdf: {}
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
asdf: {}
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
asdf: {}
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
asdf: {}
npm: [same-package, same-package]`,
			msgIncludes: []string{`duplicate "same-package"`},
		},
		{
			name: "duplicate asdf version",
			yaml: `
homebrew:
  formulae:
    main: []
  casks: []
asdf:
  nodejs: [lts, lts]
npm: []`,
			msgIncludes: []string{`duplicate "lts"`},
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

func TestLoadConfigAcceptsRepoConfig(t *testing.T) {
	if _, err := LoadConfig("../../../config.yaml"); err != nil {
		t.Errorf("the repo's config.yaml is invalid: %v", err)
	}
}
