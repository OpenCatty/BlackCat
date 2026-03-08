export type RuleAction = "allow" | "block" | "flag";

export interface Rule {
  /** Unique name for this rule. */
  name: string;
  /** Regex pattern string to match against text. */
  pattern: string;
  /** Action to take when the pattern matches. */
  action: RuleAction;
}

export interface RuleResult {
  /** The rule that matched. */
  rule: Rule;
  /** Whether the pattern matched. */
  matched: boolean;
}

export interface RulesConfig {
  /** Pre-configured rules to load at construction. */
  rules?: Rule[];
}

export interface CheckResult {
  /** True if no block rules matched. */
  passed: boolean;
  /** True if at least one block rule matched. */
  blocked: boolean;
  /** True if at least one flag rule matched (and not blocked). */
  flagged: boolean;
  /** Human-readable reasons for block/flag. */
  reasons: string[];
}
