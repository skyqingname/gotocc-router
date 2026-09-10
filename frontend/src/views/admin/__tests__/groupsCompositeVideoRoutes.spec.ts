import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

const currentDir = dirname(fileURLToPath(import.meta.url));
const groupsViewSource = readFileSync(
  resolve(currentDir, "../GroupsView.vue"),
  "utf8",
);
const typesSource = readFileSync(
  resolve(currentDir, "../../../types/index.ts"),
  "utf8",
);

describe("Unified video group contract", () => {
  it("does not introduce a Composite video route for the NewAPI-backed group", () => {
    expect(groupsViewSource).not.toContain('{ value: "videos", label:');
    expect(typesSource).not.toContain("| 'videos'");
  });
});
