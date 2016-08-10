# dotfiles 
Managing dotfiles for bash

    ~/.bash_profile
    ~/.bash_login NOT PRESENT
    ~/.profile
    ~/.bashrc

## TODO
Make some sections for:

- brew/case (stephane)
- nvm ?
- [npm globals](https://github.com/sindresorhus/guides/blob/master/npm-global-without-sudo.md)

    mkdir "${HOME}/.npm-packages"

## npm global package list

    npm uninstall -g babel-cli eslint gulp-cli http-server json serverless uglify-js

## install script
This script should be idempotent, and warn if any file are already present, or there are dead symlinks in $HOME

    cd ~/.dotfiles/
    ./install.sh


## completion (for docker)

brew install bash 

# add to allowed shells as root
sudo su -
echo /usr/local/bin/bash >> /etc/shells
# as daniel
chsh -s /usr/local/bin/bash

brew install homebrew/completions/docker-completion

cd completion.d
export SRC='https://raw.githubusercontent.com/docker/'
wget ${SRC}/docker/master/contrib/completion/bash/docker -O ./completion.d/docker-completion.sh
wget ${SRC}/compose/$(docker-compose --version | awk 'NR==1{print $NF}')/contrib/completion/bash/docker-compose -O ./completion.d/docker-compose-completion.sh
wget ${SRC}/machine/master/contrib/completion/bash/docker-machine.bash -O ./completion.d/docker-machine-completion.sh
