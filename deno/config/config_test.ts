import { assertEquals, assertThrows } from "@std/assert";
import { parseConfig } from "./config.ts";

const validConfigs = [
  {
    name: "minimal config",
    yaml: `
homebrew:
  formulae:
    main: []
  casks: []
asdf: {}
npm: []`,
  },
  {
    name: "basic sorted config",
    yaml: `
homebrew:
  formulae:
    main:
      - git
      - go
  casks:
    - 1password
asdf:
  nodejs: ["lts"]
npm:
  - typescript`,
  },
  {
    name: "asdf versions can be in any order",
    yaml: `
homebrew:
  formulae:
    main: []
  casks: []
asdf:
  nodejs: ["20.0.0", "18.0.0", "lts"]
npm: []`,
  },
] as const;

const invalidConfigs = [
  {
    name: "missing required homebrew.casks field",
    yaml: `
homebrew:
  formulae:
    main: []
  # Missing casks field
asdf: {}
npm: []`,
    msgIncludes: "Required",
  },
  {
    name: "formulae value should be array not string",
    yaml: `
homebrew:
  formulae:
    main: "not-an-array"
  casks: []
asdf: {}
npm: []`,
    msgIncludes: "Expected array",
  },
  {
    name: "invalid asdf version format",
    yaml: `
homebrew:
  formulae:
    main: []
  casks: []
asdf:
  nodejs: ["invalid-version"]
npm: []`,
    msgIncludes: "Invalid",
  },
  {
    name: "invalid cask format",
    yaml: `
homebrew:
  formulae:
    main: []
  casks:
    - this/has/way/too/many/parts
asdf: {}
npm: []`,
    msgIncludes: "Invalid",
  },
  {
    name: "unsorted formulae",
    yaml: `
homebrew:
  formulae:
    main:
      - zsh
      - git
      - bash
  casks: []
asdf: {}
npm: []`,
    msgIncludes: "must be sorted",
  },
  {
    name: "unsorted casks",
    yaml: `
homebrew:
  formulae:
    main: []
  casks:
    - zoom
    - alfred
    - 1password
asdf: {}
npm: []`,
    msgIncludes: "must be sorted",
  },
  {
    name: "unsorted npm packages",
    yaml: `
homebrew:
  formulae:
    main: []
  casks: []
asdf: {}
npm:
  - zod
  - typescript
  - deno`,
    msgIncludes: "must be sorted",
  },
] as const;

Deno.test("parseConfig()", async (t) => {
  await t.step("handles valid configs", async (t) => {
    for (const { name, yaml } of validConfigs) {
      await t.step(name, () => {
        const config = parseConfig(yaml);
        // Basic structure validation
        assertEquals(typeof config.homebrew, "object");
        assertEquals(Array.isArray(config.npm), true);
      });
    }
  });

  await t.step("rejects invalid configs", async (t) => {
    for (const { name, yaml, msgIncludes } of invalidConfigs) {
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
});
