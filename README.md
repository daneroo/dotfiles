# dotfiles

## Operating

_Systems under control:_ `dirac,shannon,davinci(M1),goedel),feynman`

Regular maintenance (_idempotent_):

```bash
./check.sh
```

## Bootstrap

- First install [homebrew](https://brew.sh/).
- then minimal formulas
- start the dance: `./check.sh`

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

brew doctor
brew install git go

ssh-keygen -t ed25519 -C "daniel.lauzon@gmail.com"
pbcopy < ~/.ssh/id_ed25519.pub
# add key to github
git clone git@github.com:daneroo/dotfiles.git .dotfiles
cd .dotfiles
./check.sh

# as root: sudo su -
echo /usr/local/bin/bash >> /etc/shells
#as user:
chsh -s /usr/local/bin/bash
echo $SHELL # to confirm

#?? yarn replace by corepack?
corepack enable

```

## TODO

- [Steal from Kent C Dodds](https://github.com/kentcdodds/dotfiles/blob/main/.macos)
- [M1'ify](https://blog.smittytone.net/2021/02/07/how-to-migrate-to-native-homebrew-on-an-m1-mac/)

- brewDeps (verbose flag), pretty map
- speed up npm global deps, and find extraneous
  - remove babel-cli,...
- gcloud - note in bash_profile, and test
- [Strap](https://github.com/MikeMcQuaid/strap)

  - [Brew Bundle](https://github.com/Homebrew/homebrew-bundle)
  - [Brew Cask](https://github.com/Homebrew/homebrew-cask)
    Make some sections for:

- List of necessary installs
  - applications (brew cask?)
  - brew
  - npm

```

```
