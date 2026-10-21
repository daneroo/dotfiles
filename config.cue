package schema

// Schema for package manager configurations.
//
// This schema validates:
// 1. Package Manager Configurations:
//    - Homebrew formulae and casks
//    - mise versioned tools and their version specs
//    - Global NPM packages
//
// 2. Required Sections:
//    All sections are required using CUE's '!' operator:
//      homebrew!: {...}  // Required homebrew section
//      mise!: {...}      // Required mise section
//      npm!: {...}       // Required npm section
//
// 3. Format Validation:
//    - Homebrew formulae: must be either "name" or "org/repo/name"
//    - mise version specs: one spec, or a list whose first is the default.
//      Mise interprets each spec, e.g. "latest", "lts", "3.12", ">=20, <22";
//      options objects are not supported
//    - NPM packages: list of package names

import (
	"list"
	"strings"
)

testValidConfigs: {
	test1Minimal: #Config & {
		homebrew: {}
		mise: {}
		npm: []
	}
	test1Smol: #Config & {
		homebrew: {
			formulae: main: ["go", "git"]
			casks: []
		}
		mise: {
			bun: ["latest"]
			python: ["3.12", "3.11"] // newest 3.12.x is the default
		}
		npm: []
	}
	test1Scalars: #Config & {
		homebrew: {}
		mise: {
			bun:  "latest"
			node: ">=20, <22" // a native expression, interpreted by mise
		}
		npm: []
	}
}
// Main configuration schema
#Config: {
	// Required sections
	homebrew!: {
		// Formulae organized by sections
		formulae: [string]: [...#Formula]
		// Casks 
		casks: [...#Formula]
	}

	// mise versioned tools, each with one version spec or a list of them
	mise!: [string]: #VersionSpec | ([#VersionSpec, ...#VersionSpec] & list.UniqueItems())

	// Global NPM packages
	npm!: [...string]
}

// Helper to get basename of a package (for sorting)
#basename: {
	input: string
	out:   strings.Split(input, "/")[-1]
}

// A version spec is any non-empty string: mise interprets it
#VersionSpec: string & !=""

// Valid formula format (either "name" or "tap/repo/name")
#Formula: string & =~"^([^/]+|[^/]+/[^/]+/[^/]+)$"
