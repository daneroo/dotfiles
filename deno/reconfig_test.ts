import { assertEquals } from "@std/assert";
import { main } from "./reconfig.ts";

// TODO(daneroo)Just a place holder for later actual integration testing
Deno.test("CLI --help shows usage", async () => {
  const originalConsoleLog = console.log;
  let output = "";

  console.log = (msg: string) => {
    output += msg + "\n";
  };

  try {
    await main(["--help"]);
  } catch (_error) {
    // yargs exits with error on --help
    assertEquals(output.includes("reconfig [options]"), true);
    assertEquals(output.includes("--config"), true);
    assertEquals(output.includes("--verbose"), true);
  } finally {
    console.log = originalConsoleLog;
  }
});
