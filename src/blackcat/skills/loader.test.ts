import { mkdirSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { SkillsLoader } from "./loader.js";

const tempRoots: string[] = [];

function makeTempDir(): string {
  const dir = join(tmpdir(), `blackcat-skills-${Math.random().toString(36).slice(2)}`);
  mkdirSync(dir, { recursive: true });
  tempRoots.push(dir);
  return dir;
}

function writeSkill(dir: string, fileName: string, content: string): string {
  const filePath = join(dir, fileName);
  writeFileSync(filePath, content, "utf8");
  return filePath;
}

afterEach(() => {
  while (tempRoots.length > 0) {
    const dir = tempRoots.pop();
    if (dir) {
      rmSync(dir, { recursive: true, force: true });
    }
  }
});

describe("SkillsLoader", () => {
  it("loads valid local markdown skills", () => {
    const dir = makeTempDir();
    writeSkill(
      dir,
      "alpha.md",
      "---\nname: alpha\nversion: 1.0.0\ndescription: Alpha\n---\n# Alpha\nUse alpha.",
    );
    writeSkill(dir, "beta.md", "---\nname: beta\n---\n# Beta\nUse beta.");

    const loader = new SkillsLoader({ dir, maxSkillsInPrompt: 10, maxSkillFileBytes: 50_000 });
    const loaded = loader.load();

    expect(loaded).toHaveLength(2);
    expect(loaded.map((skill) => skill.name)).toEqual(["alpha", "beta"]);
  });

  it("skips file over maxSkillFileBytes", () => {
    const dir = makeTempDir();
    writeSkill(dir, "small.md", "---\nname: small\n---\nsmall");
    writeSkill(dir, "large.md", `---\nname: large\n---\n${"x".repeat(500)}`);

    const loader = new SkillsLoader({ dir, maxSkillFileBytes: 80, maxSkillsInPrompt: 10 });
    const loaded = loader.load();

    expect(loaded).toHaveLength(1);
    expect(loaded[0]?.name).toBe("small");
  });

  it("skips file without frontmatter name", () => {
    const dir = makeTempDir();
    writeSkill(dir, "invalid.md", "---\ndescription: no name\n---\n# Invalid");
    writeSkill(dir, "valid.md", "---\nname: valid\n---\n# Valid");

    const loader = new SkillsLoader({ dir });
    const loaded = loader.load();

    expect(loaded).toHaveLength(1);
    expect(loaded[0]?.name).toBe("valid");
  });

  it("getSkillsForPrompt honors custom limit", () => {
    const dir = makeTempDir();
    writeSkill(dir, "a.md", "---\nname: a\n---\na");
    writeSkill(dir, "b.md", "---\nname: b\n---\nb");
    writeSkill(dir, "c.md", "---\nname: c\n---\nc");

    const loader = new SkillsLoader({ dir, maxSkillsInPrompt: 10 });
    loader.load();

    expect(loader.getSkillsForPrompt(2).map((skill) => skill.name)).toEqual(["a", "b"]);
  });

  it("returns empty list for empty dir", () => {
    const dir = makeTempDir();
    const loader = new SkillsLoader({ dir });

    expect(loader.load()).toEqual([]);
    expect(loader.getSkillsForPrompt()).toEqual([]);
  });
});
