import { mkdtemp, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { loadConfig, loadConfigFromEnv } from "../../src/blackcat/config/loader.js";
import { BlackCatSchema } from "../../src/blackcat/config/schema.js";

let tmpDir: string;

beforeEach(async () => {
  tmpDir = await mkdtemp(path.join(os.tmpdir(), "gate-b-"));
});

afterEach(async () => {
  await rm(tmpDir, { recursive: true, force: true });
});

const MINIMAL_CONFIG = `{
  // JSON5 — comments and trailing commas allowed
  llm: {
    provider: "openai",
    model: "gpt-4o",
  },
}`;

const FULL_CONFIG_WITH_ROLES = `{
  llm: { provider: "anthropic", model: "claude-3" },
  roles: [
    { name: "phantom",   priority: 10, keywords: ["infra", "deploy"] },
    { name: "astrology", priority: 20, keywords: ["crypto", "btc"] },
    { name: "wizard",    priority: 30, keywords: ["code", "debug"] },
    { name: "artist",    priority: 40, keywords: ["social", "tweet"] },
    { name: "scribe",    priority: 50, keywords: ["write", "doc"] },
    { name: "explorer",  priority: 60, keywords: ["search", "find"] },
    { name: "oracle",    priority: 100, keywords: [] },
  ],
}`;

describe("Gate B — Config Loading", () => {
  it("loads minimal JSON5 config with defaults", async () => {
    const cfgPath = path.join(tmpDir, "blackcat.json5");
    await writeFile(cfgPath, MINIMAL_CONFIG, "utf-8");

    const config = loadConfig(cfgPath);
    expect(config.llm.provider).toBe("openai");
    expect(config.llm.model).toBe("gpt-4o");
    // Default roles should be populated
    expect(config.roles).toBeDefined();
    expect(config.roles!.length).toBeGreaterThanOrEqual(7);
  });

  it("loads JSON5 config with all 7 roles", async () => {
    const cfgPath = path.join(tmpDir, "blackcat.json5");
    await writeFile(cfgPath, FULL_CONFIG_WITH_ROLES, "utf-8");

    const config = loadConfig(cfgPath);
    expect(config.roles).toHaveLength(7);
    const names = config.roles!.map((r) => r.name);
    expect(names).toEqual(["phantom", "astrology", "wizard", "artist", "scribe", "explorer", "oracle"]);
  });

  it("env overrides apply to base config", async () => {
    const cfgPath = path.join(tmpDir, "blackcat.json5");
    await writeFile(cfgPath, MINIMAL_CONFIG, "utf-8");

    const base = loadConfig(cfgPath);

    // Set env overrides
    const originalProvider = process.env.BLACKCAT_LLM_PROVIDER;
    const originalModel = process.env.BLACKCAT_LLM_MODEL;
    process.env.BLACKCAT_LLM_PROVIDER = "anthropic";
    process.env.BLACKCAT_LLM_MODEL = "claude-3-opus";

    try {
      const merged = loadConfigFromEnv(base);
      expect(merged.llm.provider).toBe("anthropic");
      expect(merged.llm.model).toBe("claude-3-opus");
    } finally {
      if (originalProvider === undefined) delete process.env.BLACKCAT_LLM_PROVIDER;
      else process.env.BLACKCAT_LLM_PROVIDER = originalProvider;
      if (originalModel === undefined) delete process.env.BLACKCAT_LLM_MODEL;
      else process.env.BLACKCAT_LLM_MODEL = originalModel;
    }
  });

  it("Zod validation error on bad schema (missing llm)", () => {
    expect(() => {
      BlackCatSchema.parse({});
    }).toThrow();
  });

  it("Zod validation error on duplicate role names", () => {
    expect(() => {
      BlackCatSchema.parse({
        llm: { provider: "openai", model: "gpt-4o" },
        roles: [
          { name: "wizard", priority: 10, keywords: ["code"] },
          { name: "wizard", priority: 20, keywords: ["debug"] },
          { name: "oracle", priority: 100, keywords: [] },
        ],
      });
    }).toThrow(/duplicate role name/i);
  });

  it("Zod validation error on roles without fallback (no empty keywords)", () => {
    expect(() => {
      BlackCatSchema.parse({
        llm: { provider: "openai", model: "gpt-4o" },
        roles: [
          { name: "wizard", priority: 10, keywords: ["code"] },
        ],
      });
    }).toThrow(/fallback/i);
  });
});
