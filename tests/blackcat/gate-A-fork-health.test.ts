import { existsSync } from "node:fs";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");

describe("Gate A — Fork Health", () => {
  it("package.json has name 'blackcat'", async () => {
    const raw = await readFile(path.join(ROOT, "package.json"), "utf-8");
    const pkg = JSON.parse(raw) as { name: string; version: string };
    expect(pkg.name).toBe("blackcat");
  });

  it("package.json has a semver version", async () => {
    const raw = await readFile(path.join(ROOT, "package.json"), "utf-8");
    const pkg = JSON.parse(raw) as { version: string };
    expect(pkg.version).toMatch(/^\d+\.\d+\.\d+/);
  });

  it("blackcat.mjs entry point exists at project root", () => {
    expect(existsSync(path.join(ROOT, "blackcat.mjs"))).toBe(true);
  });

  it("package.json bin includes blackcat entry", async () => {
    const raw = await readFile(path.join(ROOT, "package.json"), "utf-8");
    const pkg = JSON.parse(raw) as { bin?: Record<string, string> };
    expect(pkg.bin).toBeDefined();
    expect(pkg.bin!.blackcat).toBe("blackcat.mjs");
  });
});
