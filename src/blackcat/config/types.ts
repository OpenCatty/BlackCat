export interface RoleConfig {
  name: string;
  priority: number;
  keywords: string[];
  systemPrompt?: string;
  model?: string;
  provider?: string;
  temperature?: number;
  allowedTools?: string[] | null;
}

export interface LLMConfig {
  provider: string;
  model: string;
  apiKey?: string;
  baseURL?: string;
  temperature?: number;
  maxTokens?: number;
  fallback?: string[];
}

export interface ChannelConfig {
  telegram?: { enabled: boolean; token: string };
  discord?: { enabled: boolean; token: string };
  whatsapp?: { enabled: boolean; allowFrom?: string[] };
}

export interface SecurityConfig {
  vaultPath?: string;
  denyPatterns?: string[];
  autoPermit?: boolean;
  guardrails?: {
    input?: boolean;
    tool?: boolean;
    output?: boolean;
  };
  hitl?: {
    enabled?: boolean;
    timeoutMinutes?: number;
  };
}

export interface MemoryConfig {
  filePath?: string;
  sqlitePath?: string;
  consolidationThreshold?: number;
  store?: "file" | "sqlite";
  embedding?: {
    provider?: string;
    model?: string;
    apiKey?: string;
    baseURL?: string;
  };
  coreMemory?: {
    maxEntries?: number;
    maxValueLen?: number;
  };
  maxArchival?: number;
}

export interface SessionConfig {
  enabled?: boolean;
  storeDir?: string;
  maxHistory?: number;
}

export interface AgentConfig {
  name?: string;
  greeting?: string;
  language?: string;
  tone?: string;
  ackMessage?: string;
}

export interface BudgetConfig {
  enabled?: boolean;
  dailyLimitUSD?: number;
  monthlyLimitUSD?: number;
  warnThreshold?: number;
}

export interface ProviderConfig {
  enabled?: boolean;
  model?: string;
  apiKey?: string;
  baseURL?: string;
}

export interface ProvidersConfig {
  openai?: ProviderConfig;
  anthropic?: ProviderConfig;
  copilot?: ProviderConfig;
  gemini?: ProviderConfig;
  ollama?: ProviderConfig & { baseURL: string };
  openrouter?: ProviderConfig;
}

export interface DashboardConfig {
  enabled?: boolean;
  addr?: string;
  token?: string;
}

export interface BlackCatConfig {
  llm: LLMConfig;
  channels?: ChannelConfig;
  roles?: RoleConfig[];
  security?: SecurityConfig;
  memory?: MemoryConfig;
  session?: SessionConfig;
  agent?: AgentConfig;
  budget?: BudgetConfig;
  providers?: ProvidersConfig;
  dashboard?: DashboardConfig;
  mcp?: {
    servers?: Array<{ name: string; command: string; args?: string[]; env?: Record<string, string> }>;
  };
  skills?: { dir?: string; maxSkillsInPrompt?: number; maxSkillFileBytes?: number };
  logging?: { level?: string; format?: string };
  rateLimit?: { enabled?: boolean; maxRequests?: number; windowSeconds?: number };
}
