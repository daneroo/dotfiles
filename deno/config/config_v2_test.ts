import { assertEquals, assertThrows } from "@std/assert";
import { parseConfig } from "./config_v2.ts";

// Test parsing our example configuration
Deno.test("parseConfig() v2 parses proposed-host-config.yaml", async () => {
  const yaml = await Deno.readTextFile(
    "config/testdata/proposed-host-config.yaml"
  );
  const config = parseConfig(yaml);

  // Verify basic structure
  assertEquals(Object.keys(config.hosts ?? {}).sort(), [
    "davinci",
    "dirac",
    "galois",
    "shannon",
  ]);
  assertEquals(Object.keys(config.shared ?? {}).sort(), [
    "base",
    "media-tools",
    "node-dev",
    "rust-dev",
  ]);

  // Verify a specific host configuration
  const galois = config.hosts?.galois;
  assertEquals(galois?.use, ["base", "node-dev", "rust-dev"]);
  assertEquals(galois?.homebrew?.formulae, ["deno"]);
  assertEquals(galois?.homebrew?.casks, ["docker", "visual-studio-code"]);
});

// Validation counter-examples
// as JSON is valid YAML, we can write a JS Object literal, and stringify it to make valid yaml
const validationCounterExamples = [
  {
    name: "bad brewPackagePattern in host formula (way/too/many/parts)",
    yaml: JSON.stringify({
      hosts: {
        galois: {
          homebrew: {
            formulae: ["way/too/many/parts"],
          },
        },
      },
    }),
    msgIncludes: "Invalid",
  },
  {
    name: "bad brewPackagePattern in host cask (way/too/many/parts)",
    yaml: JSON.stringify({
      hosts: {
        galois: {
          homebrew: {
            casks: ["way/too/many/parts"],
          },
        },
      },
    }),
    msgIncludes: "Invalid",
  },
  {
    name: "bad brewPackagePattern in shared formula (way/too/many/parts)",
    yaml: JSON.stringify({
      shared: {
        base: {
          homebrew: {
            formulae: ["way/too/many/parts"],
          },
        },
      },
    }),
    msgIncludes: "Invalid",
  },
  {
    name: "bad brewPackagePattern in shared cask (way/too/many/parts)",
    yaml: JSON.stringify({
      shared: {
        base: {
          homebrew: {
            casks: ["way/too/many/parts"],
          },
        },
      },
    }),
    msgIncludes: "Invalid",
  },
  {
    name: "bad asdfVersionPattern in host (not-a-version)",
    yaml: JSON.stringify({
      hosts: {
        galois: {
          asdf: {
            nodejs: ["not-a-version"],
          },
        },
      },
    }),
    msgIncludes: "Invalid",
  },
  {
    name: "bad asdfVersionPattern in host (1.2.3.4)",
    yaml: JSON.stringify({
      hosts: {
        galois: {
          asdf: {
            nodejs: ["1.2.3.4"],
          },
        },
      },
    }),
    msgIncludes: "Invalid",
  },
  {
    name: "bad asdfVersionPattern in shared (not-a-version)",
    yaml: JSON.stringify({
      shared: {
        base: {
          asdf: {
            nodejs: ["not-a-version"],
          },
        },
      },
    }),
    msgIncludes: "Invalid",
  },
  {
    name: "bad asdfVersionPattern in shared (1.2.3.4)",
    yaml: JSON.stringify({
      shared: {
        base: {
          asdf: {
            nodejs: ["1.2.3.4"],
          },
        },
      },
    }),
    msgIncludes: "Invalid",
  },
  {
    name: "unsorted formulae in host (zzz, aaa)",
    yaml: JSON.stringify({
      hosts: {
        galois: {
          homebrew: {
            formulae: ["zzz", "aaa"],
          },
        },
      },
    }),
    msgIncludes: "must be sorted",
  },
  {
    name: "unsorted casks in host (zzz, aaa)",
    yaml: JSON.stringify({
      hosts: {
        galois: {
          homebrew: {
            casks: ["zzz", "aaa"],
          },
        },
      },
    }),
    msgIncludes: "must be sorted",
  },
  {
    name: "unsorted formulae in shared (zzz, aaa)",
    yaml: JSON.stringify({
      shared: {
        base: {
          homebrew: {
            formulae: ["zzz", "aaa"],
          },
        },
      },
    }),
    msgIncludes: "must be sorted",
  },
  {
    name: "unsorted casks in shared (zzz, aaa)",
    yaml: JSON.stringify({
      shared: {
        base: {
          homebrew: {
            casks: ["zzz", "aaa"],
          },
        },
      },
    }),
    msgIncludes: "must be sorted",
  },
  {
    name: "unsorted npm packages in host (zzz, aaa)",
    yaml: JSON.stringify({
      hosts: {
        galois: {
          npm: ["zzz", "aaa"],
        },
      },
    }),
    msgIncludes: "must be sorted",
  },
  {
    name: "unsorted npm packages in shared (zzz, aaa)",
    yaml: JSON.stringify({
      shared: {
        base: {
          npm: ["zzz", "aaa"],
        },
      },
    }),
    msgIncludes: "must be sorted",
  },
  {
    name: "unsorted formulae in host (same basename, full path tiebreaker)",
    yaml: JSON.stringify({
      hosts: {
        galois: {
          homebrew: {
            formulae: ["org2/repo2/name", "org1/repo1/name"],
          },
        },
      },
    }),
    msgIncludes: "must be sorted",
  },
  {
    name: "unsorted formulae in host (basename: bbb > aaa)",
    yaml: JSON.stringify({
      hosts: {
        galois: {
          homebrew: {
            formulae: ["bbb", "org/repo/aaa"],
          },
        },
      },
    }),
    msgIncludes: "must be sorted",
  },
] as const;

// Test validation rules
Deno.test("validation rules v2", async (t) => {
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
