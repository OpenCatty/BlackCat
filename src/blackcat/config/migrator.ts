import JSON5 from "json5";
import { parse as parseYaml } from "yaml";

type PlainObject = Record<string, unknown>;

function isPlainObject(value: unknown): value is PlainObject {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function normalizeRole(role: unknown): unknown {
  if (!isPlainObject(role)) {
    return role;
  }
  const next: PlainObject = { ...role };
  if ("system_prompt" in next && !("systemPrompt" in next)) {
    next.systemPrompt = next.system_prompt;
    delete next.system_prompt;
  }
  if ("allowed_tools" in next && !("allowedTools" in next)) {
    next.allowedTools = next.allowed_tools;
    delete next.allowed_tools;
  }
  return next;
}

export function migrateYamlToJson5(yamlContent: string): string {
  const parsed = parseYaml(yamlContent);
  if (!isPlainObject(parsed)) {
    return JSON5.stringify({}, null, 2);
  }

  const next: PlainObject = { ...parsed };

  if (isPlainObject(next.llm)) {
    const llm = { ...next.llm };
    if ("api_key" in llm && !("apiKey" in llm)) {
      llm.apiKey = llm.api_key;
      delete llm.api_key;
    }
    if ("base_url" in llm && !("baseURL" in llm)) {
      llm.baseURL = llm.base_url;
      delete llm.base_url;
    }
    if ("max_tokens" in llm && !("maxTokens" in llm)) {
      llm.maxTokens = llm.max_tokens;
      delete llm.max_tokens;
    }
    next.llm = llm;
  }

  if (Array.isArray(next.roles)) {
    next.roles = next.roles.map(normalizeRole);
  }

  return JSON5.stringify(next, null, 2);
}
