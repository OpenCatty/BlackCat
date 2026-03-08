import type { Tool, ToolDefinition, ToolResult } from "../types.js";

export class StatusTool implements Tool {
  definition(): ToolDefinition {
    return {
      name: "status",
      description: "Get agent status",
      parameters: {},
    };
  }

  async execute(): Promise<ToolResult> {
    const payload = {
      uptime: process.uptime(),
      memoryUsage: process.memoryUsage(),
      version: process.version,
    };

    return {
      success: true,
      output: JSON.stringify(payload),
    };
  }
}
