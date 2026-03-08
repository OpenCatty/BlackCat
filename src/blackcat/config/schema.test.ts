import { describe, expect, it, vi } from "vitest";
import { DEFAULT_ROLES } from "./defaults.js";
import { loadConfigFromEnv } from "./loader.js";
import { BlackCatSchema } from "./schema.js";

describe("BlackCatSchema", () => {
  it("parses valid minimal config", () => {
    const parsed = BlackCatSchema.parse({
      llm: {
        provider: "openai",
        model: "gpt-5.2",
      },
    });

    expect(parsed.llm.provider).toBe("openai");
    expect(parsed.llm.model).toBe("gpt-5.2");
    expect(parsed.roles).toHaveLength(7);
  });

  it("parses valid config with 7 default roles", () => {
    const parsed = BlackCatSchema.parse({
      llm: {
        provider: "anthropic",
        model: "claude-sonnet-4-6",
      },
      roles: DEFAULT_ROLES,
    });

    expect(parsed.roles).toHaveLength(7);
    expect(parsed.roles?.at(-1)?.keywords).toEqual([]);
  });

  it("rejects duplicate role names", () => {
    expect(() =>
      BlackCatSchema.parse({
        llm: {
          provider: "openai",
          model: "gpt-5.2",
        },
        roles: [
          { name: "wizard", priority: 30, keywords: ["code"] },
          { name: "wizard", priority: 31, keywords: [] },
        ],
      }),
    ).toThrowError(/duplicate role name/i);
  });

  it("rejects roles without fallback role", () => {
    expect(() =>
      BlackCatSchema.parse({
        llm: {
          provider: "openai",
          model: "gpt-5.2",
        },
        roles: [{ name: "wizard", priority: 30, keywords: ["code"] }],
      }),
    ).toThrowError(/fallback role/i);
  });

  it("rejects config when llm.provider is missing", () => {
    expect(() =>
      BlackCatSchema.parse({
        llm: {
          model: "gpt-5.2",
        },
      }),
    ).toThrowError();
  });
});

describe("loadConfigFromEnv", () => {
  it("overrides llm provider from BLACKCAT_LLM_PROVIDER", () => {
    const base = BlackCatSchema.parse({
      llm: {
        provider: "openai",
        model: "gpt-5.2",
      },
    });

    vi.stubEnv("BLACKCAT_LLM_PROVIDER", "gemini");
    const merged = loadConfigFromEnv(base);

    expect(merged.llm.provider).toBe("gemini");
  });
});
