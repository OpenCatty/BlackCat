import type { CheckResult, Rule, RuleResult } from "./types.js";

export class RulesEngine {
  private readonly rules: Rule[] = [];

  /** Add a content rule. */
  addRule(rule: Rule): void {
    this.rules.push(rule);
  }

  /** Evaluate text against all rules. Returns every matching RuleResult. */
  evaluate(text: string, _sessionId?: string): RuleResult[] {
    const results: RuleResult[] = [];

    for (const rule of this.rules) {
      const regex = new RegExp(rule.pattern, "i");
      const matched = regex.test(text);
      if (matched) {
        results.push({ rule, matched });
      }
    }

    return results;
  }

  /**
   * High-level check: returns a summary of whether the text passed, was blocked, or was flagged.
   * A `block` action takes precedence over `flag`.
   */
  check(text: string): CheckResult {
    const matches = this.evaluate(text);

    const blocked = matches.some((m) => m.rule.action === "block");
    const flagged = matches.some((m) => m.rule.action === "flag");

    const reasons: string[] = [];
    for (const m of matches) {
      if (m.rule.action === "block" || m.rule.action === "flag") {
        reasons.push(`${m.rule.action}: ${m.rule.name}`);
      }
    }

    return {
      passed: !blocked,
      blocked,
      flagged: flagged && !blocked ? true : false,
      reasons,
    };
  }

  /** List registered rules. */
  list(): Rule[] {
    return [...this.rules];
  }
}
