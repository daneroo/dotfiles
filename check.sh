#!/usr/bin/env bash

echo "-= Regular maintenance"

DOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo
./installDotLinks.sh

echo
echo "-=-= Brew update"
brew update 

echo
echo "-=-= Brew deps (mine)"
go run checkBrew.go

echo
echo "-=-= Brew outdated"
OUTDATED=$(brew outdated -v)

if [ -z "${OUTDATED}" ]; then
    echo "✓ - No updates"
else
    echo "✗ - Updates available"
    echo "${OUTDATED}"
    echo " You should:"
    echo "brew upgrade && brew cleanup"
fi


