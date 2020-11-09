# echo "*** now executing .bash_profile"
# everything goes into .bash_profile
#   .bashrc sources this, and .profile is empty

# could do some bash foo magic: 
DOTFILES=${HOME}/.dotfiles

if [ -e ~/.node_completion ] ; then
# {{{
# Node Completion - Auto-generated, do not touch.
shopt -s progcomp
for f in $(command ls ~/.node-completion); do
  f="$HOME/.node-completion/$f"
  test -f "$f" && . "$f"
done
# }}}
fi

# NVM Setup
export NVM_DIR="$HOME/.nvm"
[ -s "/usr/local/opt/nvm/nvm.sh" ] && . "/usr/local/opt/nvm/nvm.sh"  # This loads nvm
[ -s "/usr/local/opt/nvm/etc/bash_completion.d/nvm" ] && . "/usr/local/opt/nvm/etc/bash_completion.d/nvm"  # This loads nvm bash_completion

# Where should this list go?
# npm install -g babel-cli eslint gulp-cli http-server json uglify-js


# Old .profile content
# This is source'd from .bash_profile, since I installed rvm!

if [ -f $(brew --prefix)/etc/bash_completion ]; then
  . $(brew --prefix)/etc/bash_completion
fi

# brew's git and completion : brew install git bash-completion
if [ -f $(brew --prefix)/etc/bash_completion.d/git-prompt.sh ]; then
  . $(brew --prefix)/etc/bash_completion.d/git-prompt.sh
fi
if [ -f $(brew --prefix)/etc/bash_completion.d/git-completion.bash ]; then
  . $(brew --prefix)/etc/bash_completion.d/git-completion.bash
fi

# gcloud (brew) completion : brew cask install google-cloud-sdk
if [ -f $(brew --prefix)/Caskroom/google-cloud-sdk/latest/google-cloud-sdk/completion.bash.inc ]; then
  . $(brew --prefix)/Caskroom/google-cloud-sdk/latest/google-cloud-sdk/completion.bash.inc
fi

# docker completion
if [ -f $(brew --prefix)/etc/bash_completion.d/docker ]; then
  . $(brew --prefix)/etc/bash_completion.d/docker
fi

# kube completion - conditional, 
# TODO: unless? brew install bash-completion@2
# see kubectl completion -h
if [ -f $(brew --prefix)/bin/kubectl ]; then
  source <(kubectl completion bash)
fi

# kube-ps1 prompt functions
# side effect: appends _kube_ps1_update_cache to PROMPT_COMMAND
if [ -f $(brew --prefix)/opt/kube-ps1/share/kube-ps1.sh ]; then
  . $(brew --prefix)/opt/kube-ps1/share/kube-ps1.sh
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

# put /usr/local/bin ahead of /usr/bin
# as per homebrew's suggestion add /usr/local/sbin
export PATH=/usr/local/sbin:/usr/local/bin:$PATH


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

# for Go, without docker
export GOPATH=$HOME/Code/Go
# If I want to export my own built binaries, or installed go utils (govend)
export PATH=$PATH:$GOPATH/bin

# for Rust, cargo's build dir in PATH
export PATH=$PATH:$HOME/.cargo/bin

# For ngs NATS.io utility
export PATH=$HOME/.ngs/bin:$PATH  #Add NGS utility to the path

# Docker default on OSX: careful if this dotfile goes to cantor/ubuntu
# Should go to extras ?? or if boot2docker exists...
# export DOCKER_HOST=tcp://192.168.59.103:2375
alias b2d='boot2docker init && boot2docker up && $(boot2docker shellinit)'
alias dme='eval "$(docker-machine env dev)"; env|grep DOCKER;echo docker-machine env dev set'
alias dmc='docker-machine create -d virtualbox --virtualbox-disk-size "40000" --virtualbox-memory "4096" --virtualbox-cpu-count "2" dev'
alias dangling='docker rmi $(docker images --quiet --filter "dangling=true")'


# Path for Rust
export PATH="$PATH:$HOME/.cargo/bin"
