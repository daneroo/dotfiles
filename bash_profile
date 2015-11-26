# echo "*** now executing .bash_profile"
# everything goes into .bash_profile
#   .bashrc sources this, and .profile is empty

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

# This is how we keep global packages...
# nvm install v4 --reinstall-packages-from=0.10.31
TIMEFORMAT="nvm.sh took %lR" # reset later
time {
	# export NVM_DIR="/Users/daniel/.nvm"
    export NVM_SYMLINK_CURRENT=true
    [ -s "/Users/daniel/.nvm/nvm.sh" ] && . "/Users/daniel/.nvm/nvm.sh" # This loads nvm    
}
export TIMEFORMAT="%lR"

# Old .profile content
# This is source'd from .bash_profile, since I installed rvm!

# brew's git and completion : brew install git bash-completion
if [ -f $(brew --prefix)/etc/bash_completion.d/git-prompt.sh ]; then
  . $(brew --prefix)/etc/bash_completion.d/git-prompt.sh
fi
if [ -f $(brew --prefix)/etc/bash_completion.d/git-completion.bash ]; then
  . $(brew --prefix)/etc/bash_completion.d/git-completion.bash
fi

 
GIT_PS1_SHOWUPSTREAM="auto"
GIT_PS1_SHOWCOLORHINTS="yes"

## modified to keep updated terminal cwd: for nerw tabs
# Fixed my problems by not 'exporting PROMPT_COMMAND'
PROMPT_COMMAND="__git_ps1 '\u@\h:\w' '\\$ '; $PROMPT_COMMAND"
PS1='\h:\W$(__git_ps1 "(%s)") \u\$ '

# put /usr/local/bin ahead of /usr/bin
export PATH=/usr/local/bin:$PATH

# Mac OSX color stuff
# my TERM var should be xterm-color
# and for git stuff:
#   git config color.ui true
export CLICOLOR=1

export TIMEFORMAT="%lR"
#TIMEFORMAT="%lR"
alias ls='ls -sF'
alias pp='pushd'
alias po='popd'

# add mongo to path
#export MONGO_HOME=~/Downloads/mongo
#export PATH=$PATH:$MONGO_HOME/bin

#export JAVA_HOME=/System/Library/Frameworks/JavaVM.framework/Home/

# as per homebrew's suggestion add /usr/local/sbin
export PATH=$PATH:/usr/local/sbin

# for Go, without docker
export GOPATH=$HOME/Code/Go
# If I want to export my own built binaries, or installed go utils (govend)
export PATH=$PATH:$GOPATH/bin
export GO15VENDOREXPERIMENT=1

# Android ADT path for phonegap to work
#export ADTHOME=/Users/daniel/Downloads/devops/adt-bundle-mac-x86_64-20130917
#export PATH=${PATH}:${ADTHOME}/sdk/platform-tools:${ADTHOME}/sdk/tools

### Added by the Heroku Toolbelt
export PATH="/usr/local/heroku/bin:$PATH"

# Docker default on OSX: careful if this dotfile goes to cantor/ubuntu
# Should go to extr5as ?? or if boot2docker exists...
# export DOCKER_HOST=tcp://192.168.59.103:2375
alias b2d='boot2docker init && boot2docker up && $(boot2docker shellinit)'
alias dme='eval "$(docker-machine env dev)"; env|grep DOCKER;echo docker-machine env dev set'
alias dmc='docker-machine create -d virtualbox --virtualbox-disk-size "40000" --virtualbox-memory "2048" --virtualbox-cpu-count "2" dev'
