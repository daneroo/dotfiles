package asdf

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// filterAndSortVersions takes a list of version strings and a prefix,
// filters for clean semver versions matching the prefix,
// and returns them sorted in ascending order
func filterAndSortVersions(versions []string, prefix string) []string {
	filtered := filterVersions(versions, prefix)
	return sortVersions(filtered)
}

// filterVersions returns versions matching the prefix pattern X.Y.Z
func filterVersions(versions []string, prefix string) []string {
	pattern := fmt.Sprintf("^%s(\\.\\d+)*$", regexp.QuoteMeta(prefix))
	re := regexp.MustCompile(pattern)

	var matches []string
	for _, v := range versions {
		if re.MatchString(v) {
			matches = append(matches, v)
		}
	}
	return matches
}

// sortVersions sorts version strings in ascending order
func sortVersions(versions []string) []string {
	sorted := make([]string, len(versions))
	copy(sorted, versions)
	sort.Slice(sorted, func(i, j int) bool {
		a := strings.Split(sorted[i], ".")
		b := strings.Split(sorted[j], ".")
		for k := 0; k < len(a) && k < len(b); k++ {
			ai, _ := strconv.Atoi(a[k])
			bi, _ := strconv.Atoi(b[k])
			if ai != bi {
				return ai < bi
			}
		}
		return len(a) < len(b)
	})
	return sorted
}
