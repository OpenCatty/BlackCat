import type { Tool, ToolDefinition, ToolResult } from "../types.js";

const MAX_OUTPUT_BYTES = 10_000;

function truncateToBytes(value: string, maxBytes: number): string {
  const buf = Buffer.from(value, "utf8");
  if (buf.length <= maxBytes) {
    return value;
  }
  return buf.subarray(0, maxBytes).toString("utf8");
}

export class WebTool implements Tool {
  definition(): ToolDefinition {
    return {
      name: "web",
      description: "Fetch URL content",
      parameters: {
        url: {
          type: "string",
          description: "URL to fetch",
          required: true,
        },
        method: {
          type: "string",
          description: "HTTP method (GET or POST)",
        },
      },
    };
  }

  async execute(args: Record<string, unknown>): Promise<ToolResult> {
    const rawUrl = typeof args.url === "string" ? args.url.trim() : "";
    if (!rawUrl) {
      return { success: false, output: "", error: "url is required" };
    }

    let url: URL;
    try {
      url = new URL(rawUrl);
    } catch {
      return { success: false, output: "", error: "invalid url" };
    }

    const rawMethod = typeof args.method === "string" ? args.method.trim().toUpperCase() : "GET";
    const method = rawMethod === "POST" ? "POST" : "GET";

    try {
      const response = await fetch(url, { method });
      const body = await response.text();

      return {
        success: response.ok,
        output: truncateToBytes(body, MAX_OUTPUT_BYTES),
        error: response.ok ? undefined : `http ${response.status}`,
      };
    } catch (error) {
      return {
        success: false,
        output: "",
        error: error instanceof Error ? error.message : String(error),
      };
    }
  }
}
