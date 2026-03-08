import type { Tool, ToolDefinition, ToolResult } from "./types.js";

function validateRequiredParams(
  definition: ToolDefinition,
  args: Record<string, unknown>,
): string | undefined {
  for (const [name, parameter] of Object.entries(definition.parameters)) {
    if (parameter.required && !(name in args)) {
      return `missing required field: ${name}`;
    }
  }
  return undefined;
}

export class ToolRegistry {
  private readonly tools = new Map<string, Tool>();

  register(tool: Tool): void {
    this.tools.set(tool.definition().name, tool);
  }

  get(name: string): Tool | undefined {
    return this.tools.get(name);
  }

  list(): ToolDefinition[] {
    return Array.from(this.tools.values()).map((tool) => tool.definition());
  }

  async execute(name: string, args: Record<string, unknown>): Promise<ToolResult> {
    const tool = this.get(name);
    if (!tool) {
      return {
        success: false,
        output: "",
        error: `tool not found: ${name}`,
      };
    }

    const definition = tool.definition();
    const validationError = validateRequiredParams(definition, args);
    if (validationError) {
      return {
        success: false,
        output: "",
        error: validationError,
      };
    }

    try {
      return await tool.execute(args);
    } catch (error) {
      return {
        success: false,
        output: "",
        error: error instanceof Error ? error.message : String(error),
      };
    }
  }

  filter(allowedNames: string[] | null): ToolRegistry {
    const filtered = new ToolRegistry();

    if (allowedNames === null) {
      for (const tool of this.tools.values()) {
        filtered.register(tool);
      }
      return filtered;
    }

    const allowed = new Set(allowedNames);
    for (const [name, tool] of this.tools.entries()) {
      if (allowed.has(name)) {
        filtered.register(tool);
      }
    }

    return filtered;
  }
}
