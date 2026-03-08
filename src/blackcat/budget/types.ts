export interface BudgetConfig {
  maxTokens?: number;
  maxCostUsd?: number;
}

export interface CostEntry {
  sessionId: string;
  tokens: number;
  costUsd: number;
  recordedAt: number;
}

export interface UsageSummary {
  totalTokens: number;
  totalCostUsd: number;
  entryCount: number;
}
