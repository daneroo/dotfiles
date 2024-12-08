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
go run ./go/cmd/checkdeps/main.go

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

  declare -a asdf_plugins=("nodejs" "python" "deno" "bun")
  for plugin in "${asdf_plugins[@]}"; do
    if asdf plugin-list | grep -q "^$plugin$"; then
      echo "✓ - asdf plugin $plugin is installed"    
    else
      echo "✗ - asdf plugin $plugin is missing. Installing" 
      asdf plugin-add $plugin
    fi
    # Check/Apply for plugin update
    # There is no way to only "check for updates" for a plugin, so we just update it, the reporting is finetuned however.
    update_output=$(asdf plugin-update $plugin 2>&1)
    if echo "$update_output" | grep -q "Already on 'master'" || echo "$update_output" | grep -q "Your branch is up to date"; then
      echo "✓ - $plugin plugin is already up to date."
    else
      echo "✓ - $plugin plugin updated."
      echo "$update_output"
    fi
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

# This section enforces that the lated LTS version of Node.js is installed and set as the global version
# It will also warn about any extrneous versions. (Which might be fine)
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

# This section enforces that we have the lates patch version from our desired major.minor versions
# It will also warn about any extrneous versions. (Which might be fine)
function update_python_versions() {
  # Define the major.minor versions you want to maintain
  # The last one in the list will be set as the global version
  declare -a python_versions=("3.12" "3.11")

  # Initialize a string to keep all desired versions - to filter for extranous versions at the end
  desired_versions=""

  # Initialize a variable to keep track of the last installed version - it will be made global at the end
  last_installed_version=""

  # For each major.minor version, find and install the latest patch version
  for version in "${python_versions[@]}"; do
    latest_patch=$(asdf list-all python | grep -E "^${version}\.[0-9]+$" | sort -V | tail -1)
    # Keep track of this path version in our desired versions string
    desired_versions+="${latest_patch}|"

    # echo "Latest patch version for Python ${version} is ${latest_patch}"    
    if asdf list python "${latest_patch}" >/dev/null 2>&1; then
      echo "✓ - Python latest ${version} version: ${latest_patch} is installed"
    else
      echo "✗ - Python latest ${version} version: ${latest_patch} is not installed. Installing..."
      asdf install python "${latest_patch}"
    fi
    last_installed_version="${latest_patch}" # Keep track of the last installed version
  done

  # Check if the desired version is set as the global version
  current_global_version=$(awk '/^python/ {print $2}' ~/.tool-versions)
  if [ "${current_global_version}" = "${last_installed_version}" ]; then
    echo "✓ - Python ${last_installed_version} is set as the global version"
  else
    echo "✗ - Python ${last_installed_version} is not the global version. Making it global"
    asdf global python "${last_installed_version}"
  fi

  # Remove the trailing pipe character from the string: e.g.  3.12.0|3.11.6|
  desired_versions=${desired_versions%|}
  # Check for extraneous versions
  # grep -Ev "(${desired_versions})":
  #   grep -E: This option enables extended regular expressions, allowing more complex patterns.
  #   v: This option inverts the match, so grep selects lines that do not match the pattern.
  #   (${desired_versions}): The pattern is a group containing all desired versions, 
  #   separated by the pipe character |, which acts as an "or" operator in regular expressions.
  #   For example, if desired_versions is 3.12.0|3.11.6, the pattern matches either 3.12.0 or 3.11.6.
  #   So this grep command filters out the installed Python versions that are 
  #   not in the list of desired versions (i.e., the latest patch versions of 3.12 and 3.11 in your case).

  extraneous_versions=$(asdf list python | grep -Ev "(${desired_versions})" | awk '{print $1}' | tr '\n' ' ')
  if [ -z "$extraneous_versions" ]; then
    echo "✓ - No extraneous Python versions installed."
  else
    echo "✗ - Extraneous Python versions found: $extraneous_versions"
    echo "Consider removing them with: asdf uninstall python $extraneous_versions (one at a time - TABTAB)"
  fi
}

echo
echo "-=-= Python versions"
update_python_versions

function update_latest_deno() {
  latest_deno=$(asdf latest deno)
  if asdf list deno | grep -q "$latest_deno"; then
    echo "✓ - Deno latest version ($latest_deno) is installed"
  else
    echo "✗ - Deno latest version ($latest_deno) is not installed. Installing..."
    asdf install deno "$latest_deno"
  fi

  # Set and check global version
  asdf global deno "$latest_deno"
  current_global=$(asdf current deno | awk '{print $2}')
  if [ "$current_global" = "$latest_deno" ]; then
    echo "✓ - Deno $latest_deno is set as the global version"
  else
    echo "✗ - Failed to set Deno $latest_deno as the global version"
  fi

  # Check for extraneous versions
  extraneous_versions=$(asdf list deno | grep -v "$latest_deno" | awk '{print $1}' | tr '\n' '|')
  if [ -z "$extraneous_versions" ]; then
    echo "✓ - No extraneous Deno versions installed."
  else
    echo "✗ - Extraneous Deno versions found: $extraneous_versions"
    echo "Consider removing them with: asdf uninstall deno $extraneous_versions (one at a time - TABTAB)"
  fi
}

echo
echo "-=-= Deno Versions"
update_latest_deno

function update_latest_bun() {
  latest_bun=$(asdf latest bun)
  if asdf list bun | grep -q "$latest_bun"; then
    echo "✓ - Bun latest version ($latest_bun) is installed"
  else
    echo "✗ - Bun latest version ($latest_bun) is not installed. Installing..."
    asdf install bun "$latest_bun"
  fi

  # Set and check global version
  asdf global bun "$latest_bun"
  current_global=$(asdf current bun | awk '{print $2}')
  if [ "$current_global" = "$latest_bun" ]; then
    echo "✓ - Bun $latest_bun is set as the global version"
  else
    echo "✗ - Failed to set Bun $latest_bun as the global version"
  fi

  # Check for extraneous versions
  extraneous_versions=$(asdf list bun | grep -v "$latest_bun" | awk '{print $1}' | tr '\n' '|')
  if [ -z "$extraneous_versions" ]; then
    echo "✓ - No extraneous Bun versions installed."
  else
    echo "✗ - Extraneous Bun versions found: $extraneous_versions"
    echo "Consider removing them with: asdf uninstall bun $extraneous_versions (one at a time - TABTAB)"
  fi
}

echo
echo "-=-= Bun Versions"
update_latest_bun

update_npm_completions() {
    NPM_COMPLETION_FILE="./incl/npm_completion.sh"

    # Check if npm command is available
    if ! command -v npm &>/dev/null; then
        echo "✗ - npm command not found"
        return 1
    fi

    # Generate the current npm completion text and its SHA256 hash
    completion_text=$(npm completion)
    completion_sha256=$(echo "${completion_text}" | sha256sum | awk '{print $1}')

    # Default to requiring an update
    update_required=true

    # Check if the file exists and compare hashes to determine if update is needed
    if [[ -r "${NPM_COMPLETION_FILE}" ]]; then
        file_sha256=$(sha256sum "${NPM_COMPLETION_FILE}" | awk '{print $1}')
        if [[ "${completion_sha256}" == "${file_sha256}" ]]; then
            update_required=false
            echo "✓ - npm completions are up to date (sha256)"
        fi
    fi

    # Update the completion file if required
    if [[ "${update_required}" == "true" ]]; then
        echo "✗ - npm completions need updating (sha256: ${completion_sha256})"
        # Ensure the incl directory exists
        mkdir -p ./incl
        echo "${completion_text}" > "${NPM_COMPLETION_FILE}"
    fi
}

echo
echo "-=-= NPM completions"
update_npm_completions


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
