# echo "*** now executing empty .bashrc"
# This should be empty 
# everything goes into .bash_profile
[ -n "$PS1" ] && source ~/.bash_profile;

export NVM_DIR="/Users/daniel/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"  # This loads nvm
