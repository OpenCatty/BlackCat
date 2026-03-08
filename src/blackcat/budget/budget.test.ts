import { describe, it, expect, beforeEach } from "vitest";

import { BudgetTracker } from "./tracker.js";
import type { BudgetConfig } from "./types.js";

describe("BudgetTracker", () => {
  let tracker: BudgetTracker;

  beforeEach(() => {
    tracker = new BudgetTracker();
  });

  it("record and getUsage returns correct totals", () => {
    tracker.record("session-1", 100, 0.01);
    tracker.record("session-1", 200, 0.02);

    const usage = tracker.getUsage("session-1");
    expect(usage.totalTokens).toBe(300);
    expect(usage.totalCostUsd).toBe(0.03);
    expect(usage.entryCount).toBe(2);
  });

  it("getUsage returns zeros for unknown session", () => {
    const usage = tracker.getUsage("nonexistent");
    expect(usage.totalTokens).toBe(0);
    expect(usage.totalCostUsd).toBe(0);
    expect(usage.entryCount).toBe(0);
  });

  it("per-session isolation", () => {
    tracker.record("s1", 100, 0.01);
    tracker.record("s2", 500, 0.05);

    const u1 = tracker.getUsage("s1");
    const u2 = tracker.getUsage("s2");

    expect(u1.totalTokens).toBe(100);
    expect(u2.totalTokens).toBe(500);
    expect(u1.totalCostUsd).toBe(0.01);
    expect(u2.totalCostUsd).toBe(0.05);
  });

  it("isOverBudget returns true when maxTokens exceeded", () => {
    tracker.record("s1", 1000, 0.1);

    const config: BudgetConfig = { maxTokens: 500 };
    expect(tracker.isOverBudget("s1", config)).toBe(true);
  });

  it("isOverBudget returns true when maxCostUsd exceeded", () => {
    tracker.record("s1", 100, 5.0);

    const config: BudgetConfig = { maxCostUsd: 1.0 };
    expect(tracker.isOverBudget("s1", config)).toBe(true);
  });

  it("isOverBudget returns false when within limits", () => {
    tracker.record("s1", 100, 0.01);

    const config: BudgetConfig = { maxTokens: 1000, maxCostUsd: 1.0 };
    expect(tracker.isOverBudget("s1", config)).toBe(false);
  });

  it("isOverBudget returns false for unknown session", () => {
    const config: BudgetConfig = { maxTokens: 100 };
    expect(tracker.isOverBudget("nope", config)).toBe(false);
  });

  it("reset clears session usage", () => {
    tracker.record("s1", 500, 0.5);
    tracker.reset("s1");

    const usage = tracker.getUsage("s1");
    expect(usage.totalTokens).toBe(0);
    expect(usage.totalCostUsd).toBe(0);
    expect(usage.entryCount).toBe(0);
  });

  it("reset does not affect other sessions", () => {
    tracker.record("s1", 100, 0.01);
    tracker.record("s2", 200, 0.02);
    tracker.reset("s1");

    expect(tracker.getUsage("s1").totalTokens).toBe(0);
    expect(tracker.getUsage("s2").totalTokens).toBe(200);
  });

  it("getAllSessions lists all tracked sessions", () => {
    tracker.record("alpha", 10, 0.001);
    tracker.record("beta", 20, 0.002);
    tracker.record("gamma", 30, 0.003);

    const sessions = tracker.getAllSessions();
    expect(sessions).toContain("alpha");
    expect(sessions).toContain("beta");
    expect(sessions).toContain("gamma");
    expect(sessions).toHaveLength(3);
  });

  it("getAllSessions excludes reset sessions", () => {
    tracker.record("a", 10, 0.001);
    tracker.record("b", 20, 0.002);
    tracker.reset("a");

    const sessions = tracker.getAllSessions();
    expect(sessions).not.toContain("a");
    expect(sessions).toContain("b");
  });

  it("rejects negative token values", () => {
    expect(() => tracker.record("s1", -100, 0.01)).toThrow("non-negative");
  });

  it("rejects negative cost values", () => {
    expect(() => tracker.record("s1", 100, -0.01)).toThrow("non-negative");
  });

  it("handles floating-point cost accumulation without drift", () => {
    // 0.1 + 0.2 classically produces 0.30000000000000004
    tracker.record("s1", 10, 0.1);
    tracker.record("s1", 10, 0.2);

    const usage = tracker.getUsage("s1");
    expect(usage.totalCostUsd).toBe(0.3);
  });

  it("isOverBudget checks both limits (either triggers)", () => {
    tracker.record("s1", 100, 10.0);

    // Under token limit but over cost limit
    const config: BudgetConfig = { maxTokens: 1000, maxCostUsd: 1.0 };
    expect(tracker.isOverBudget("s1", config)).toBe(true);
  });
});
