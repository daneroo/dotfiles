# dotfiles

## Current State and Plan

This repo is for managing:

- personal config and dotfiles
- required software installed by brew formulae and casks

### Current State

- Entry point: `check.sh`
  - Minimal bootstrap checks (brew, go installation)
  - Orchestrates the reconciliation process
  - Manages dotfiles via `installDotLinks.sh` (last bash holdout)
    - Maintains and validates symlinks for bash configuration files
  - Delegates all dependency management to Go implementation

- Core functionality (Go implementation `go/cmd/checkdeps/main.go`):
  - Manages Homebrew packages
    - Validates installed formulae and casks against desired state
    - Detects missing packages
    - Identifies extraneous packages
    - Ensures transitive dependencies are maintained
  - Manages runtime versions via asdf
    - Manages plugins: Node.js, Python, Deno, Bun
    - For each plugin:
      - Automatic installation of desired versions
      - Global version management
      - Detection of extraneous versions
      - Plugin updates
  - Manages npm global packages
    - Desired packages: corepack, eslint, json, turbo, nx, etc.
    - Maintains shell completions
    - Reports outdated packages
    - Proposes upgrades
    - Handles pnpm via corepack
  - Common features across all reconcilers:
    - Consistent ✓/✗ status reporting
    - Idempotent reconciliation pattern
    - New implementation (1,677 lines of Go <--> 255 lines of bash)

### Desired State and Migration Plan

- Code Modernization
  - [x] Successfully bloated (6.5x) a 255-line bash script into 1,677 lines of Go. Because type safety. 🎉
  - [ ] (not now) Port `installDotLinks.sh` to Go (last bash holdout)
  - [ ] (feature) Detect and propose removing unused taps
  - [ ] Improve Go implementation
    - [ ] Better abstractions for reconciliation loop
    - [ ] More tests, but be practical
  - [ ] Implement proper UX
    - [ ] Progress indicators for long-running operations
    - [ ] Better error reporting and recovery
    - [ ] Proper logging levels (debug, info, warn, error)

- Consider a compatible equivalent implementation in Deno/Typescript
- Consider a compatible equivalent implementation in Elixir/Gleam
- Documentation and Testing
  - [ ] Add succinct but complete documentation
  - [ ] Create automated tests if possible
  - [ ] Document bootstrap process

## Operating

*Systems under control:* `galois, davinci, shannon, dirac, goedel, feynman`

Regular maintenance (*idempotent*):

```bash
./check.sh
```

## TODO

- Rename the executable to: reconfig
- Configuration Enhancement
  - [x] Move config out of brewdeps package, add asdf, npm sections
  - [ ] Per machine specialization
    - [x] CUE has been declared a disaster area
- [ ] update python tooling, after move to uv/ruff
- [ ] Ghostty/Starship prompt: add colors and fonts?
  - Default VSCode font: 'MesloLGS Nerd Font Mono', Menlo, Monaco, 'Courier New', monospace
  - Consider installing 'JetBrains Mono' and 'FiraCode Nerd Font Mono'

## Bootstrap

### Setup `ssh`

If you need an ssh key (to clone this repo) - see [Generate ssh key](https://docs.github.com/en/authentication/connecting-to-github-with-ssh)

```bash
ssh-keygen -t ed25519 -C "daniel@new_machine"
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

- nvm npm stuff ??

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
