# VSCode et al

This documents my VSCode setup.

**Cursor and Antigravity were both removed on 2026-08-30.**

## TODO

- [x] Trim to a defensible set (66 -> 43), removing stale and duplicated extensions
- [ ] Validate that each remaining extension does something I actually want
- [ ] Retry [tode](https://terminal-code.com) - VSCode in the terminal. v0.3.4 too rough (2026-08-30)

## Ideas

Split the problem:

- validation method: minimal requirements, or cue validation of json files
- explicit requirements for extensions and workflows
  - js/ts +deno,bun, markdown, go
  - formating and linting

## Minimal extensions

We added these to the Workspace recommendations: `.vscode/extensions.json`

## Validation

- [x] Markdown
  - [ ] should format the table below:

| Name               | Extension ID                          | Check |
| ------------------ | ------------------------------------- | :---: |
| Markdownlint       | davidanson.vscode-markdownlint        |   ✗   |
| Prettier           | esbenp.prettier-vscode                |   ✗   |
| Deno               | denoland.vscode-deno                  |   ✗   |
| Code Spell Checker | streetsidesoftware.code-spell-checker |   ✗   |

## Extensions

Current state. Cursor and Antigravity lists were removed with the editors;
`git log -- README-VSCode.md` has them.

### VSCode

```bash
$ code --list-extensions
arrterian.nix-env-selector
astro-build.astro-vscode
bierner.markdown-mermaid
bradlc.vscode-tailwindcss
brody715.vscode-cuelang
charliermarsh.ruff
davidanson.vscode-markdownlint
dbaeumer.vscode-eslint
denoland.vscode-deno
dnicolson.binary-plist
eamodio.gitlens
esbenp.prettier-vscode
evilz.vscode-reveal
firsttris.vscode-jest-runner
github.codespaces
github.vscode-pull-request-github
golang.go
google.google-antigravity
jakebecker.elixir-ls
jnoortheen.nix-ide
khaeransori.json2csv
mechatroner.rainbow-csv
ms-azuretools.vscode-containers
ms-python.debugpy
ms-python.python
ms-python.vscode-pylance
ms-python.vscode-python-envs
ms-vscode-remote.remote-containers
ms-vscode-remote.remote-ssh
ms-vscode-remote.remote-ssh-edit
ms-vscode-remote.vscode-remote-extensionpack
ms-vscode.remote-explorer
ms-vscode.remote-server
ms-vsliveshare.vsliveshare
nefrob.vscode-just-syntax
pantajoe.vscode-elixir-credo
phoenixframework.phoenix
redhat.vscode-yaml
samuel-pordeus.elixir-test
streetsidesoftware.code-spell-checker
unifiedjs.vscode-mdx
vitest.explorer
yoavbls.pretty-ts-errors
```
