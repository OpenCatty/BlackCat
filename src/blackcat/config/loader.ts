import { readFileSync } from "node:fs";
import JSON5 from "json5";
import { BlackCatSchema } from "./schema.js";
import type { BlackCatConfig } from "./types.js";

function parseEnvValue(value: string): string | number | boolean {
  const trimmed = value.trim();
  if (/^(true|false)$/i.test(trimmed)) {
    return trimmed.toLowerCase() === "true";
  }
  if (/^-?\d+(\.\d+)?$/.test(trimmed)) {
    return Number(trimmed);
  }
  return value;
}

function toCamelCase(segment: string): string {
  const lower = segment.toLowerCase();
  return lower.replace(/_([a-z0-9])/g, (_, letter: string) => letter.toUpperCase());
}

function setNested(target: Record<string, unknown>, path: string[], value: unknown): void {
  let cursor: Record<string, unknown> = target;
  for (let index = 0; index < path.length - 1; index += 1) {
    const key = path[index];
    if (!cursor[key] || typeof cursor[key] !== "object" || Array.isArray(cursor[key])) {
      cursor[key] = {};
    }
    cursor = cursor[key] as Record<string, unknown>;
  }
  cursor[path[path.length - 1]] = value;
}

export function loadConfig(path: string): BlackCatConfig {
  const raw = readFileSync(path, "utf-8");
  const parsed = JSON5.parse(raw);
  return BlackCatSchema.parse(parsed);
}

export function loadConfigFromEnv(base: BlackCatConfig): BlackCatConfig {
  const next = structuredClone(base) as Record<string, unknown>;

  const aliases: Record<string, string[]> = {
    BLACKCAT_LLM_PROVIDER: ["llm", "provider"],
    BLACKCAT_LLM_MODEL: ["llm", "model"],
    BLACKCAT_LLM_API_KEY: ["llm", "apiKey"],
    BLACKCAT_LLM_APIKEY: ["llm", "apiKey"],
    BLACKCAT_TELEGRAM_TOKEN: ["channels", "telegram", "token"],
    BLACKCAT_DISCORD_TOKEN: ["channels", "discord", "token"],
    BLACKCAT_CHANNELS_TELEGRAM_TOKEN: ["channels", "telegram", "token"],
    BLACKCAT_CHANNELS_DISCORD_TOKEN: ["channels", "discord", "token"],
  };

  for (const [envName, envValue] of Object.entries(process.env)) {
    if (envValue === undefined) {
      continue;
    }

    const aliasPath = aliases[envName];
    if (aliasPath) {
      setNested(next, aliasPath, parseEnvValue(envValue));
      continue;
    }

    if (!envName.startsWith("BLACKCAT_")) {
      continue;
    }

    const rawPath = envName.slice("BLACKCAT_".length);
    if (!rawPath) {
      continue;
    }

    const segments = rawPath.split("_").filter(Boolean).map(toCamelCase);
    if (segments.length < 2) {
      continue;
    }
    setNested(next, segments, parseEnvValue(envValue));
  }

  return BlackCatSchema.parse(next);
}
