import { assertEquals, assertThrows } from "@std/assert";
import { parseConfig } from "./config_v2.ts";

// Test parsing our example configuration
Deno.test("parseConfig() v2 parses proposed-host-config.yaml", async () => {
  const yaml = await Deno.readTextFile(
    "config/testdata/proposed-host-config.yaml"
  );
  const config = parseConfig(yaml);

  // Verify basic structure
  assertEquals(Object.keys(config.hosts).sort(), [
    "davinci",
    "dirac",
    "galois",
    "shannon",
  ]);
  assertEquals(Object.keys(config.shared).sort(), [
    "base",
    "media-tools",
    "node-dev",
    "rust-dev",
  ]);

  // Verify a specific host configuration
  const galois = config.hosts.galois;
  assertEquals(galois.use, ["base", "node-dev", "rust-dev"]);
  assertEquals(galois.homebrew.formulae, ["deno"]);
  assertEquals(galois.homebrew.casks, ["docker", "visual-studio-code"]);
});

// Validation counter-examples
// Test generators can be abused, but this has allowed us to get better systematic coverage,
// while keeping our testing table readable, and the test output speaks for itself.
//  it also allows us to confirm the schema is correct, not just that it's constituents are valid.
// as JSON is valid YAML, we can write a JS Object literal, and stringify it to make valid yaml
const validationCounterExamples = [
  // BrewPattern validation
  ...allBrewPackageCases({
    namePrefix: "bad brewPackagePattern",
    packages: ["way/too/many/parts"],
    msgIncludes: "Invalid",
  }),
  // asdf Version validation
  ...allAsdfCases({
    namePrefix: "bad asdfVersionPattern",
    versions: ["not-a-version"],
    msgIncludes: "Invalid",
  }),
  ...allAsdfCases({
    namePrefix: "bad asdfVersionPattern",
    versions: ["1.2.3.4"],
    msgIncludes: "Invalid",
  }),
  // Sorting validation
  ...allBrewPackageCases({
    namePrefix: "unsorted",
    packages: ["zzz", "aaa"],
    msgIncludes: "must be sorted",
  }),
  ...allBrewPackageCases({
    namePrefix: "unsorted same basename",
    packages: ["org2/repo2/name", "org1/repo1/name"],
    msgIncludes: "must be sorted",
  }),
  ...allBrewPackageCases({
    namePrefix: "unsorted by basename",
    packages: ["bbb", "org/repo/aaa"],
    msgIncludes: "must be sorted",
  }),
  ...allNpmCases({
    namePrefix: "unsorted npm packages",
    packages: ["zzz", "aaa"],
    msgIncludes: "must be sorted",
  }),
  // Extra properties validation
  {
    name: "extra property at root level",
    yaml: JSON.stringify({
      extra: "should not be here",
    }),
    msgIncludes: "Unrecognized key",
  },
  {
    name: "extra property in hosts section",
    yaml: JSON.stringify({
      hosts: {
        galois: {
          extra: "should not be here",
        },
      },
    }),
    msgIncludes: "Unrecognized key",
  },
  {
    name: "extra property in homebrew section",
    yaml: JSON.stringify({
      hosts: {
        galois: {
          homebrew: {
            extra: "should not be here",
          },
        },
      },
    }),
    msgIncludes: "Unrecognized key",
  },
  // Shared config validation
  {
    name: "reference to non-existent shared config",
    yaml: JSON.stringify({
      hosts: {
        galois: {
          use: ["missing-config"],
        },
      },
      shared: {},
    }),
    msgIncludes:
      'Host "galois" references non-existent shared config "missing-config"',
  },
  {
    name: "reference to multiple non-existent shared configs",
    yaml: JSON.stringify({
      hosts: {
        galois: {
          use: ["missing1", "missing2"],
        },
      },
      shared: {},
    }),
    msgIncludes:
      'Host "galois" references non-existent shared config "missing1", Host "galois" references non-existent shared config "missing2"',
  },
  // Duplicate validation
  ...allBrewPackageCases({
    namePrefix: "duplicate package",
    packages: ["same-package", "same-package"],
    msgIncludes: "must not contain duplicates",
  }),
  ...allAsdfCases({
    namePrefix: "duplicate version",
    versions: ["lts", "lts"],
    msgIncludes: "must not contain duplicates",
  }),
  ...allNpmCases({
    namePrefix: "duplicate package",
    packages: ["same-package", "same-package"],
    msgIncludes: "must not contain duplicates",
  }),
] as const;

// Test validation rules
Deno.test("validation rules v2 - by counter examples", async (t) => {
  for (const { name, yaml, msgIncludes } of validationCounterExamples) {
    await t.step(name, () => {
      assertThrows(
        () => parseConfig(yaml),
        Error,
        msgIncludes,
        `Config should be rejected: ${name}`
      );
    });
  }
});

// Helper to generate four test cases for brew package patterns
// {formulae,casks} x {hosts,shared}
function allBrewPackageCases(testCase: {
  namePrefix: string;
  packages: string[];
  msgIncludes: string;
}) {
  const packageTypes = ["formulae", "casks"];
  const locations = ["hosts", "shared"];

  return locations.flatMap((where) =>
    packageTypes.map((packageType) => ({
      name: `${
        testCase.namePrefix
      } in ${where} ${packageType} (${testCase.packages.join(", ")})`,
      yaml: JSON.stringify({
        [where]: {
          nameOfHostOrShared: {
            homebrew: {
              [packageType]: testCase.packages,
            },
          },
        },
      }),
      msgIncludes: testCase.msgIncludes,
    }))
  );
}

// Helper to generate 2 test cases for ASDF version pattern in host and shared
function allAsdfCases(testCase: {
  namePrefix: string;
  versions: string[];
  msgIncludes: string;
}) {
  const locations = ["hosts", "shared"];
  return locations.map((where) => ({
    name: `${testCase.namePrefix} in ${where} (${testCase.versions.join(
      ", "
    )})`,
    yaml: JSON.stringify({
      [where]: {
        nameOfHostOrShared: {
          asdf: {
            nodejs: testCase.versions,
          },
        },
      },
    }),
    msgIncludes: testCase.msgIncludes,
  }));
}

// Helper to generate 2 test cases for npm packages in host and shared
function allNpmCases(testCase: {
  namePrefix: string;
  packages: string[];
  msgIncludes: string;
}) {
  const locations = ["hosts", "shared"];
  return locations.map((where) => ({
    name: `${testCase.namePrefix} in ${where} (${testCase.packages.join(
      ", "
    )})`,
    yaml: JSON.stringify({
      [where]: {
        nameOfHostOrShared: {
          npm: testCase.packages,
        },
      },
    }),
    msgIncludes: testCase.msgIncludes,
  }));
}
