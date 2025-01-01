import { parse } from "@std/yaml";
import { z } from "zod";

// Matches either "name" or "org/repo/name" pattern
const brewPackagePattern = /^([^/]+|[^/]+\/[^/]+\/[^/]+)$/;

// Matches "latest", "lts", or semantic version (X[.Y[.Z]])
const asdfVersionPattern = /^(latest|lts|\d+(\.\d+){0,2})$/;

// Helper to check if array is sorted
const compareByBasename = (i: string, j: string): boolean => {
  const iBase = i.split("/").pop() ?? i;
  const jBase = j.split("/").pop() ?? j;
  if (iBase === jBase) {
    return i < j; // Use full path as tiebreaker
  }
  return iBase < jBase;
};

const isSorted = (arr: string[]) => {
  return arr.every((v, i) => i === 0 || compareByBasename(arr[i - 1], v));
};

// Helper for sorted string arrays
const sortedStringArray = (pattern?: RegExp) => {
  const baseSchema = pattern ? z.string().regex(pattern) : z.string();
  return z.array(baseSchema).refine(isSorted, (arr) => ({
    message: `Array must be sorted, got: [${arr.join(", ")}]`,
  }));
};

// Package manager configurations
const PackageConfig = z
  .object({
    homebrew: z
      .object({
        formulae: sortedStringArray(brewPackagePattern).default([]),
        casks: sortedStringArray(brewPackagePattern).default([]),
      })
      .strict()
      .default({}),
    asdf: z.record(z.array(z.string().regex(asdfVersionPattern))).default({}),
    npm: sortedStringArray().default([]),
  })
  .strict();

// Host configuration with 'use' directive
const HostConfig = PackageConfig.extend({
  use: z.array(z.string()).default([]),
}).strict();

// Complete configuration schema
export const ConfigSchema = z
  .object({
    hosts: z.record(HostConfig).default({}),
    shared: z.record(PackageConfig).default({}),
  })
  .strict()
  .superRefine((config, ctx) => {
    // Check that all referenced shared configs exist
    for (const [hostName, host] of Object.entries(config.hosts)) {
      for (const sharedName of host.use) {
        if (!config.shared[sharedName]) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: `Host "${hostName}" references non-existent shared config "${sharedName}"`,
            path: ["hosts", hostName, "use"],
          });
        }
      }
    }
  });

// Infer the type from the schema
export type Config = z.infer<typeof ConfigSchema>;

// Parse YAML string to Config
export function parseConfig(yaml: string): Config {
  const data = parse(yaml);
  try {
    return ConfigSchema.parse(data);
  } catch (e) {
    if (e instanceof z.ZodError) {
      // Format the error messages
      const messages = e.errors.map((err) => err.message);
      throw new Error(messages.join(", "));
    }
    throw e;
  }
}

// Load and parse config from file
export async function loadConfig(path: string): Promise<Config> {
  const text = await Deno.readTextFile(path);
  return parseConfig(text);
}
