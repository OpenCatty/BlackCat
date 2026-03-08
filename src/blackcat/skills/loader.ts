import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";

import { parseFrontmatterBlock } from "../../markdown/frontmatter.js";
import type { Skill, SkillFrontmatter } from "./types.js";

const DEFAULT_MAX_SKILLS_IN_PROMPT = 10;
const DEFAULT_MAX_SKILL_FILE_BYTES = 50_000;

function normalizeTags(raw: string | undefined): string[] | undefined {
  if (!raw) {
    return undefined;
  }

  try {
    const parsed = JSON.parse(raw) as unknown;
    if (Array.isArray(parsed)) {
      const tags = parsed
        .filter((item): item is string => typeof item === "string")
        .map((tag) => tag.trim())
        .filter(Boolean);
      return tags.length > 0 ? tags : undefined;
    }
  } catch {
    // Continue to comma-separated fallback.
  }

  const tags = raw
    .split(",")
    .map((tag) => tag.trim())
    .filter(Boolean);
  return tags.length > 0 ? tags : undefined;
}

function parseFrontmatter(content: string): SkillFrontmatter | null {
  const trimmed = content.replace(/\r\n/g, "\n").replace(/\r/g, "\n");
  if (!trimmed.startsWith("---\n")) {
    return null;
  }

  const parsed = parseFrontmatterBlock(trimmed);
  const name = parsed.name?.trim();
  if (!name) {
    return null;
  }

  return {
    name,
    version: parsed.version?.trim() || undefined,
    description: parsed.description?.trim() || undefined,
    tags: normalizeTags(parsed.tags),
  };
}

function extractBody(content: string): string {
  const normalized = content.replace(/\r\n/g, "\n").replace(/\r/g, "\n");
  if (!normalized.startsWith("---\n")) {
    return normalized;
  }
  const endIndex = normalized.indexOf("\n---", 4);
  if (endIndex < 0) {
    return normalized;
  }
  const bodyStart = endIndex + "\n---".length;
  return normalized.slice(bodyStart).replace(/^\n/, "").trim();
}

export class SkillsLoader {
  private readonly dir: string;
  private readonly maxSkillsInPrompt: number;
  private readonly maxSkillFileBytes: number;
  private loaded: Skill[] = [];

  constructor(config: { dir: string; maxSkillsInPrompt?: number; maxSkillFileBytes?: number }) {
    this.dir = config.dir;
    this.maxSkillsInPrompt = config.maxSkillsInPrompt ?? DEFAULT_MAX_SKILLS_IN_PROMPT;
    this.maxSkillFileBytes = config.maxSkillFileBytes ?? DEFAULT_MAX_SKILL_FILE_BYTES;
  }

  load(): Skill[] {
    let entries: string[];
    try {
      entries = readdirSync(this.dir);
    } catch {
      this.loaded = [];
      return this.loaded;
    }

    const skills: Skill[] = [];
    for (const entry of entries) {
      if (!entry.toLowerCase().endsWith(".md")) {
        continue;
      }

      const filePath = join(this.dir, entry);
      let stats;
      try {
        stats = statSync(filePath);
      } catch {
        continue;
      }

      if (!stats.isFile()) {
        continue;
      }

      if (stats.size > this.maxSkillFileBytes) {
        continue;
      }

      let content: string;
      try {
        content = readFileSync(filePath, "utf8");
      } catch {
        continue;
      }

      const frontmatter = parseFrontmatter(content);
      if (!frontmatter) {
        continue;
      }

      skills.push({
        name: frontmatter.name,
        content,
        body: extractBody(content),
        frontmatter,
        filePath,
        sizeBytes: stats.size,
      });
    }

    skills.sort((a, b) => a.name.localeCompare(b.name));
    this.loaded = skills.slice(0, this.maxSkillsInPrompt);
    return this.loaded;
  }

  getSkillsForPrompt(limit?: number): Skill[] {
    const effectiveLimit = limit ?? this.maxSkillsInPrompt;
    return this.loaded.slice(0, Math.max(0, effectiveLimit));
  }
}
