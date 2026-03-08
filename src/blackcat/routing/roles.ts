import type { RoleConfig } from "../config/types.js";

export type RoleType = string;

export const ROLE_PHANTOM: RoleType = "phantom";
export const ROLE_ASTROLOGY: RoleType = "astrology";
export const ROLE_WIZARD: RoleType = "wizard";
export const ROLE_ARTIST: RoleType = "artist";
export const ROLE_SCRIBE: RoleType = "scribe";
export const ROLE_EXPLORER: RoleType = "explorer";
export const ROLE_ORACLE: RoleType = "oracle";

// Mirrors internal/agent/router.go defaultRoles.
export const DEFAULT_ROLES: RoleConfig[] = [
  {
    name: ROLE_PHANTOM,
    keywords: ["restart", "deploy", "server", "status", "docker", "systemctl", "health", "infra", "devops", "service", "nginx", "ssl"],
    priority: 10,
  },
  {
    name: ROLE_ASTROLOGY,
    keywords: [
      "crypto",
      "bitcoin",
      "btc",
      "eth",
      "ethereum",
      "trading",
      "token",
      "defi",
      "nft",
      "wallet",
      "market",
      "portfolio",
      "investment",
      "stock",
      "forex",
      "chart",
      "candlestick",
      "pump",
      "whale",
    ],
    priority: 20,
  },
  {
    name: ROLE_WIZARD,
    keywords: [
      "code",
      "implement",
      "function",
      "bug",
      "fix",
      "test",
      "build",
      "compile",
      "git",
      "deploy",
      "opencode",
      "typescript",
      "golang",
      "python",
      "javascript",
      "refactor",
      "debug",
      "api",
      "endpoint",
      "database",
      "sql",
      "migration",
    ],
    priority: 30,
  },
  {
    name: ROLE_ARTIST,
    keywords: [
      "instagram",
      "tiktok",
      "twitter",
      "tweet",
      "linkedin",
      "facebook",
      "threads",
      "caption",
      "hashtag",
      "reel",
      "story",
      "content",
      "social",
      "viral",
      "engagement",
      "schedule",
      "publish",
    ],
    priority: 40,
  },
  {
    name: ROLE_SCRIBE,
    keywords: [
      "write",
      "draft",
      "article",
      "blog",
      "email",
      "document",
      "copy",
      "copywriting",
      "proofread",
      "translate",
      "summarize",
      "report",
      "newsletter",
      "pitch",
      "proposal",
    ],
    priority: 50,
  },
  {
    name: ROLE_EXPLORER,
    keywords: ["search", "find", "look up", "what is", "explain", "research", "summarize", "web", "browse", "read", "compare", "analyze", "review", "investigate"],
    priority: 60,
  },
  {
    name: ROLE_ORACLE,
    keywords: [],
    priority: 100,
  },
];
