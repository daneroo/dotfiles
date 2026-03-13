#!/usr/bin/env bun

import { homedir } from "node:os";

const DEFAULT_CWD = `${homedir()}/Code`;
const DEFAULT_TIMEOUT_MS = 30_000;
const APP_NAME = "Ghostty";

// ENTRY POINT
if (import.meta.main) {
  try {
    await main();
  } catch (error) {
    console.error("Unable to find/launch Ghostty");
    process.exit(1);
  }
}

// MAIN
async function main(): Promise<void> {
  await newWindowOnSingleGhosttyInstance();
}

// THROWS: timeout/failure error if Ghostty cannot be ensured/opened.
async function newWindowOnSingleGhosttyInstance(
  cwd: string = DEFAULT_CWD,
  timeoutMs: number = DEFAULT_TIMEOUT_MS,
): Promise<void> {
  await ensureGhosttyRunning(timeoutMs);
  await newGhosttyWindowWithCwd(cwd, timeoutMs);
}

// For running-app path: reliably create a new window in the existing instance.
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

// THROWS: timeout/failure error if Ghostty does not become running before `timeoutMs`.
// RETURNS: current window count after running is guaranteed.
// Timeout is enforced by Bun timeout on the osascript process, so this JXA loop can poll forever.
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

// Runs JXA and returns trimmed stdout.
// Throws if the process times out, exits non-zero, or stderr is non-empty.
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
