# ghostty-sessions

Capture and restore Ghostty terminal sessions via AppleScript/JXA.

**Ghotty 1.3.1 new features** see <https://github.com/ghostty-org/ghostty/issues/11316>

**Status: Experiment.** The real goal is to integrate a tasking system
(when we build one) with terminal sessions - mapping tasks to tabs/cwds/titles.

## Usage

```text
Usage: ghostty-sessions.ts <command> [file]

Commands:
  show               Capture current Ghostty sessions as JSON
  save [file]        Save current sessions to file
  restore [file]     Select and restore one tab
  restore-all [file] Restore all windows and tabs

Default file: ./active-shells.json (relative to script)

Examples:
  ghostty-sessions.ts show
  ghostty-sessions.ts save
  ghostty-sessions.ts restore
  ghostty-sessions.ts restore-all
```

## Ghostty AppleScript Learnings

- `open -na Ghostty.app` forces a new app instance - avoid this
- Use native AppleScript API to create windows in existing instance:

  ```javascript
  const config = app.newSurfaceConfiguration();
  config.initialWorkingDirectory = cwd;
  app.newWindow({ withConfiguration: config });
  ```

- Title cannot be set reliably via scripting - treat as unsupported
- New windows need ~1s to stabilize before `show` captures them correctly
