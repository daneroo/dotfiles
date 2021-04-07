# dotfiles

## Operating

_Systems under control:_ `dirac,shannon,MB292 (goedel)`

Regular maintenance (_idempotent_):

```bash
./check.sh
```

### Note for `gcloud` cask

This is not captured in brewDeps: `brew cask install google-cloud-sdk`

## TODO

- [Strap](https://github.com/MikeMcQuaid/strap)
  - [Brew Bundle](https://github.com/Homebrew/homebrew-bundle)
  - [Brew Cask](https://github.com/Homebrew/homebrew-cask)
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

This script should be idempotent, and warn if any file are already present, or there are dead symbolic links in `$HOME/`.

```bash
cd ~/.dotfiles/
./installDotLinks.sh
```

## Brew management

```bash
brew update
brew outdated
brew upgrade
brew cleanup

brew tap beeftornado/rmtree
brew rmtree thing
```

Checking for extra casks

```bash
go run checkBrew.go
```
