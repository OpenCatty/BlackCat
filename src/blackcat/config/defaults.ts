import type { RoleConfig } from "./types.js";

export const DEFAULT_ROLES: RoleConfig[] = [
  {
    name: "phantom",
    priority: 10,
    keywords: ["infra", "deploy", "docker", "k8s", "server", "host", "systemd"],
  },
  {
    name: "astrology",
    priority: 20,
    keywords: ["crypto", "blockchain", "web3", "eth", "btc", "wallet"],
  },
  {
    name: "wizard",
    priority: 30,
    keywords: ["code", "coding", "refactor", "debug", "build", "test", "go", "rust"],
  },
  {
    name: "artist",
    priority: 40,
    keywords: ["social", "post", "tweet", "thread", "image", "media"],
  },
  {
    name: "scribe",
    priority: 50,
    keywords: ["write", "doc", "readme", "blog", "article", "copy"],
  },
  {
    name: "explorer",
    priority: 60,
    keywords: ["research", "search", "find", "analyze", "investigate"],
  },
  { name: "oracle", priority: 100, keywords: [] },
];
