import type { BudgetConfig, CostEntry, UsageSummary } from "./types.js";

function now(): number {
  return Date.now();
}

const EMPTY_USAGE: UsageSummary = {
  totalTokens: 0,
  totalCostUsd: 0,
  entryCount: 0,
};

export class BudgetTracker {
  private readonly entries: Map<string, CostEntry[]> = new Map();

  record(sessionId: string, tokens: number, costUsd: number): void {
    if (tokens < 0 || costUsd < 0) {
      throw new Error("tokens and costUsd must be non-negative");
    }

    const entry: CostEntry = {
      sessionId,
      tokens,
      costUsd,
      recordedAt: now(),
    };

    const existing = this.entries.get(sessionId);
    if (existing) {
      existing.push(entry);
    } else {
      this.entries.set(sessionId, [entry]);
    }
  }

  getUsage(sessionId: string): UsageSummary {
    const entries = this.entries.get(sessionId);
    if (!entries || entries.length === 0) {
      return { ...EMPTY_USAGE };
    }

    let totalTokens = 0;
    let totalCostUsd = 0;

    for (const entry of entries) {
      totalTokens += entry.tokens;
      totalCostUsd += entry.costUsd;
    }

    return {
      totalTokens,
      totalCostUsd: Math.round(totalCostUsd * 1_000_000) / 1_000_000, // avoid floating-point drift
      entryCount: entries.length,
    };
  }

  isOverBudget(sessionId: string, config: BudgetConfig): boolean {
    const usage = this.getUsage(sessionId);

    if (config.maxTokens !== undefined && usage.totalTokens > config.maxTokens) {
      return true;
    }

    if (config.maxCostUsd !== undefined && usage.totalCostUsd > config.maxCostUsd) {
      return true;
    }

    return false;
  }

  reset(sessionId: string): void {
    this.entries.delete(sessionId);
  }

  getAllSessions(): string[] {
    return [...this.entries.keys()];
  }
}
