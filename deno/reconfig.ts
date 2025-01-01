import yargs from "yargs";
import { loadConfig } from "./config/config.ts";

interface Options {
  config: string;
  verbose: boolean;
}

export async function main(args: string[]): Promise<void> {
  const argv = await yargs(args)
    .scriptName("reconfig")
    .version("0.1.0")
    .usage("$0 [options]")
    .option("config", {
      alias: "c",
      type: "string",
      description: "Config file path",
      default: "config.yaml",
    })
    .option("verbose", {
      alias: "v",
      type: "boolean",
      description: "Enable verbose logging",
    })
    .strict()
    .help()
    .parseAsync();

  const { config, verbose } = argv as Options;
  console.log(`flags: ${JSON.stringify({ config, verbose }, null, 2)}`);
  const configuration = await loadConfig(config);
  console.log(configuration);
}

if (import.meta.main) {
  main(Deno.args).catch((error) => {
    console.error(error);
    Deno.exit(1);
  });
}
