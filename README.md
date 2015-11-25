# dotfiles 
Managing dotfiles for bash

    ~/.bash_profile
    ~/.bash_login NOT PRESENT
    ~/.profile
    ~/.bashrc

# install script
This script should be idempotent, and warn if any file are already present, or there are dead symlinks in $HOME

    cd ~/.dotfiles/
    ./install.sh
