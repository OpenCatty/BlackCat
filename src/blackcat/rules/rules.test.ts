import { describe, it, expect, beforeEach } from "vitest";

import { RulesEngine } from "./engine.js";
import type { Rule } from "./types.js";

function makeRule(
  name: string,
  pattern: string,
  action: "allow" | "block" | "flag",
): Rule {
  return { name, pattern, action };
}

describe("RulesEngine", () => {
  let engine: RulesEngine;

  beforeEach(() => {
    engine = new RulesEngine();
  });

  it("addRule and list returns registered rules", () => {
    const rule = makeRule("no-secrets", "password", "block");
    engine.addRule(rule);

    const listed = engine.list();
    expect(listed).toHaveLength(1);
    expect(listed[0].name).toBe("no-secrets");
  });

  it("evaluate returns matching rules", () => {
    engine.addRule(makeRule("has-hello", "hello", "flag"));
    engine.addRule(makeRule("has-world", "world", "flag"));

    const results = engine.evaluate("hello world");
    expect(results).toHaveLength(2);
    expect(results[0].rule.name).toBe("has-hello");
    expect(results[1].rule.name).toBe("has-world");
    expect(results.every((r) => r.matched)).toBe(true);
  });

  it("evaluate returns empty array for no matches", () => {
    engine.addRule(makeRule("has-foo", "foo", "block"));

    const results = engine.evaluate("bar baz");
    expect(results).toHaveLength(0);
  });

  it("block rule blocks and check returns passed=false", () => {
    engine.addRule(makeRule("block-secret", "secret", "block"));

    const result = engine.check("this is a secret");
    expect(result.passed).toBe(false);
    expect(result.blocked).toBe(true);
    expect(result.reasons).toContain("block: block-secret");
  });

  it("allow rule passes and check returns passed=true", () => {
    engine.addRule(makeRule("allow-greet", "hello", "allow"));

    const result = engine.check("hello friend");
    expect(result.passed).toBe(true);
    expect(result.blocked).toBe(false);
    expect(result.flagged).toBe(false);
    expect(result.reasons).toHaveLength(0);
  });

  it("flag rule flags without blocking", () => {
    engine.addRule(makeRule("flag-warning", "warning", "flag"));

    const result = engine.check("this is a warning message");
    expect(result.passed).toBe(true);
    expect(result.blocked).toBe(false);
    expect(result.flagged).toBe(true);
    expect(result.reasons).toContain("flag: flag-warning");
  });

  it("empty rules passes all text", () => {
    const result = engine.check("anything goes");
    expect(result.passed).toBe(true);
    expect(result.blocked).toBe(false);
    expect(result.flagged).toBe(false);
    expect(result.reasons).toHaveLength(0);
  });

  it("text matching regex pattern", () => {
    engine.addRule(makeRule("email-detect", "[a-z]+@[a-z]+\\.[a-z]+", "flag"));

    const result = engine.check("contact me at user@example.com");
    expect(result.flagged).toBe(true);
  });

  it("block action takes precedence over flag", () => {
    engine.addRule(makeRule("flag-it", "danger", "flag"));
    engine.addRule(makeRule("block-it", "danger", "block"));

    const result = engine.check("danger zone");
    expect(result.blocked).toBe(true);
    expect(result.passed).toBe(false);
    // flagged should be false when blocked
    expect(result.flagged).toBe(false);
    expect(result.reasons).toContain("block: block-it");
    expect(result.reasons).toContain("flag: flag-it");
  });

  it("rules are evaluated in registration order", () => {
    engine.addRule(makeRule("first", "test", "flag"));
    engine.addRule(makeRule("second", "test", "block"));

    const results = engine.evaluate("test input");
    expect(results[0].rule.name).toBe("first");
    expect(results[1].rule.name).toBe("second");
  });

  it("pattern matching is case-insensitive", () => {
    engine.addRule(makeRule("upper", "HELLO", "flag"));

    const result = engine.check("hello world");
    expect(result.flagged).toBe(true);
  });

  it("evaluate accepts optional sessionId parameter", () => {
    engine.addRule(makeRule("r1", "test", "flag"));

    const results = engine.evaluate("test", "session-123");
    expect(results).toHaveLength(1);
  });

  it("multiple block rules accumulate reasons", () => {
    engine.addRule(makeRule("block-a", "bad", "block"));
    engine.addRule(makeRule("block-b", "evil", "block"));

    const result = engine.check("bad and evil content");
    expect(result.blocked).toBe(true);
    expect(result.reasons).toHaveLength(2);
    expect(result.reasons).toContain("block: block-a");
    expect(result.reasons).toContain("block: block-b");
  });
});
