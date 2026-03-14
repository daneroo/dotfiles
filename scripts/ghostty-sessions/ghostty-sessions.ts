#!/usr/bin/env bun
// ghostty-sessions - Save and restore Ghostty terminal sessions
// Requires: macOS, Ghostty 1.3+, bun, gum

import { $ } from "bun";
import { homedir } from "os";

const DEFAULT_FILE = `${import.meta.dir}/active-shells.json`;
const DEFAULT_TIMEOUT_MS = 30_000;
const APP_NAME = "Ghostty";

interface Tab {
  title: string;
  cwd: string;
}

interface Window {
  tabs: Tab[];
}

// Capture current Ghostty sessions via JXA
async function captureSessions(): Promise<Window[]> {
  const jxa = `
    const app = Application('Ghostty');
    const sessions = app.windows().map(w => ({
      tabs: w.tabs().map(t => ({
        title: t.name(),
        cwd: t.terminals()[0].workingDirectory()
      }))
    }));
    JSON.stringify(sessions);
  `;
  const result = await $`osascript -l JavaScript -e ${jxa}`.text();
  return JSON.parse(result.trim());
}

async function cmdShow() {
  const sessions = await captureSessions();
  console.log(JSON.stringify(sessions, null, 2));
}

async function cmdSave(file: string) {
  const sessions = await captureSessions();
  await Bun.write(file, JSON.stringify(sessions, null, 2) + "\n");
  console.log(`Saved sessions to: ${file}`);
}

async function cmdRestore(file: string) {
  const content = await Bun.file(file).text();
  const windows: Window[] = JSON.parse(content);

  // Flatten all tabs
  const tabs = windows.flatMap((w) => w.tabs);
  if (tabs.length === 0) {
    console.log("No sessions found");
    return;
  }

  // Use gum to select - pass choices as arguments since gum supports that
  const titles = tabs.map((t) => t.title);
  const gum = Bun.spawnSync([
    "gum",
    "choose",
    "--header",
    "Select session to restore:",
    ...titles,
  ], {
    stdin: "inherit",
    stderr: "inherit",
  });
  const selected = gum.stdout.toString();
  const selectedTitle = selected.trim();

  if (!selectedTitle) {
    console.log("No selection made");
    return;
  }

  const tab = tabs.find((t) => t.title === selectedTitle);
  if (!tab) return;

  // Expand ~ to $HOME (Ghostty's --working-directory doesn't expand ~)
  const cwd = tab.cwd.replace(/^~/, homedir());

  console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
  console.log(`Selected: ${selectedTitle}`);
  console.log(`Directory: ${cwd}`);
  console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
  console.log();

  // Ask user to confirm before opening (prevents accidental window spawning)
  const confirm = Bun.spawnSync(
    ["gum", "confirm", "Open new Ghostty window?"],
    {
      stdin: "inherit",
      stdout: "inherit",
      stderr: "inherit",
    },
  );
  if (confirm.exitCode !== 0) {
    // User declined - provide command to copy manually as fallback
    console.log("\nCopy and paste:");
    console.log(`  cd ${cwd}`);
    return;
  }

  // Open new Ghostty window in existing instance via AppleScript
  await ensureGhosttyRunning(DEFAULT_TIMEOUT_MS);
  await newGhosttyWindowWithCwd(cwd, DEFAULT_TIMEOUT_MS);
  console.log("Opened new window");
}

function usage() {
  console.log(`Usage: ghostty-sessions.ts <command> [file]

Commands:
  show           Show current Ghostty sessions
  save [file]    Save current sessions to JSON (default: ${DEFAULT_FILE})
  restore [file] Select and restore a session from JSON

Examples:
  ghostty-sessions.ts show
  ghostty-sessions.ts save
  ghostty-sessions.ts restore`);
}

// Main
const [cmd, file] = Bun.argv.slice(2);
const targetFile = file || DEFAULT_FILE;

switch (cmd) {
  case "show":
    await cmdShow();
    break;
  case "save":
    await cmdSave(targetFile);
    break;
  case "restore":
    await cmdRestore(targetFile);
    break;
  case "-h":
  case "--help":
  case "help":
    usage();
    break;
  default:
    usage();
    process.exit(1);
}

// --- Ghostty AppleScript helpers (from ghostty-new.ts) ---

// Ensure Ghostty is running, then create new window with cwd
async function ensureGhosttyRunning(timeoutMs: number): Promise<number> {
  const script = `
    const app = Application("${APP_NAME}");
    app.launch();

    const systemEvents = Application("System Events");
    const process = systemEvents.applicationProcesses.byName("${APP_NAME}");
    while (!process.exists()) {
      delay(0.1);
    }
    process.windows().length;
  `;

  const rawCount = await runOsaScriptStdout(script, timeoutMs);
  const windowCount = Number.parseInt(rawCount, 10);
  if (!Number.isFinite(windowCount) || windowCount < 0) {
    throw new Error("Unable to find/launch Ghostty");
  }
  return windowCount;
}

// Create a new window in the existing Ghostty instance
async function newGhosttyWindowWithCwd(
  cwd: string,
  timeoutMs: number,
): Promise<void> {
  const cwdLiteral = JSON.stringify(cwd);
  const script = `
    const app = Application("${APP_NAME}");
    const config = app.newSurfaceConfiguration();
    config.initialWorkingDirectory = ${cwdLiteral};
    app.newWindow({ withConfiguration: config });
  `;
  await runOsaScriptStdout(script, timeoutMs);
}

// Run JXA script and return trimmed stdout
async function runOsaScriptStdout(
  script: string,
  timeoutMs: number,
): Promise<string> {
  const { exitCode, stdout, stderr } = Bun.spawnSync({
    cmd: ["osascript", "-l", "JavaScript", "-e", script],
    stdout: "pipe",
    stderr: "pipe",
    timeout: timeoutMs,
  });

  if (exitCode !== 0) {
    throw new Error("Unable to find/launch Ghostty");
  }

  const stderrText = stderr.toString().trim();
  if (stderrText) {
    throw new Error("Unable to find/launch Ghostty");
  }

  return stdout.toString().trim();
}
