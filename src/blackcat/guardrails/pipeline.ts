import { buildInputDenyPatterns, buildToolApprovalPatterns } from "./patterns.js";
import type { GuardrailResult, GuardrailsConfig } from "./types.js";

const MAX_OUTPUT_LENGTH = 50_000;

export class GuardrailsPipeline {
  private readonly inputPatterns: RegExp[];
  private readonly toolPatterns: RegExp[];

  constructor(private readonly config: GuardrailsConfig = {}) {
    this.inputPatterns = buildInputDenyPatterns(config.denyPatterns);
    this.toolPatterns = buildToolApprovalPatterns(config.requireApprovalPatterns);
  }

  checkInput(input: string): GuardrailResult {
    if (this.config.inputEnabled === false) {
      return { allow: true };
    }

    for (const pattern of this.inputPatterns) {
      if (pattern.test(input)) {
        return { allow: false, reason: "input blocked by prompt injection guardrail" };
      }
    }

    return { allow: true };
  }

  checkTool(toolName: string, toolArgs: Record<string, unknown>): GuardrailResult {
    if (this.config.toolEnabled === false) {
      return { allow: true };
    }

    const payload = `${toolName} ${JSON.stringify(toolArgs ?? {})}`.trim();
    for (const pattern of this.toolPatterns) {
      if (pattern.test(payload)) {
        // HITL approval flow will be implemented in later wave.
        return { allow: true, reason: "tool matched approval pattern (approval flow pending)" };
      }
    }

    return { allow: true };
  }

  checkOutput(output: string): GuardrailResult {
    if (this.config.outputEnabled === false) {
      return { allow: true };
    }

    const trimmed = output.trim();
    if (trimmed.length === 0) {
      return { allow: false, reason: "output is empty" };
    }

    if (trimmed.startsWith("Error:")) {
      return { allow: false, reason: "output starts with raw error" };
    }

    if (output.length >= MAX_OUTPUT_LENGTH) {
      return { allow: false, reason: "output exceeds length limit" };
    }

    return { allow: true };
  }
}

export function createGuardrails(config?: GuardrailsConfig): GuardrailsPipeline {
  return new GuardrailsPipeline(config);
}
