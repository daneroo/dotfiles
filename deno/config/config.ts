import { parse } from "@std/yaml";
import { z } from "zod";

// Matches either "name" or "org/repo/name" pattern
const formulaPattern = /^([^/]+|[^/]+\/[^/]+\/[^/]+)$/;

// Matches "latest", "lts", or semantic version (X[.Y[.Z]])
const asdfVersionPattern = /^(latest|lts|\d+(\.\d+){0,2})$/;

// Helper to check if array is sorted
const isSorted = (arr: string[]) => {
  return arr.every((v, i) => i === 0 || arr[i - 1] <= v);
};

// Helper for sorted string arrays
const sortedStringArray = (pattern?: RegExp) => {
  const baseSchema = pattern ? z.string().regex(pattern) : z.string();
  return z.array(baseSchema).refine(isSorted, (arr) => ({
    message: `Array must be sorted, got: [${arr.join(", ")}]`,
  }));
};

export const ConfigSchema = z
  .object({
    homebrew: z
      .object({
        formulae: z.record(sortedStringArray(formulaPattern)),
        casks: sortedStringArray(formulaPattern),
      })
      .required(),
    asdf: z.record(z.array(z.string().regex(asdfVersionPattern))),
    npm: sortedStringArray(),
  })
  .required();

// Infer the type from the schema
export type Config = z.infer<typeof ConfigSchema>;

// Parse YAML string to Config
export function parseConfig(yaml: string): Config {
  const data = parse(yaml);
  return ConfigSchema.parse(data);
}

// Load and parse config from file
export async function loadConfig(path: string): Promise<Config> {
  const text = await Deno.readTextFile(path);
  return parseConfig(text);
}
