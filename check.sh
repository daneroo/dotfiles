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

function nvm_update_lts() {
  export NVM_DIR=$HOME/.nvm;
  # This loads nvm (from brew installed setup, in this script)
  [ -s "${HOMEBREW_PREFIX}/opt/nvm/nvm.sh" ] && . "${HOMEBREW_PREFIX}/opt/nvm/nvm.sh"  

  # assuming default (current) is lts
  local -r current_node_version=$(nvm current)
  local -r next_node_version=$(nvm version-remote --lts)
  if [ "$current_node_version" != "$next_node_version" ]; then
    echo "✗ - Updates available"
    echo "  Upgrading to latest node lts: $next_node_version"
    echo "  from current: $current_node_version"
    local -r previous_node_version=$current_node_version
    nvm install --lts
    nvm reinstall-packages "$previous_node_version"
    nvm uninstall "$previous_node_version"
    nvm cache clear
  else
    echo "✓ - No updates: Latest LTS is default/current ($current_node_version)"    
  fi
}
echo
echo "-=-= nvm --lts"
nvm_update_lts

echo
echo "-=-= npm global requirements (slow)"
npm_global_deps="babel-cli eslint hasura-cli json lerna serve ssvmup standard typescript vercel wasm-pack yarn"
any_missing=false

for i in $npm_global_deps; do
  # echo "Checking $i"
  if npm ls -g --depth=0 --parseable 2>/dev/null | grep -q "$i"; then
    echo "✓ - Found $i"
  else
    echo "✗ - Missing $i"
    any_missing=true
  fi
done
if [ "$any_missing" = true ] ; then
    echo "Install missing:"
    echo "npm i -g $npm_global_deps"
fi

echo
echo "-=-= NPM -g outdated"
OUTDATED=$(npm -g outdated)

if [ -z "${OUTDATED}" ]; then
    echo "✓ - No npm global updates"
else
    echo "✗ - Updates available"
    echo "${OUTDATED}"
    echo " You should: ..."
fi

echo
echo "-=-= TODO clean up extraneous node nvm versions, accelerate npm -g checks (Go)"
