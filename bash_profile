# echo "*** now executing .bash_profile"
#echo PATH: $PATH
# everything goes into .bash_profile
#   .bashrc sources this, and .profile is empty

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

# NVM Setup
export NVM_DIR="$HOME/.nvm"
[ -s "${HOMEBREW_PREFIX}/opt/nvm/nvm.sh" ] && . "${HOMEBREW_PREFIX}/opt/nvm/nvm.sh"  # This loads nvm
[ -s "${HOMEBREW_PREFIX}/opt/nvm/etc/bash_completion.d/nvm" ] && . "${HOMEBREW_PREFIX}/opt/nvm/etc/bash_completion.d/nvm"  # This loads nvm bash_completion

# Old .profile content
# This is source'd from .bash_profile, since I installed rvm!

# gcloud (brew) completion : brew cask install google-cloud-sdk
if [ -f ${HOMEBREW_PREFIX}/Caskroom/google-cloud-sdk/latest/google-cloud-sdk/completion.bash.inc ]; then
  . ${HOMEBREW_PREFIX}/Caskroom/google-cloud-sdk/latest/google-cloud-sdk/completion.bash.inc
fi

# kube-ps1 prompt functions
# side effect: appends _kube_ps1_update_cache to PROMPT_COMMAND
if [ -f ${HOMEBREW_PREFIX}/opt/kube-ps1/share/kube-ps1.sh ]; then
  . ${HOMEBREW_PREFIX}/opt/kube-ps1/share/kube-ps1.sh
fi

# brew install awscli / aws completion
complete -C aws_completer aws

##### PROMPT START - replaced with Starship #####
# # *** Prompt: (bash+git+kube)
# # Prompt uses PROMPT_COMMAND instead of PS1, 
# # because git prompt only supports color when used that way.
# # Note that PROMPT_COMMAND was appended to by kube-ps1.sh
# PROMPT_DIRTRIM=2  # depth of directory for \w directive

# GIT_PS1_SHOWUPSTREAM="auto"
# GIT_PS1_SHOWCOLORHINTS=true
# GIT_PS1_SHOWDIRTYSTATE=true
# GIT_PS1_SHOWUNTRACKEDFILES=true

# # KUBE_PS1_SYMBOL_ENABLE=false # default is true
# KUBE_PS1_SEPARATOR=''  # to remove separator, because symbol addds a space.

# ## Don't forget to append previous PROMPT_COMMAND..
# PROMPT_PFX='\u@\h:\w'
# if [ -n "$KUBE_PS1_BINARY" ]; then
#   PROMPT_SFX='$(kube_ps1)$ '
#   kubeoff
# else
#   PROMPT_SFX='$ ' # if no kube-ps1
# fi
# PROMPT_COMMAND="__git_ps1 '${PROMPT_PFX}' '${PROMPT_SFX}'; ${PROMPT_COMMAND}"
##### PROMPT END #####

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

#export JAVA_HOME=/System/Library/Frameworks/JavaVM.framework/Home/

# Put Homebrew's Python ahead in the path
export PATH="${HOMEBREW_PREFIX}/opt/python/libexec/bin:$PATH"

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

## `pnpm install-completion` genrated this
##  It should be run on each host to create the target files in ~/.config/tabtab
# tabtab source for packages
# uninstall by removing these lines
[ -f ~/.config/tabtab/bash/__tabtab.bash ] && . ~/.config/tabtab/bash/__tabtab.bash || true

# Starship prompt
export STARSHIP_CONFIG=~/.dotfiles/starship.toml
# starship-nice.toml requires nerdfont
# export STARSHIP_CONFIG=~/.dotfiles/starship-nice.toml
eval "$(starship init bash)"

