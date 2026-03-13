#!/usr/bin/env bun
// ghostty-sessions - Save and restore Ghostty terminal sessions
// Requires: macOS, Ghostty 1.3+, bun, gum

import { $ } from "bun";
import { homedir } from "os";
import { basename } from "path";

const DEFAULT_FILE = `${homedir()}/.dotfiles/active-shells.json`;

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

  // Open new Ghostty window
  // -n spawns new instance (macOS limitation: no IPC to existing Ghostty for new window with args)
  const cwdBasename = basename(cwd);
  const args = [
    "open",
    "-na",
    "Ghostty.app",
    "--args",
    `--working-directory=${cwd}`,
  ];
  if (selectedTitle !== cwd && selectedTitle !== cwdBasename) {
    args.push(`--title=${selectedTitle}`);
  }
  Bun.spawnSync(args);
  console.log("Opened new window");
}

function usage() {
  console.log(`Usage: ghostty-sessions <command> [file]

Commands:
  show           Show current Ghostty sessions
  save [file]    Save current sessions to JSON (default: ${DEFAULT_FILE})
  restore [file] Select and restore a session from JSON

Examples:
  ghostty-sessions show
  ghostty-sessions save
  ghostty-sessions restore`);
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
