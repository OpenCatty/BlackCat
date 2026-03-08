import { z } from "zod";
import { DEFAULT_ROLES } from "./defaults.js";

const NonEmptyString = z.string().trim().min(1);

export const RoleSchema = z
  .object({
    name: NonEmptyString,
    priority: z.number().int(),
    keywords: z.array(z.string()),
    systemPrompt: z.string().optional(),
    model: z.string().optional(),
    provider: z.string().optional(),
    temperature: z.number().optional(),
    allowedTools: z.array(z.string()).nullable().optional(),
  })
  .strict();

export const LLMConfigSchema = z
  .object({
    provider: NonEmptyString,
    model: NonEmptyString,
    apiKey: z.string().optional(),
    baseURL: z.string().optional(),
    temperature: z.number().optional(),
    maxTokens: z.number().int().positive().optional(),
    fallback: z.array(NonEmptyString).optional(),
  })
  .strict();

const ChannelSchema = z.object({ enabled: z.boolean(), token: z.string() }).strict();
const WhatsAppChannelSchema = z
  .object({ enabled: z.boolean(), allowFrom: z.array(z.string()).optional() })
  .strict();

const SecuritySchema = z
  .object({
    vaultPath: z.string().optional(),
    denyPatterns: z.array(z.string()).optional(),
    autoPermit: z.boolean().optional(),
    guardrails: z
      .object({
        input: z.boolean().optional().default(true),
        tool: z.boolean().optional().default(true),
        output: z.boolean().optional().default(true),
      })
      .optional()
      .default({ input: true, tool: true, output: true }),
    hitl: z
      .object({
        enabled: z.boolean().optional(),
        timeoutMinutes: z.number().int().positive().optional().default(5),
      })
      .optional()
      .default({ timeoutMinutes: 5 }),
  })
  .strict();

const MemorySchema = z
  .object({
    filePath: z.string().optional(),
    sqlitePath: z.string().optional(),
    consolidationThreshold: z.number().int().positive().optional(),
    store: z.enum(["file", "sqlite"]).optional(),
    embedding: z
      .object({
        provider: z.string().optional(),
        model: z.string().optional(),
        apiKey: z.string().optional(),
        baseURL: z.string().optional(),
      })
      .strict()
      .optional(),
    coreMemory: z
      .object({
        maxEntries: z.number().int().positive().optional(),
        maxValueLen: z.number().int().positive().optional(),
      })
      .strict()
      .optional(),
    maxArchival: z.number().int().positive().optional(),
  })
  .strict();

const SessionSchema = z
  .object({
    enabled: z.boolean().optional(),
    storeDir: z.string().optional(),
    maxHistory: z.number().int().positive().optional(),
  })
  .strict();

const AgentSchema = z
  .object({
    name: z.string().optional(),
    greeting: z.string().optional(),
    language: z.string().optional(),
    tone: z.string().optional(),
    ackMessage: z.string().optional(),
  })
  .strict();

const BudgetSchema = z
  .object({
    enabled: z.boolean().optional(),
    dailyLimitUSD: z.number().nonnegative().optional(),
    monthlyLimitUSD: z.number().nonnegative().optional(),
    warnThreshold: z.number().min(0).max(1).optional(),
  })
  .strict();

const ProviderSchema = z
  .object({
    enabled: z.boolean().optional(),
    model: z.string().optional(),
    apiKey: z.string().optional(),
    baseURL: z.string().optional(),
  })
  .strict();

const ProvidersSchema = z
  .object({
    openai: ProviderSchema.optional(),
    anthropic: ProviderSchema.optional(),
    copilot: ProviderSchema.optional(),
    gemini: ProviderSchema.optional(),
    ollama: ProviderSchema.extend({ baseURL: NonEmptyString }).optional(),
    openrouter: ProviderSchema.optional(),
  })
  .strict();

const DashboardSchema = z
  .object({
    enabled: z.boolean().optional(),
    addr: z.string().optional(),
    token: z.string().optional(),
  })
  .strict();

const MCPServerSchema = z
  .object({
    name: NonEmptyString,
    command: NonEmptyString,
    args: z.array(z.string()).optional(),
    env: z.record(z.string(), z.string()).optional(),
  })
  .strict();

export const BlackCatSchema = z
  .object({
    llm: LLMConfigSchema,
    channels: z
      .object({
        telegram: ChannelSchema.optional(),
        discord: ChannelSchema.optional(),
        whatsapp: WhatsAppChannelSchema.optional(),
      })
      .strict()
      .optional(),
    roles: z.array(RoleSchema).optional().default(DEFAULT_ROLES),
    security: SecuritySchema.optional().default({
      guardrails: { input: true, tool: true, output: true },
      hitl: { timeoutMinutes: 5 },
    }),
    memory: MemorySchema.optional(),
    session: SessionSchema.optional(),
    agent: AgentSchema.optional(),
    budget: BudgetSchema.optional(),
    providers: ProvidersSchema.optional(),
    dashboard: DashboardSchema.optional(),
    mcp: z.object({ servers: z.array(MCPServerSchema).optional() }).strict().optional(),
    skills: z
      .object({
        dir: z.string().optional(),
        maxSkillsInPrompt: z.number().int().positive().optional(),
        maxSkillFileBytes: z.number().int().positive().optional(),
      })
      .strict()
      .optional(),
    logging: z.object({ level: z.string().optional(), format: z.string().optional() }).strict().optional(),
    rateLimit: z
      .object({
        enabled: z.boolean().optional(),
        maxRequests: z.number().int().positive().optional(),
        windowSeconds: z.number().int().positive().optional(),
      })
      .strict()
      .optional(),
  })
  .strict()
  .superRefine((cfg, ctx) => {
    const roles = cfg.roles ?? [];
    if (roles.length === 0) {
      ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["roles"], message: "roles cannot be empty" });
      return;
    }

    const seen = new Set<string>();
    for (let index = 0; index < roles.length; index += 1) {
      const name = roles[index]?.name?.toLowerCase();
      if (seen.has(name)) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ["roles", index, "name"],
          message: `duplicate role name: ${roles[index]?.name}`,
        });
      }
      seen.add(name);
    }

    const hasFallback = roles.some((role) => role.keywords.length === 0);
    if (!hasFallback) {
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: ["roles"],
        message: "roles must include fallback role with empty keywords",
      });
    }
  });

export type BlackCatConfig = z.infer<typeof BlackCatSchema>;
