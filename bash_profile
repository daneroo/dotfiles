# echo "*** now executing .bash_profile"
#echo PATH: $PATH
# everything goes into .bash_profile
#   .bashrc sources this, and .profile is empty

# This is to perform timing of the bash_profile load
# start_time=$(/opt/homebrew/bin/gdate +%s%N)  # Start time in nanoseconds

# Move this up and make it conditional on platform
# Sets HOMEBREW vars and PATH's - brew shellenv : to see output
#  - consider both known candidate prefixes
#  - since this sets HOMEBREW_PREFIX, no longer need to invoke $(brew --prefix)
for homebrew_pfx in /usr/local /opt/homebrew; do
  if [ -x ${homebrew_pfx}/bin/brew ]; then
    # set the variables and paths
    eval "$(${homebrew_pfx}/bin/brew shellenv)"
  fi
done

# brew's bash completion - assumes HOMEBREW_PREFIX is set
[[ -r "${HOMEBREW_PREFIX}/etc/profile.d/bash_completion.sh" ]] && . "${HOMEBREW_PREFIX}/etc/profile.d/bash_completion.sh"

# ASDF Setup
ASDF_DATA_DIR=$HOME/.asdf
# Add asdf shims to PATH
export PATH="${ASDF_DATA_DIR:-$HOME/.asdf}/shims:$PATH"
# asdf completion
# if asdf is installed, use its completion
if command -v asdf &>/dev/null; then
  . <(asdf completion bash)
fi

# NPM completion
# This command is slow so was replaced by the file below, generated and checked in check.sh
# command -v npm &>/dev/null && source <(npm completion)
NPM_COMPLETION_FILE=~/.dotfiles/incl/npm_completion.sh
[[ -r "${NPM_COMPLETION_FILE}" ]] && . "${NPM_COMPLETION_FILE}"

# PNPM completion
# * This replaces the old tabtab completion
PNPM_COMPLETION_FILE=~/.dotfiles/incl/pnpm_completion.bash
[[ -r "${PNPM_COMPLETION_FILE}" ]] && . "${PNPM_COMPLETION_FILE}"

# Old .profile content
# This is source'd from .bash_profile, since I installed rvm!

# kube-ps1 prompt functions
# side effect: appends _kube_ps1_update_cache to PROMPT_COMMAND
if [ -f ${HOMEBREW_PREFIX}/opt/kube-ps1/share/kube-ps1.sh ]; then
  . ${HOMEBREW_PREFIX}/opt/kube-ps1/share/kube-ps1.sh
fi

# brew install awscli / aws completion
complete -C aws_completer aws

# Path put /usr/local/bin ahead of /usr/bin
# this is redundant if HOMEBREW_PREFIX is /usr/local
export PATH="/usr/local/bin:/usr/local/sbin${PATH+:$PATH}";


# Mac OSX color stuff
# ys TERM var should be xterm-color
# and for git stuff:
#   git config color.ui true
export CLICOLOR=1

export TIMEFORMAT="%Rs"
#for human readble, use export TIMEFORMAT="%lR"
alias ls='ls -sF'
alias pp='pushd'
alias po='popd'
alias c='cursor'

# This is gum format's default theme
# Set GUM theme to dark by default, light for Apple Terminal
export GUM_FORMAT_THEME="dark"
if [[ "${TERM_PROGRAM}" == "Apple_Terminal" ]]; then
  export GUM_FORMAT_THEME="light"
fi

# for Go, without docker
export GOPATH=$HOME/Code/Go
# If I want to export my own built binaries, or installed go utils (govend)
export PATH=$PATH:$GOPATH/bin

# for Rust, cargo's build dir in PATH
if [ -f  "$HOME/.cargo/env" ]; then
  . "$HOME/.cargo/env"
fi

# for Deno as installed from denoland's install script
export PATH=$PATH:$HOME/.deno/bin

# For ngs NATS.io utility
export PATH=$PATH:$HOME/.ngs/bin  #Add NGS utility to the path


# Starship prompt
export STARSHIP_CONFIG=~/.dotfiles/starship.toml
# starship-nice.toml requires nerdfont
# export STARSHIP_CONFIG=~/.dotfiles/starship-nice.toml
eval "$(starship init bash)"

# This is to perform timing of the bash_profile load: matches section above
# end_time=$(gdate +%s%N)  # End time in nanoseconds
# elapsed_time=$((end_time - start_time))  # Elapsed time in nanoseconds
# echo "bash_profile load time: $((elapsed_time / 1000000)) ms"
