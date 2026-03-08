export interface RoleConfig {
  name: string;
  priority: number;
  keywords: string[];
  agentId: string;
}

export const DEFAULT_ROLES: RoleConfig[] = [
  {
    name: "phantom",
    priority: 10,
    agentId: "blackcat-phantom",
    keywords: [
      "restart", "deploy", "server", "status", "docker", "systemctl", "health",
      "infra", "devops", "service", "nginx", "ssl", "ssh", "vpn", "firewall",
      "kubernetes", "k8s",
    ],
  },
  {
    name: "astrology",
    priority: 20,
    agentId: "blackcat-astrology",
    keywords: [
      "crypto", "bitcoin", "btc", "eth", "ethereum", "trading", "token", "defi",
      "nft", "wallet", "market", "portfolio", "investment", "stock", "forex",
      "chart", "candlestick", "pump", "whale", "altcoin", "blockchain", "web3",
    ],
  },
  {
    name: "wizard",
    priority: 30,
    agentId: "blackcat-wizard",
    keywords: [
      "code", "implement", "function", "bug", "fix", "test", "build", "compile",
      "git", "opencode", "typescript", "golang", "python", "javascript", "refactor",
      "debug", "api", "endpoint", "database", "sql", "migration", "error",
      "exception", "crash",
    ],
  },
  {
    name: "artist",
    priority: 40,
    agentId: "blackcat-artist",
    keywords: [
      "instagram", "tiktok", "twitter", "linkedin", "facebook", "threads", "post",
      "caption", "hashtag", "reel", "story", "content", "social", "viral",
      "engagement", "schedule", "publish", "influencer", "brand", "creative",
    ],
  },
  {
    name: "scribe",
    priority: 50,
    agentId: "blackcat-scribe",
    keywords: [
      "write", "draft", "article", "blog", "email", "document", "copy",
      "copywriting", "proofread", "translate", "summarize", "report", "newsletter",
      "pitch", "proposal", "readme", "documentation", "essay",
    ],
  },
  {
    name: "explorer",
    priority: 60,
    agentId: "blackcat-explorer",
    keywords: [
      "search", "find", "look up", "what is", "explain", "research", "summarize",
      "web", "browse", "read", "compare", "analyze", "review", "investigate",
      "information", "news", "latest",
    ],
  },
  {
    name: "oracle",
    priority: 100,
    agentId: "blackcat-oracle",
    keywords: [],
  },
];

/**
 * Classify an incoming message text against role keyword lists.
 * Returns the first matching role sorted by priority (lower = higher precedence),
 * or the fallback role (highest priority number, typically oracle).
 */
export function classifyMessage(
  text: string,
  roles: RoleConfig[] = DEFAULT_ROLES,
): RoleConfig {
  const lower = text.toLowerCase();
  const sorted = [...roles].sort((a, b) => a.priority - b.priority);
  for (const role of sorted) {
    if (role.keywords.length === 0) continue;
    if (role.keywords.some((kw) => lower.includes(kw))) return role;
  }
  // Fallback: highest priority number (last after sort) — typically oracle
  return sorted[sorted.length - 1];
}
