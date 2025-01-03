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

// Package manager configurations
const PackageConfig = z
  .object({
    homebrew: z
      .object({
        formulae: sortedUniqueStringArray(brewPackagePattern).default([]),
        casks: sortedUniqueStringArray(brewPackagePattern).default([]),
      })
      .strict()
      .default({}),
    asdf: z.record(uniqueStringArray(asdfVersionPattern)).default({}),
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
    hosts: z
      .record(
        z
          .string()
          .regex(
            identifierPattern,
            "Invalid identifier: must start with a letter and contain only letters, numbers, underscore, hyphen"
          ),
        HostConfig
      )
      .default({}),
    shared: z
      .record(
        z
          .string()
          .regex(
            identifierPattern,
            "Invalid identifier: must start with a letter and contain only letters, numbers, underscore, hyphen"
          ),
        PackageConfig
      )
      .default({}),
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

// Below are helper functions in top-down calling order
// Each function is defined after the functions it calls

// Helper for sorted unique string arrays
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

// Helper for unique string arrays (no sorting)
function uniqueStringArray(pattern?: RegExp) {
  const baseSchema = pattern ? z.string().regex(pattern) : z.string();
  return z.array(baseSchema).refine(uniqueArray, (arr) => ({
    message: `Array must not contain duplicates, got: [${arr.join(", ")}]`,
  }));
}

// Base unique array refinement
function uniqueArray<T>(arr: T[]): boolean {
  return arr.every((v, i) => arr.indexOf(v) === i);
}

// Check if array is sorted
function isSorted(arr: string[]): boolean {
  return arr.every((v, i) => i === 0 || compareByBasename(arr[i - 1], v));
}

// Compare strings by basename
function compareByBasename(i: string, j: string): boolean {
  const iBase = i.split("/").pop() ?? i;
  const jBase = j.split("/").pop() ?? j;
  if (iBase === jBase) {
    return i < j; // Use full path as tiebreaker
  }
  return iBase < jBase;
}
