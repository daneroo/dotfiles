package schema

// Schema for package manager configurations.
//
// This schema validates:
// 1. Package Manager Configurations:
//    - Homebrew formulae and casks
//    - ASDF runtime versions
//    - Global NPM packages
//
// 2. Required Sections:
//    All sections are required using CUE's '!' operator:
//      homebrew!: {...}  // Required homebrew section
//      asdf!: {...}      // Required asdf section
//      npm!: {...}       // Required npm section
//
// 3. Format Validation:
//    - Homebrew formulae: must be either "name" or "org/repo/name"
//    - ASDF versions: must be one of:
//      * "latest": latest stable version
//      * "lts": latest LTS version (nodejs only)
//      * Semantic version: "X[.Y[.Z]]" (e.g., "3", "3.12", "3.12.1")
//    - NPM packages: list of package names

import (
	"strings"
)

testValidConfigs: {
	test1Minimal: #Config & {
		homebrew: {}
		asdf: {}
		npm: []
	}
	test1Smol: #Config & {
		homebrew: {
			formulae: main: ["go", "git"]
			casks: []
		}
		asdf: {
			nodejs: ["lts"] //Latest LTS version
			deno: ["latest"] // Latest stable
			bun: ["latest"] // Latest stable
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

	// ASDF version manager configuration
	asdf!: [string]: #VersionList

	// Global NPM packages
	npm!: [...string]
}

// Helper to get basename of a package (for sorting)
#basename: {
	input: string
	out:   strings.Split(input, "/")[-1]
}

// Version list with format validation
#VersionList: [...#Version]

// Valid version formats
#Version: string & =~"^(latest|lts|\\d+(\\.\\d+){0,2})$"

// Valid formula format (either "name" or "tap/repo/name")
#Formula: string & =~"^([^/]+|[^/]+/[^/]+/[^/]+)$"
