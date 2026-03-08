import { describe, expect, it } from "vitest";
import type { RoleConfig } from "../../src/blackcat/config/types.js";
import { DEFAULT_ROLES } from "../../src/blackcat/config/defaults.js";
import { classifyMessage } from "../../src/blackcat/routing/router.js";
import { Supervisor } from "../../src/blackcat/routing/supervisor.js";

describe("Gate C — Router Classification Parity", () => {
  // Test each of the 7 roles using their keywords from defaults.ts
  const testCases: Array<{ input: string; expectedRole: string }> = [
    { input: "I need to deploy to docker", expectedRole: "phantom" },
    { input: "Check the crypto market for btc", expectedRole: "astrology" },
    { input: "Can you code a function in go?", expectedRole: "wizard" },
    { input: "Post this to social media", expectedRole: "artist" },
    { input: "Write a doc about this", expectedRole: "scribe" },
    { input: "Research and find information", expectedRole: "explorer" },
    { input: "Hello, how are you today?", expectedRole: "oracle" }, // fallback
  ];

  for (const { input, expectedRole } of testCases) {
    it(`classifies "${input}" → ${expectedRole}`, () => {
      const result = classifyMessage(input, DEFAULT_ROLES);
      expect(result.name).toBe(expectedRole);
    });
  }

  it("case-insensitive matching (uppercase input)", () => {
    const result = classifyMessage("DEPLOY THIS TO DOCKER NOW", DEFAULT_ROLES);
    expect(result.name).toBe("phantom");
  });

  it("priority ordering: deploy matches phantom(10) before wizard(30)", () => {
    // 'deploy' is a keyword for phantom (priority 10)
    // Even though 'build' or similar might match wizard, 'deploy' is phantom-only in defaults
    const result = classifyMessage("deploy this code", DEFAULT_ROLES);
    expect(result.name).toBe("phantom");
  });

  it("fallback to oracle when no keywords match", () => {
    const result = classifyMessage("tell me a joke about cats", DEFAULT_ROLES);
    expect(result.name).toBe("oracle");
    expect(result.priority).toBe(100);
  });

  it("oracle has empty keywords", () => {
    const oracle = DEFAULT_ROLES.find((r) => r.name === "oracle");
    expect(oracle).toBeDefined();
    expect(oracle!.keywords).toEqual([]);
  });

  it("all 7 default roles present", () => {
    expect(DEFAULT_ROLES).toHaveLength(7);
    const names = DEFAULT_ROLES.map((r) => r.name).sort();
    expect(names).toEqual(["artist", "astrology", "explorer", "oracle", "phantom", "scribe", "wizard"]);
  });

  it("uses empty array as fallback to DEFAULT_ROLES from router", () => {
    const result = classifyMessage("deploy now", []);
    // Router falls back to DEFAULT_ROLES from routing/roles.ts when empty array
    expect(result.name).toBe("phantom");
  });
});

describe("Gate C — Supervisor Route Overlays", () => {
  const customRoles: RoleConfig[] = [
    {
      name: "wizard",
      priority: 30,
      keywords: ["code"],
      systemPrompt: "You are a senior engineer.",
      model: "gpt-4o",
      provider: "openai",
      temperature: 0.2,
      allowedTools: ["read", "write"],
    },
    { name: "oracle", priority: 100, keywords: [] },
  ];

  it("route() returns correct SubAgentConfig for matched role", () => {
    const supervisor = new Supervisor(customRoles);
    const result = supervisor.route("help me code this");

    expect(result.role.name).toBe("wizard");
    expect(result.agentName).toBe("BlackCat (Wizard)");
    expect(result.systemPromptOverlay).toBe("You are a senior engineer.");
    expect(result.modelOverride).toBe("gpt-4o");
    expect(result.providerOverride).toBe("openai");
    expect(result.temperatureOverride).toBe(0.2);
    expect(result.allowedTools).toEqual(["read", "write"]);
  });

  it("route() returns undefined overlays for role without overrides", () => {
    const supervisor = new Supervisor(customRoles);
    const result = supervisor.route("tell me a joke");

    expect(result.role.name).toBe("oracle");
    expect(result.agentName).toBe("BlackCat (Oracle)");
    expect(result.systemPromptOverlay).toBeUndefined();
    expect(result.modelOverride).toBeUndefined();
    expect(result.providerOverride).toBeUndefined();
    expect(result.temperatureOverride).toBeUndefined();
    expect(result.allowedTools).toBeNull(); // null = all tools allowed
  });

  it("custom agent base name in agentName", () => {
    const supervisor = new Supervisor(customRoles, "MyBot");
    const result = supervisor.route("help me code");
    expect(result.agentName).toBe("MyBot (Wizard)");
  });
});
