#!/usr/bin/env bash

echo "-= Installing dotfiles"

DOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DOTFILES="bash_profile profile bashrc"

doSymlinks () {
    for f in ${DOTFILES}; do
        safesymlink ${f}
        # ls -l ~/.${f} ${f}
    done    
}

# Will not clobber an existing file
safesymlink () {
    if [[ -z "$1" ]]; then 
        echo safesymlink: needs a dotfilename
        exit -1
    fi
    local f=$1
    local dotfile="${DOTDIR}/${f}"
    local linkfile="${HOME}/.${f}"

    # echo " * Considering $f"

    # echo Linking dotfile ${dotfile} to ${linkfile}
    if [[ ! -f ${dotfile} ]]; then
        echo "✗ - Missing dotfile ${dotfile}"
        return
    fi
    if [ "${dotfile}" -ef "${linkfile}" ]; then # $1 and $2 are different files        
        echo "✓ - ${f} already linked and identical"
        return
    fi

    # perform the link and report error
    ln -s ${dotfile} ${linkfile}
    if [ $? -ne 0 ]; then
        echo "✗ - remove ${linkfile} first"
    else
        echo "✓ - Linked dotfile ${dotfile} to ${linkfile}"
    fi
}

showBadlinks(){
    echo
    echo "Checking for broken links..."
    local badLines=`find -L $HOME -maxdepth 1 -name .\* -type l -ls | wc -l`
    if [ $badLines -eq 0 ]; then
        echo "✓ - No broken links"
    else
        # now show them...
        find -L $HOME -maxdepth 1 -name .\* -type l -ls
        echo
        echo "✗ - Check the above broken links: e.g. ~/.somebadlink -> missing_file"
    fi
}

doSymlinks
showBadlinks




