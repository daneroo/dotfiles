# dotfiles

**Note:** NPM Globals are now in `~/.npm-global` as decraed in our dotfile managed [`.npmrc`](npmrc). (Yes, prefix=XX means global install go here)

## Current State and Plan

This repo is for managing:

- personal config and dotfiles
- required software installed by brew formulae and casks

## Current State

- Entry point: `reconfig.sh`
  - `check.sh` is retired: it only says to run `reconfig.sh`
  - Minimal bootstrap checks (brew, go installation)
  - Builds and runs the Go implementation, forwarding its arguments

- Core functionality (Go implementation `go/cmd/reconfig/main.go`): each
  reconciler observes the machine, compares it with `config.yaml`, and shows its
  plan; the execution mode decides whether the plan is applied. In order:
  - Homebrew upgrades: `brew update`, then upgrades outdated formulae and
    casks (pinned ones excepted) and runs `brew cleanup`
  - Homebrew packages: installs missing formulae and casks and uninstalls
    unwanted ones, keeping every dependency of a desired package; waits for
    pending upgrades, since they can change dependencies
  - Versioned tools (Python, Bun) through mise: installs the versions each
    version spec resolves to, sets mise's global defaults to `config.yaml`'s
    specs, in order, uninstalls versions no spec resolves to, and removes
    strays (tools in mise's global config that `config.yaml` doesn't list)
  - npm global packages: installs missing or outdated ones, uninstalls
    unwanted ones
  - Shell completions: writes the completion scripts of aws, bun (corrected),
    docker, mise, npm and pnpm to bash-completion's per-user directory,
    `${XDG_DATA_HOME:-~/.local/share}/bash-completion/completions`, where bash
    loads each on demand, on its first Tab; only changed scripts are written,
    and scripts of tools no longer listed are removed. Other files there are
    never touched
  - Dotfiles: **GNU Stow** links `core/` into `~`; dead links in `~` are
    reported, never changed
  - Common features across all reconcilers:
    - Consistent ✓/✗/△ status reporting
    - A missing tool (mise, npm, stow, a completion tool) is reported as
      `not installed, skipping`; only real errors exit non-zero
    - Idempotent: a rerun after applying has nothing to do

Execution modes:

```bash
./reconfig.sh            # ask before applying each plan
./reconfig.sh --dry-run  # or -n: show every plan, apply nothing
./reconfig.sh --yolo     # or -y: apply every plan without asking
./reconfig.sh --help     # all options
```

Without a terminal (piped input, cron), the default mode previews only, as
`--dry-run` does. Declining a plan, or any error, previews the rest of the run.

### Desired State and Migration Plan

- Code Modernization
  - [x] Successfully bloated (6.5x) a 255-line bash script into 1,677 lines of Go. Because type safety. 🎉
  - [x] Port `installDotLinks.sh` to use GNU Stow (improved bash logic)
  - [x] Port `installDotLinks.sh` logic to Go (the Stow reconciler)
  - [ ] (future) Migrate to **Chezmoi** for native secrets and templating support
  - [ ] (feature) Detect and propose removing unused taps
  - [ ] Improve Go implementation
    - [x] Better abstractions for reconciliation loop (`go/pkg/run`)
    - [ ] More tests, but be practical
  - [ ] Implement proper UX
    - [ ] Progress indicators for long-running operations
    - [ ] Better error reporting and recovery
    - [ ] Proper logging levels (debug, info, warn, error)

- Consider a compatible equivalent implementation in Elixir/Gleam
- Documentation and Testing
  - [ ] Add succinct but complete documentation
  - [ ] Create automated tests if possible
  - [ ] Document bootstrap process

## Operating

_Systems under control:_ `galois`, the only macOS machine. The others
(davinci, shannon, dirac, goedel, feynman) are not macOS and are moving to
other setups, such as nix and omarchy; reconfig does not manage them.

Regular maintenance (_idempotent_):

```bash
just reconfig # or ./reconfig.sh
```

## TODO

- [ ] Update determinate-nix: `sudo determinate-nixd upgrade`
  - flakehub 401 on builds is expected - never signed in, harmless
- [ ] Get sanity on syno packages/config setup
- [ ] lazyrsync - installed with discovered tap
  - [ ] how does it work with syno - and can we use a newer rsync
    - [ ] <https://docs.synocommunity.com/packages/>
    - [ ] [SynoCli Network Tools](https://docs.synocommunity.com/packages/synocli-net/)
    - [ ] [SynoCli Monitor Tools](https://docs.synocommunity.com/packages/synocli-monitor/)
- [ ] Add AGENTS.md/CLAUDE.md - and make script usable by agent/skill - `/reconfig`?
- [ ] Remove unused taps (use `brew tap-info --installed --json` to reconcile)

```bash
for t in $(brew tap); do
  brew list --formula --full-name | grep -q "^${t}/" || \
  brew list --cask --full-name | grep -q "^${t}/" || \
  echo "unused: $t"
done
```

- [ ] Ghostty/Starship prompt: add colors and fonts?
  - [ ] remove ligatures
  - [ ] starship new prompt with Nerd font test
    - Default VSCode font: 'MesloLGS Nerd Font Mono', Menlo, Monaco, 'Courier New', monospace
    - Consider installing 'JetBrains Mono' and 'FiraCode Nerd Font Mono'
- [ ] manage versioned tools with [mise](https://mise.jdx.dev/) (#5)
- [ ] Per machine specialization
  - [x] CUE has been declared a disaster area
- [ ] config content: update python tooling, after move to uv/ruff
- [x] Rename the executable to: reconfig
- [ ] [Charm v2](https://charm.land/blog/v2/)
- [ ] Consider [brew bundle](https://docs.brew.sh/Brew-Bundle-and-Brewfile)

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

- start the dance: `./reconfig.sh`

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
