import type { RoleConfig } from "../config/types.js";
import { classifyMessage } from "./router.js";

export interface SubAgentConfig {
  role: RoleConfig;
  // resolved overlays
  agentName: string;
  systemPromptOverlay?: string;
  modelOverride?: string;
  providerOverride?: string;
  temperatureOverride?: number;
  allowedTools: string[] | null; // null = all, [] = none
}

export class Supervisor {
  constructor(
    private readonly roles: RoleConfig[],
    private readonly agentBaseName: string = "BlackCat",
  ) {}

  /**
   * Classify message and return SubAgentConfig with role overlays applied.
   * Mirrors Go Supervisor.RouteWithCfg() overlay behavior.
   */
  route(message: string): SubAgentConfig {
    const role = classifyMessage(message, this.roles);
    return {
      role,
      agentName: `${this.agentBaseName} (${capitalize(role.name)})`,
      systemPromptOverlay: role.systemPrompt || undefined,
      modelOverride: role.model || undefined,
      providerOverride: role.provider || undefined,
      temperatureOverride: role.temperature || undefined,
      allowedTools: role.allowedTools ?? null,
    };
  }
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1);
}
