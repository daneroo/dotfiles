package main

// 1-Compares requested dependancies: `brewDeps`
// to make sure thay are all installed
// 2- makes sure any other installed casks are dependants of the requested ones.

import (
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"
)

var verbose = true

func main() {
	required := getRequired()
	deps := getDeps()
	installed := getInstalled()

	fmt.Printf("--- Checks: --- ( ✓ / ✗ ) \n")

	ok := sanity(installed, deps)
	if ok {
		fmt.Printf("✓ - Sanity passed: installed < keys(deps)\n")
	} else {
		fmt.Printf("✗ -Sanity failed: installed > keys(deps)\n")
	}

	missing := checkMissing(required, installed)
	if len(missing) > 0 {
		fmt.Printf("✗ -Missing casks:\n")
		for _, cask := range missing {
			fmt.Printf(" brew install %s\n", cask)
		}
		fmt.Printf("\n")
	} else {
		fmt.Printf("✓ - No missing casks\n")
	}

	// Check if all installed are either required, or a dpendant of a required package
	extra := extraneous(required, installed, deps)
	if len(extra) > 0 {
		fmt.Printf("✗ -Extraneous casks: (brew uninstall or brew rmtree)\n")
		for _, e := range extra {
			fmt.Printf(" brew uninstall %s\n", e)
		}
		fmt.Printf("\n")
	} else {
		fmt.Printf("✓ - No extraneous casks\n")
	}
	fmt.Printf("---\n")

}

func checkMissing(required, installed []string) []string {
	missing := []string{}
	for _, req := range required {
		if !contains(installed, req) {
			missing = append(missing, req)
			// fmt.Printf(" - required cask: %s is missing\n", req)
		} else {
			// fmt.Printf(" - required cask: %s is installed\n", req)
		}
	}
	return missing
}

func extraneous(required, installed []string, deps map[string][]string) []string {
	extra := []string{}
	for _, inst := range installed {
		ok := false
		if contains(required, inst) {
			ok = true
			// fmt.Printf(" - %s is required\n", inst)
		} else {
			// if cask is required, then it's deps are OK
			for cask, deps := range deps {
				if contains(required, cask) {
					if contains(deps, inst) {
						ok = true
						// fmt.Printf(" - %s is required transitively by %s\n", inst, cask)
					}
				}
			}
		}
		if !ok {
			extra = append(extra, inst)
			fmt.Printf(" - %s is not required (transitevely)\n", inst)
		}
	}
	return extra
}

func sanity(installed []string, deps map[string][]string) bool {
	// Sanity: make sure all installed appear as a key in deps
	insane := false
	for _, inst := range installed {
		_, ok := deps[inst]
		if !ok {
			insane = true
			fmt.Printf("(In)Sanity: Installed package %s not present in dependancies\n", inst)
		}
	}
	return !insane
}

func getRequired() []string {
	out, err := ioutil.ReadFile("brewDeps")
	if err != nil {
		log.Fatal(err)
	}
	// remove emtpy lines, and lines starting with # (comment)
	required := filter(
		strings.Split(string(out), "\n"),
		func(s string) bool {
			return len(s) > 0 && !strings.HasPrefix(strings.TrimSpace(s), "#")
		})

	// Trim entries
	for i := 0; i < len(required); i++ {
		required[i] = strings.TrimSpace(required[i])
	}

	if verbose {
		fmt.Printf("Required: (./brewDeps)\n %v\n\n", required)
	}
	return required
}

func getDeps() map[string][]string {
	// Parse the output of: brew deps --installed
	// asciinema: gdbm openssl python readline sqlite xz
	// aws-iam-authenticator:
	out, err := exec.Command("brew", "deps", "--installed").Output()
	if err != nil {
		log.Fatal(err)
	}

	// split by line, remove empty lines
	installedColonDeps := spliyByLineNoEmpty(string(out))

	deps := map[string][]string{}

	for _, line := range installedColonDeps {
		ss := strings.SplitN(line, ":", 2)
		if len(ss) != 2 {
			log.Fatalf("Cannot split(:) %q \n", line)
		}
		c := ss[0]
		ds := strings.Fields(ss[1])
		deps[c] = ds
	}

	if verbose {
		fmt.Printf("Deps: (brew deps --installed)\n %v\n\n", deps)
	}
	return deps
}

func getInstalled() []string {
	out, err := exec.Command("brew", "ls", "--full-name").Output()
	// out, err := exec.Command("brew", "ls").Output()
	if err != nil {
		log.Fatal(err)
	}

	// split by line, remove empty lines
	installed := spliyByLineNoEmpty(string(out))
	if verbose {
		fmt.Printf("Installed: (brew ls --full-name)\n %v\n\n", installed)
	}
	return installed
}

func spliyByLineNoEmpty(s string) []string {
	return filter(
		strings.Split(s, "\n"),
		nonEmptyString,
	)
}

func nonEmptyString(s string) bool {
	return len(s) > 0
}

func filter(vs []string, f func(string) bool) []string {
	vsf := make([]string, 0)
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
