#!/usr/bin/env bash
# Verify Antigravity is fully removed. Run after Pearcleaner + brew uninstall.
echo "== disk =="
for p in /Applications/Antigravity.app "/Applications/Antigravity IDE.app" \
         ~/.antigravity ~/.antigravity-ide \
         ~/Library/Application\ Support/Antigravity ~/Library/Application\ Support/Antigravity\ IDE \
         ~/Library/Logs/Antigravity ~/Library/HTTPStorages/com.google.antigravity \
         ~/Library/HTTPStorages/com.google.antigravity-ide \
         ~/Library/Preferences/com.google.antigravity.plist \
         ~/Library/Preferences/com.google.antigravity-ide.plist; do
  [ -e "$p" ] && printf "  STILL THERE  %6s  %s\n" "$(du -sh "$p" 2>/dev/null | cut -f1)" "$p"
done
echo "== brew =="
brew list --cask 2>/dev/null | grep -i antigrav | sed 's/^/  STILL LISTED  /'
echo "== dotfiles refs =="
grep -rn "antigrav" ~/.dotfiles/config.yaml ~/.dotfiles/core/.bash_profile 2>/dev/null | sed 's/^/  /'
echo "== done (no output above each heading means clean) =="
