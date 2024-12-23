package reconcile

import (
	"testing"

	"github.com/daneroo/dotfiles/go/pkg/brewdeps/types"
)

func TestShowCommands(t *testing.T) {
	tests := []struct {
		name     string
		pkgs     []types.Package
		action   string
		opts     commandOptions
		expected []string
	}{
		{
			name: "single formula",
			pkgs: []types.Package{
				{Name: "wget", IsCask: false},
			},
			action: "install",
			opts:   commandOptions{isCask: false, groupCommand: false},
			expected: []string{
				" brew install --formula wget",
			},
		},
		{
			name: "multiple formulas grouped",
			pkgs: []types.Package{
				{Name: "wget", IsCask: false},
				{Name: "git", IsCask: false},
			},
			action: "install",
			opts:   commandOptions{isCask: false, groupCommand: true},
			expected: []string{
				" brew install --formula wget git",
			},
		},
		{
			name: "mixed formulas and casks - show only casks",
			pkgs: []types.Package{
				{Name: "wget", IsCask: false},
				{Name: "vlc", IsCask: true},
			},
			action: "install",
			opts:   commandOptions{isCask: true, groupCommand: false},
			expected: []string{
				" brew install --cask vlc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// TODO: Capture output and compare with expected
			showCommands(tt.pkgs, tt.action, tt.opts)
		})
	}
}
