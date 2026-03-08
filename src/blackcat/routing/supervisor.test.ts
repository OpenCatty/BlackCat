import { describe, expect, it } from "vitest";

import type { RoleConfig } from "../config/types.js";
import { DEFAULT_ROLES } from "./roles.js";
import { Supervisor } from "./supervisor.js";

describe("Supervisor.route overlays", () => {
  it("returns classified role and agent name suffix", () => {
    const supervisor = new Supervisor(DEFAULT_ROLES, "BlackCat");
    const cfg = supervisor.route("please deploy server update");

    expect(cfg.role.name).toBe("phantom");
    expect(cfg.agentName).toBe("BlackCat (Phantom)");
    expect(cfg.allowedTools).toBeNull();
  });

  it("applies role overlays for prompt/model/provider/temperature/tools", () => {
    const roles: RoleConfig[] = [
      {
        name: "wizard",
        priority: 30,
        keywords: ["code"],
        systemPrompt: "You are a coding specialist",
        model: "gpt-5-mini",
        provider: "openai",
        temperature: 0.2,
        allowedTools: ["read", "edit"],
      },
      { name: "oracle", priority: 100, keywords: [] },
    ];

    const supervisor = new Supervisor(roles, "Agent");
    const cfg = supervisor.route("please code this feature");

    expect(cfg.role.name).toBe("wizard");
    expect(cfg.agentName).toBe("Agent (Wizard)");
    expect(cfg.systemPromptOverlay).toBe("You are a coding specialist");
    expect(cfg.modelOverride).toBe("gpt-5-mini");
    expect(cfg.providerOverride).toBe("openai");
    expect(cfg.temperatureOverride).toBe(0.2);
    expect(cfg.allowedTools).toEqual(["read", "edit"]);
  });

  it("maps empty allowedTools to empty array and missing to null", () => {
    const roles: RoleConfig[] = [
      { name: "scribe", priority: 50, keywords: ["write"], allowedTools: [] },
      { name: "oracle", priority: 100, keywords: [] },
    ];

    const supervisor = new Supervisor(roles);
    const scribeCfg = supervisor.route("write docs");
    const fallbackCfg = supervisor.route("unmatched sentence");

    expect(scribeCfg.allowedTools).toEqual([]);
    expect(fallbackCfg.allowedTools).toBeNull();
  });
});
