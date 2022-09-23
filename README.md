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

- [ ] Prompt

  - [ ] Starship - add kubernetes and docker-context
  - [ ] ifelse starship, kubeon
  - [ ] Profile shell startup performance?

```bash
# hyperfine
time bash -c 'exit'
time bash -i -c 'exit'
# and may play with --norc and --noprofile.
hyperfine bash -c 'exit'
hyperfine bash -i -c 'exit'
```

- Clean up

- [ ] [bashrc vs bash_profile vs ..](https://superuser.com/questions/789448/choosing-between-bashrc-profile-bash-profile-etc)
- Bootstrap new machine script? casks, default writes.. (see Kent)
- Zsh - determine why
- brewDeps (verbose flag), pretty map
- speed up npm global deps, and find extraneous
  - remove babel-cli,...

## References

- [Strap](https://github.com/MikeMcQuaid/strap)
  - [Brew Bundle](https://github.com/Homebrew/homebrew-bundle)
  - [Brew Cask](https://github.com/Homebrew/homebrew-cask)
- [Steal from Kent C Dodds](https://github.com/kentcdodds/dotfiles/blob/main/.macos)
- [M1'ify](https://blog.smittytone.net/2021/02/07/how-to-migrate-to-native-homebrew-on-an-m1-mac/)
- [bashrc vs bash_profile vs ..](https://superuser.com/questions/789448/choosing-between-bashrc-profile-bash-profile-etc)
- [profile bash startup](https://stackoverflow.com/questions/5014823/how-can-i-profile-a-bash-shell-script-slow-startup)
