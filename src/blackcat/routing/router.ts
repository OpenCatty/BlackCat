import type { RoleConfig } from "../config/types.js";
import { DEFAULT_ROLES } from "./roles.js";

/**
 * Classify a message into a role using keyword-priority matching.
 * Semantics mirror Go's ClassifyMessage() in internal/agent/router.go:
 * - Case-insensitive substring match using String.toLowerCase() (NOT toLocaleLowerCase)
 * - Roles sorted ascending by priority, first match wins
 * - Fallback: role with highest priority number (oracle)
 */
export function classifyMessage(message: string, roles: RoleConfig[]): RoleConfig {
  const effectiveRoles = roles.length > 0 ? roles : DEFAULT_ROLES;
  const lower = message.toLowerCase();
  const sorted = [...effectiveRoles].sort((a, b) => a.priority - b.priority);

  for (const role of sorted) {
    for (const kw of role.keywords) {
      if (lower.includes(kw.toLowerCase())) {
        return role;
      }
    }
  }

  // fallback: role with highest priority number
  return sorted[sorted.length - 1]!;
}
