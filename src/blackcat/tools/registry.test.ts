import { describe, expect, it } from "vitest";

import { ExecTool } from "./builtin/exec.js";
import { ToolRegistry } from "./registry.js";
import type { Tool, ToolDefinition, ToolResult } from "./types.js";

class MockTool implements Tool {
  definition(): ToolDefinition {
    return {
      name: "mock",
      description: "Mock test tool",
      parameters: {
        value: {
          type: "string",
          description: "value",
          required: true,
        },
      },
    };
  }

  async execute(args: Record<string, unknown>): Promise<ToolResult> {
    return {
      success: true,
      output: String(args.value ?? ""),
    };
  }
}

describe("ToolRegistry", () => {
  it("registers tools and list returns definitions", () => {
    const registry = new ToolRegistry();
    registry.register(new MockTool());

    const list = registry.list();
    expect(list).toHaveLength(1);
    expect(list[0]?.name).toBe("mock");
  });

  it("executes tool by name successfully", async () => {
    const registry = new ToolRegistry();
    registry.register(new MockTool());

    const result = await registry.execute("mock", { value: "ok" });
    expect(result.success).toBe(true);
    expect(result.output).toBe("ok");
  });

  it("returns ToolResult error for unknown tool", async () => {
    const registry = new ToolRegistry();

    const result = await registry.execute("missing", {});
    expect(result.success).toBe(false);
    expect(result.error).toBe("tool not found: missing");
  });

  it("filter(null) keeps all tools", () => {
    const registry = new ToolRegistry();
    registry.register(new MockTool());
    registry.register(new ExecTool());

    const filtered = registry.filter(null);
    expect(filtered.list().map((d) => d.name).sort()).toEqual(["exec", "mock"]);
  });

  it("filter([]) keeps no tools", () => {
    const registry = new ToolRegistry();
    registry.register(new MockTool());

    const filtered = registry.filter([]);
    expect(filtered.list()).toEqual([]);
  });

  it("filter(['exec']) keeps only exec tool", () => {
    const registry = new ToolRegistry();
    registry.register(new MockTool());
    registry.register(new ExecTool());

    const filtered = registry.filter(["exec"]);
    expect(filtered.list().map((d) => d.name)).toEqual(["exec"]);
  });

  it("returns error ToolResult when required param is missing", async () => {
    const registry = new ToolRegistry();
    registry.register(new MockTool());

    const result = await registry.execute("mock", {});
    expect(result.success).toBe(false);
    expect(result.error).toBe("missing required field: value");
  });
});
