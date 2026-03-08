import { spawnSync } from "node:child_process";

import type { Tool, ToolDefinition, ToolResult } from "../types.js";

const DEFAULT_TIMEOUT_MS = 30_000;
const DENY_PATTERNS: RegExp[] = [
  /\brm\s+-rf\s+\//i,
  /\brm\s+-rf\s+--no-preserve-root\b/i,
  /\bformat\b/i,
  /\bmkfs\b/i,
  /\bshutdown\b/i,
  /\breboot\b/i,
  /\bpoweroff\b/i,
  /\bdel\s+\/[sqf]/i,
];

export class ExecTool implements Tool {
  definition(): ToolDefinition {
    return {
      name: "exec",
      description: "Run shell command",
      parameters: {
        command: {
          type: "string",
          description: "Shell command to execute",
          required: true,
        },
      },
    };
  }

  async execute(args: Record<string, unknown>): Promise<ToolResult> {
    const command = typeof args.command === "string" ? args.command.trim() : "";
    if (!command) {
      return { success: false, output: "", error: "command is required" };
    }

    for (const pattern of DENY_PATTERNS) {
      if (pattern.test(command)) {
        return {
          success: false,
          output: "",
          error: "command blocked by deny list",
        };
      }
    }

    const result = spawnSync(command, {
      shell: true,
      timeout: DEFAULT_TIMEOUT_MS,
      encoding: "utf8",
    });

    const stdout = result.stdout ?? "";
    const stderr = result.stderr ?? "";
    const output = [stdout, stderr].filter(Boolean).join("\n").trim();

    if (result.error) {
      return {
        success: false,
        output,
        error: result.error.message,
      };
    }

    return {
      success: result.status === 0,
      output,
      error: result.status === 0 ? undefined : `process exited with code ${result.status}`,
    };
  }
}
