package schema

// Schema for package manager configurations.
//
// This schema validates:
// 1. Package Manager Configurations:
//    - Homebrew formulae and casks
//    - Global NPM packages
//
// 2. Required Sections:
//    All sections are required using CUE's '!' operator:
//      homebrew!: {...}  // Required homebrew section
//      npm!: {...}       // Required npm section
//
// 3. Format Validation:
//    - Homebrew formulae: must be either "name" or "org/repo/name"
//    - NPM packages: list of package names

import (
	"strings"
)

testValidConfigs: {
	test1Minimal: #Config & {
		homebrew: {}
		npm: []
	}
	test1Smol: #Config & {
		homebrew: {
			formulae: main: ["go", "git"]
			casks: []
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

	// Global NPM packages
	npm!: [...string]
}

// Helper to get basename of a package (for sorting)
#basename: {
	input: string
	out:   strings.Split(input, "/")[-1]
}

// Valid formula format (either "name" or "tap/repo/name")
#Formula: string & =~"^([^/]+|[^/]+/[^/]+/[^/]+)$"
