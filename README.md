# dotfiles

## Operating

_Systems under control:_ `galois, davinci, shannon, dirac, goedel, feynman`

Regular maintenance (_idempotent_):

```bash
./check.sh
```

## TODO

- [ ] update python tooling, after move to uv/ruff
- [ ] move bash and go code to deno
  - [ ] separate casks
- [ ] Starship prompt: add colors and fonts?
  - Default VSCode font: 'MesloLGS Nerd Font Mono', Menlo, Monaco, 'Courier New', monospace
  - Consider installing 'JetBrains Mono' and 'FiraCode Nerd Font Mono'

## Bootstrap

### Setup `ssh`

If you need an ssh key (to clone this repo) - see [Generate ssh key](https://docs.github.com/en/authentication/connecting-to-github-with-ssh)

```bash
ssh-keygen -t ed25519 -C "daniel@newmachine"
pbcopy < ~/.ssh/id_ed25519.pub
```

Then [add it to GitHub's keys](https://github.com/settings/keys)

### Clone this repo

```bash
cd ~
# This might trigger xcode developer tools download...
git clone git@github.com:daneroo/dotfiles.git .dotfiles
cd .dotfiles
```

And setup git:

```bash
git config --global user.name "Daniel Lauzon"
git config --global user.email "daniel.lauzon@gmail.com"
```

### Install [homebrew](https://brew.sh/)

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

### Minimal formulas

```bash
# put brew in the path - just for this shell
eval "$(/opt/homebrew/bin/brew shellenv)"
brew doctor
brew install bash git go
```

### Update default shell with brew's bash

```bash
echo $HOMEBREW_PREFIX/bin/bash

# as root: sudo su -
echo /opt/homebrew/bin/bash >> /etc/shells
#  or
echo /usr/local/bin/bash >> /etc/shells

# as user:
chsh -s /opt/homebrew/bin/bash
#  or
chsh -s /usr/local/bin/bash

echo $SHELL # to confirm
```

### Normal update procedure starts

- start the dance: `./check.sh`

- nvm npm stuf ??

```bash
#?? yarn replace by corepack?
corepack enable
```

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
