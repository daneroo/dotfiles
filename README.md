# dotfiles

Managing dotfiles for bash

```bash
  ~/.bash_profile
  ~/.bash_login NOT PRESENT
  ~/.profile
  ~/.bashrc
```

## TODO

Make some sections for:

- List of necessary installs
  - applications (brew cask?)
  - brew
  - npm

## npm global package list

```bash
  ├── babel-cli@6.11.4
  ├── eslint@3.3.0
  ├── gulp-cli@1.2.2
  ├── http-server@0.9.0
  ├── json@9.0.4
  ├── npm@6.4.1
  └── uglify-js@2.7.0
```

## install script

This script should be idempotent, and warn if any file are already present, or there are dead symlinks in $HOME

```bash
cd ~/.dotfiles/
./install.sh
```

## Brew management

```bash
brew update
brew outdated
brew upgrade
brew cleanup

brew rmtree thing
```

Checking for extra casks

```bash
go run checkBrew.go
```
