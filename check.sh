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


function update_asdf_plugins() {
  if ! command -v asdf &>/dev/null; then
    echo "✗ - asdf is not installed. Exiting"
    exit 1
  else
    echo "✓ - asdf is installed"    
  fi

  declare -a asdf_plugins=("nodejs")
  for plugin in "${asdf_plugins[@]}"; do
    if asdf plugin-list | grep -q "^$plugin$"; then
      echo "✓ - asdf plugin $plugin is installed"    
    else
      echo "✗ - asdf plugin $plugin is missing. Installing" 
      asdf plugin-add $plugin
    fi
    # TODO(danieroo): update plugins
  done

}
echo
echo "-=-= asdf - and plugins"
update_asdf_plugins

function get_latest_node_lts_version() {
  # This assumes that the json file is sorted by descending versions (latest first)
  # we want the first one that has lts != false
  #  curl -s https://nodejs.org/dist/index.json | jq -r '.[] | "\(.version) LTS: \(.lts) Security: \(.security)"'
  curl -s https://nodejs.org/dist/index.json | jq -r '[.[] | select(.lts != false)][0].version'
}

function update_node_lts() {
  desired_node_version=$(get_latest_node_lts_version)

  #  asdf list nodejs <version> will return 0 if the version is installed; and 1 if it is not
  if asdf list nodejs "${desired_node_version}" >/dev/null 2>&1; then
    echo "✓ - Node.js latest LTS (${desired_node_version}) is installed"
  else
    echo "✗ - Node.js latest LTS version ${desired_node_version} is not installed. Installing, and making global"
    # Optional: Include a command to install the version if not present
    asdf install nodejs "${desired_node_version}"
    # this is also checked below
    asdf global nodejs "${desired_node_version}"
  fi

  # Make it Global (the system default) - Check if the desired version is set as the global version
  current_global_version=$(awk '/^nodejs/ {print $2}' ~/.tool-versions)
  if [ "${current_global_version}" = "${desired_node_version}" ]; then
    echo "✓ - Node.js latest LTS (${desired_node_version}) is set as the global version"
  else
    echo "✗ - Node.js latest LTS version ${desired_node_version} is not the global version. Making it global"
    asdf global nodejs "${desired_node_version}"
  fi

  # Check for extraneous nodejs versions
  extraneous_versions=$(asdf list nodejs | grep -v "$desired_node_version" | awk '{print $1}' | tr '\n' '|')
  if [ -z "$extraneous_versions" ]; then
    echo "✓ - No extraneous Node.js versions installed."
  else
    # TODO put all versions in a single string at the end of the comment
    echo "✗ - Extraneous Node.js versions found: $extraneous_versions"
    echo "Consider removing them with: asdf uninstall nodejs $extraneous_versions (one at a time - TABTAB)"
  fi

}

echo
echo "-=-= Node.js LTS"
update_node_lts


echo
echo "-=-= npm global requirements (slow)"
# removed yarn for corepack?
npm_global_deps="corepack eslint json turbo nx pino-pretty serve standard typescript vercel npm"
any_missing=false

installed_packages=$(npm ls -g --depth=0 --parseable 2>/dev/null )
for i in $npm_global_deps; do
  # echo "Checking $i"
  if echo "${installed_packages}" | grep -q "$i"; then
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
echo "npm $(bash -c 'npm --version')"
echo "node $(bash -c 'node --version')"

OUTDATED=$(npm -g outdated)

if [ -z "${OUTDATED}" ]; then
    echo "✓ - No npm global updates"
else
    echo "✗ - Updates available"
    echo "${OUTDATED}"
    echo " You should: some subset of ..."
    echo "npm i -g $npm_global_deps"
fi
echo
echo "pnpm section:"
echo " assuming pnpm is installed with corepack, and not homebrew for now"
echo "enable and update: (no global packages yet)"
echo "corepack enable && corepack prepare pnpm@latest --activate"

echo
echo "-=-= TODO clean up extraneous node nvm versions, accelerate npm -g checks (Go)"
