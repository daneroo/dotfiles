#!/usr/bin/env bash

echo "-= Regular maintenance"

DOTDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo
echo "-=-= Bootstrap"
# Check for brew
if ! command -v brew &>/dev/null; then
    echo "✗ - brew is missing, install it with:"
    echo '/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"'
    exit 1
fi
# Check for go
if ! command -v go &>/dev/null; then
    echo "✓ - brew is installed but go is missing"
    echo " temporarily install it with (it may be replaced with an asdf version later):"
    echo " brew install go"
    exit 1
else
    echo "✓ - brew is installed and go is available"
fi

echo
./installDotLinks.sh

echo
echo "-=-= Brew deps (mine)"
go run ./go/cmd/checkdeps/main.go
code=$?
echo "Debug: raw exit code: ${code}"
if [ $code -ne 0 ]; then
    echo "Exiting: checkdeps failed with error code: ${code}"
    exit "${code}"
fi

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
