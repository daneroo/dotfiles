# ghostty-sessions

Capture and restore Ghostty terminal sessions via AppleScript/JXA.

## Usage

```text
Usage: ghostty-sessions.ts <command> [file]

Commands:
  show           Show current Ghostty sessions
  save [file]    Save current sessions to JSON (default: active-shells.json)
  restore [file] Select and restore a session from JSON

Examples:
  ghostty-sessions.ts show
  ghostty-sessions.ts save
  ghostty-sessions.ts restore
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
