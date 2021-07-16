#echo "*** now executing .bash_profile"
#echo PATH: $PATH
# everything goes into .bash_profile
#   .bashrc sources this, and .profile is empty

# Silence the MacOS default zsh warning message
export BASH_SILENCE_DEPRECATION_WARNING=1

# Move this up and make it conditional on platform
# Sets HOMEBREW vars and PATH's - brew shellenv : to see output
#  - consider both known candidate prefixes
#  - since this sets HOMEBREW_PREFIX, no longer need to invoke $(brew --prefix)
for homebrew_pfx in /usr/local /opt/homebrew; do
  echo testing $homebrew_pfx
  if [ -x ${homebrew_pfx}/bin/brew ]; then
    echo executing $homebrew_pfx
    eval "$(${homebrew_pfx}/bin/brew shellenv)"
  fi
done

# brew's bash completion - assumes HOMEBREW_PREFIX is set
[[ -r "${HOMEBREW_PREFIX}/etc/profile.d/bash_completion.sh" ]] && . "${HOMEBREW_PREFIX}/etc/profile.d/bash_completion.sh"

# NVM Setup
export NVM_DIR="$HOME/.nvm"
[ -s "/usr/local/opt/nvm/nvm.sh" ] && . "/usr/local/opt/nvm/nvm.sh"  # This loads nvm
[ -s "/usr/local/opt/nvm/etc/bash_completion.d/nvm" ] && . "/usr/local/opt/nvm/etc/bash_completion.d/nvm"  # This loads nvm bash_completion

# Old .profile content
# This is source'd from .bash_profile, since I installed rvm!

# gcloud (brew) completion : brew cask install google-cloud-sdk
if [ -f $(brew --prefix)/Caskroom/google-cloud-sdk/latest/google-cloud-sdk/completion.bash.inc ]; then
  . $(brew --prefix)/Caskroom/google-cloud-sdk/latest/google-cloud-sdk/completion.bash.inc
fi

# kube-ps1 prompt functions
# side effect: appends _kube_ps1_update_cache to PROMPT_COMMAND
if [ -f ${HOMEBREW_PREFIX}/opt/kube-ps1/share/kube-ps1.sh ]; then
  . ${HOMEBREW_PREFIX}/opt/kube-ps1/share/kube-ps1.sh
fi

# brew install awscli / aws completion
complete -C aws_completer aws

# *** Prompt: (bash+git+kube)
# Prompt uses PROMPT_COMMAND instead of PS1, 
# because git prompt only supports color when used that way.
# Note that PROMPT_COMMAND was appended to by kube-ps1.sh
PROMPT_DIRTRIM=2  # depth of directory for \w directive

GIT_PS1_SHOWUPSTREAM="auto"
GIT_PS1_SHOWCOLORHINTS=true
GIT_PS1_SHOWDIRTYSTATE=true
GIT_PS1_SHOWUNTRACKEDFILES=true

# KUBE_PS1_SYMBOL_ENABLE=false # default is true
KUBE_PS1_SEPARATOR=''  # to remove separator, because symbol addds a space.

## Don't forget to append previous PROMPT_COMMAND..
PROMPT_PFX='\u@\h:\w'
if [ -n "$KUBE_PS1_BINARY" ]; then
  PROMPT_SFX='$(kube_ps1)$ '
  kubeoff
else
  PROMPT_SFX='$ ' # if no kube-ps1
fi
PROMPT_COMMAND="__git_ps1 '${PROMPT_PFX}' '${PROMPT_SFX}'; ${PROMPT_COMMAND}"

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
export PATH=$PATH:$HOME/.cargo/bin

# for Deno as installed from denoland's install script
export PATH=$PATH:$HOME/.deno/bin

# For ngs NATS.io utility
export PATH=$PATH:$HOME/.ngs/bin  #Add NGS utility to the path

# For Chia on MacOS
CHIA_PATH='/Applications/Chia.app/Contents/Resources/app.asar.unpacked/daemon'
if [ -d ${CHIA_PATH} ]; then
  export PATH=$PATH:${CHIA_PATH}
fi



# Docker default on OSX: careful if this dotfile goes to cantor/ubuntu
# Should go to extras ?? or if boot2docker exists...
# export DOCKER_HOST=tcp://192.168.59.103:2375
alias b2d='boot2docker init && boot2docker up && $(boot2docker shellinit)'
alias dme='eval "$(docker-machine env dev)"; env|grep DOCKER;echo docker-machine env dev set'
alias dmc='docker-machine create -d virtualbox --virtualbox-disk-size "40000" --virtualbox-memory "4096" --virtualbox-cpu-count "2" dev'
alias dangling='docker rmi $(docker images --quiet --filter "dangling=true")'
