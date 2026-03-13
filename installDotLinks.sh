#!/usr/bin/env bash

# Dotfiles Management Script
# Currently uses GNU Stow for simple symlink farming.
#
# TODO (Future Migration to Chezmoi):
# When we need to manage secrets natively or template configurations
# across different machines (e.g., home vs work), we will migrate to Chezmoi.
# Chezmoi will replace Stow by copying target files into `~/.local/share/chezmoi`
# instead of symlinking them. This script will then be retired.


DOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "## Installing dotfiles via GNU Stow"

doSymlinks() {
    # Ensure stow is installed (macOS example)
    if ! command -v stow &> /dev/null; then
        echo "Stow not found. Please 'brew install stow'"
        exit 1
    fi

    # Run stow from the dotfiles directory, targeting the home directory
    cd "$DOTDIR" || exit 1
    stow -t ~ core
    
    if [ $? -ne 0 ]; then
        echo "✗ - Stow encountered an error"
    else
        echo "✓ - Stow complete."
    fi
}

showBadlinks(){
    echo
    # Keep your awesome dead link checker!
    echo "### Checking for dead links (like your old .nix-profile)..."
    local badLines=$(find -L "$HOME" -maxdepth 1 -name .\* -type l -ls | wc -l)
    if [ "$badLines" -eq 0 ]; then
        echo "✓ - No dead links found."
    else
        # now show them...
        find -L "$HOME" -maxdepth 1 -name .\* -type l -ls
        echo
        echo "✗ - Check the above dead links: e.g. ~/.somebadlink -> missing_file"
    fi
}

doSymlinks
showBadlinks




