/**
 * Configuration Schema and Validation
 *
 * Example with validation rules:
 * {
 *   hosts: {
 *     // Host names must be identifiers: [a-zA-Z][a-zA-Z0-9_-]*
 *     galois: {
 *       // Referenced shared configs must exist
 *       use: ["base", "node-dev"],  // Sorted, no duplicates
 *       homebrew: {
 *         // All arrays are sorted by basename and contain no duplicates
 *         formulae: ["deno"],
 *         casks: ["docker", "visual-studio-code"]
 *       }
 *     }
 *   },
 *   shared: {
 *     // Shared names must be identifiers: [a-zA-Z][a-zA-Z0-9_-]*
 *     base: {
 *       homebrew: {
 *         formulae: ["git", "go"],  // Must match pattern: name or org/repo/name
 *         casks: ["1password"]
 *       }
 *     },
 *     "node-dev": {
 *       asdf: {
 *         nodejs: ["20.0.0", "lts"]  // "latest", "lts", or X[.Y[.Z]], order preserved
 *       },
 *       npm: ["ts-node", "typescript"]  // Sorted, no duplicates
 *     }
 *   }
 * }
 *
 * Note: All sections use strict validation (no extra properties)
 */

import { parse } from "@std/yaml";
import { z } from "zod";

// Matches either "name" or "org/repo/name" pattern
// Used in: homebrew.formulae, homebrew.casks
const brewPackagePattern = /^([^/]+|[^/]+\/[^/]+\/[^/]+)$/;

// Matches "latest", "lts", or semantic version (X[.Y[.Z]])
// Used in: asdf plugin versions
const asdfVersionPattern = /^(latest|lts|\d+(\.\d+){0,2})$/;

// Matches identifier-like names: letters, numbers, underscore, hyphen, but must start with a letter
// Used in: hosts and shared section names
const identifierPattern = /^[a-zA-Z][a-zA-Z0-9_-]*$/;

// Schema for section names (hosts and shared)
const IdentifierSchema = z
  .string()
  .regex(
    identifierPattern,
    "Invalid identifier: must start with a letter and contain only letters, numbers, underscore, hyphen"
  );

// Schema for homebrew packages
const HomebrewSchema = z
  .object({
    formulae: sortedUniqueStringArray(brewPackagePattern).default([]),
    casks: sortedUniqueStringArray(brewPackagePattern).default([]),
  })
  .strict();

// Schema for asdf versions
const AsdfSchema = z.record(uniqueStringArray(asdfVersionPattern)).default({});

// Package manager configurations: {homebrew, asdf, npm}
const PackageConfig = z
  .object({
    homebrew: HomebrewSchema.default({}),
    asdf: AsdfSchema,
    npm: sortedUniqueStringArray().default([]),
  })
  .strict();

// Host configuration with 'use' directive
const HostConfig = PackageConfig.extend({
  use: sortedUniqueStringArray().default([]),
}).strict();

// Complete configuration schema
export const ConfigSchema = z
  .object({
    hosts: z.record(IdentifierSchema, HostConfig).default({}),
    shared: z.record(IdentifierSchema, PackageConfig).default({}),
  })
  .strict()
  .superRefine((config, ctx) => {
    // Ensure all shared configs referenced in 'use' arrays exist
    validateSharedUseExistence(config, ctx);
    // Prevent ASDF plugin definitions in multiple places
    validateAsdfPluginsUniqueness(config, ctx);
  });

// Export types inferred from schemas
/** Complete configuration with hosts and shared sections */
export type Config = z.infer<typeof ConfigSchema>;

/** Core package configuration: homebrew, asdf, npm */
export type PackageConfig = z.infer<typeof PackageConfig>;

/** Host configuration extends PackageConfig with 'use' array referencing shared configs */
export type HostConfig = z.infer<typeof HostConfig>;

/** Homebrew configuration with sorted formulae and casks (org/repo/name pattern) */
export type HomebrewConfig = z.infer<typeof HomebrewSchema>;

/** ASDF configuration mapping plugin names to version arrays (order preserved) */
export type AsdfConfig = z.infer<typeof AsdfSchema>;

/** Valid identifier: must start with letter, allows letters, numbers, underscore, hyphen */
export type Identifier = z.infer<typeof IdentifierSchema>;

// Type aliases for configurations after merging shared dependencies
/** A host's complete package configuration after merging shared dependencies */
export type SingleHostConfig = PackageConfig; // No 'use' array

/** All hosts' package configurations after merging shared dependencies */
export type MultiHostConfig = Record<Identifier, PackageConfig>;

/** Load and parse config from file */
export async function loadConfig(path: string): Promise<Config> {
  const text = await Deno.readTextFile(path);
  return parseConfig(text);
}

/** Parse YAML string into a validated Config */
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

// Below are helper functions in top-down calling order
// Each function is defined after the functions it calls

/** Helper for sorted unique string arrays with optional pattern validation */
function sortedUniqueStringArray(pattern?: RegExp) {
  return uniqueStringArray(pattern).refine(isSorted, (arr) => {
    // Create a sorted copy to show the expected order
    const expected = [...arr].sort((a, b) =>
      compareByBasename(a, b) ? -1 : 1
    );
    // Find first out-of-order term
    const firstOutOfOrder = arr.find(
      (v, i) => i > 0 && !compareByBasename(arr[i - 1], v)
    );
    return {
      message: `Array must be sorted. Expected: [${expected.join(", ")}]${
        firstOutOfOrder ? ` (first out of order: ${firstOutOfOrder})` : ""
      }`,
    };
  });
}

/** Helper for unique string arrays with optional pattern validation */
function uniqueStringArray(pattern?: RegExp) {
  const baseSchema = pattern ? z.string().regex(pattern) : z.string();
  return z.array(baseSchema).refine(uniqueArray, (arr) => ({
    message: `Array must not contain duplicates, got: [${arr.join(", ")}]`,
  }));
}

/** Check if array has no duplicate elements */
function uniqueArray<T>(arr: T[]): boolean {
  return arr.every((v, i) => arr.indexOf(v) === i);
}

/** Check if array is sorted by basename */
function isSorted(arr: string[]): boolean {
  return arr.every((v, i) => i === 0 || compareByBasename(arr[i - 1], v));
}

/** Compare strings by basename, using full path as tiebreaker */
function compareByBasename(i: string, j: string): boolean {
  const iBase = i.split("/").pop() ?? i;
  const jBase = j.split("/").pop() ?? j;
  if (iBase === jBase) {
    return i < j; // Use full path as tiebreaker
  }
  return iBase < jBase;
}

/** Validate that each ASDF plugin appears in *at most one* of host or any shared config */
function validateAsdfPluginsUniqueness(config: Config, ctx: z.RefinementCtx) {
  // For each host, collect all sources of all plugins
  for (const [hostName, host] of Object.entries(config.hosts)) {
    // Track plugin -> [where defined], e.g.:
    // nodejs -> ["host:galois"]
    // python -> ["shared:base"]
    // deno -> ["shared:node-dev", "shared:base"]  // Error case
    const pluginSources = new Map<string, string[]>();

    // Collect plugins defined in host config
    for (const plugin of Object.keys(host.asdf ?? {})) {
      pluginSources.set(plugin, [`host:${hostName}`]);
    }

    // Collect plugins from all used shared configs
    for (const sharedName of host.use) {
      const shared = config.shared[sharedName];
      if (!shared) continue; // Skip if shared config doesn't exist
      for (const plugin of Object.keys(shared.asdf ?? {})) {
        const sources = pluginSources.get(plugin) ?? [];
        sources.push(`shared:${sharedName}`);
        pluginSources.set(plugin, sources);
      }
    }

    // Report any plugin with multiple sources
    for (const [plugin, sources] of pluginSources) {
      if (sources.length > 1) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: `ASDF plugin cannot be merged ${plugin}: defined in ${sources.join(
            ", "
          )}`,
          path: ["hosts", hostName, "use"],
        });
      }
    }
  }
}

/** Validate that all referenced shared configs exist */
function validateSharedUseExistence(config: Config, ctx: z.RefinementCtx) {
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
}
