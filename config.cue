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
//    - mise version specs, in order (the first is the default), each one of:
//      * "latest": the newest release
//      * A version prefix: "X[.Y[.Z...]]" (e.g., "3", "3.12", "3.12.15")
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

	// mise versioned tools, each with its version specs
	mise!: [string]: [#VersionSpec, ...#VersionSpec] & list.UniqueItems()

	// Global NPM packages
	npm!: [...string]
}

// Helper to get basename of a package (for sorting)
#basename: {
	input: string
	out:   strings.Split(input, "/")[-1]
}

// Valid version spec: latest or a version prefix
#VersionSpec: string & =~"^(latest|\\d+(\\.\\d+)*)$"

// Valid formula format (either "name" or "tap/repo/name")
#Formula: string & =~"^([^/]+|[^/]+/[^/]+/[^/]+)$"
