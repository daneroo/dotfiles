package asdf

import (
	"reflect"
	"testing"
)

func TestFilterVersions(t *testing.T) {
	tests := []struct {
		name     string
		versions []string
		prefix   string
		want     []string
	}{
		{
			name:     "python 3.12",
			versions: []string{"3.12-dev", "3.12.0", "3.12.1", "3.12.0-rc1", "3.13.0"},
			prefix:   "3.12",
			want:     []string{"3.12.0", "3.12.1"},
		},
		{
			name:     "major version only",
			versions: []string{"3.0.0", "3.1.0", "4.0.0", "3-dev"},
			prefix:   "3",
			want:     []string{"3.0.0", "3.1.0"},
		},
		{
			name:     "exact version",
			versions: []string{"3.12.0", "3.12.1", "3.12.0-rc1"},
			prefix:   "3.12.0",
			want:     []string{"3.12.0"},
		},
		{
			name:     "no matches",
			versions: []string{"3.11.0", "3.13.0", "3.12-dev"},
			prefix:   "3.12",
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterVersions(tt.versions, tt.prefix)
			if len(got) != len(tt.want) {
				t.Errorf("filterVersions() returned %d items, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("filterVersions()[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSortVersions(t *testing.T) {
	tests := []struct {
		name     string
		versions []string
		want     []string
	}{
		{
			name:     "simple sort",
			versions: []string{"1.2.0", "1.10.0", "1.1.0"},
			want:     []string{"1.1.0", "1.2.0", "1.10.0"},
		},
		{
			name:     "different lengths",
			versions: []string{"1.2", "1.2.1", "1.2.0"},
			want:     []string{"1.2", "1.2.0", "1.2.1"},
		},
		{
			name:     "major version differences",
			versions: []string{"2.0.0", "10.0.0", "1.0.0"},
			want:     []string{"1.0.0", "2.0.0", "10.0.0"},
		},
		{
			name:     "empty list",
			versions: []string{},
			want:     []string{},
		},
		{
			name:     "single version",
			versions: []string{"1.0.0"},
			want:     []string{"1.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortVersions(tt.versions)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sortVersions() = %v, want %v", got, tt.want)
			}
		})
	}
}
