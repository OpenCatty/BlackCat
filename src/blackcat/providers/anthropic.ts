import type { LLMBackend, LLMMessage, LLMResponse, ProviderConfig } from "./types.js";

export class AnthropicBackend implements LLMBackend {
  readonly name = "anthropic";

  constructor(private readonly config: ProviderConfig) {
    if (!config.apiKey) {
      throw new Error("Anthropic apiKey required");
    }
  }

  async chat(messages: LLMMessage[]): Promise<LLMResponse> {
    const systemMsg = messages.find((m) => m.role === "system")?.content;
    const nonSystem = messages.filter((m) => m.role !== "system");
    const body: Record<string, unknown> = {
      model: this.config.model,
      max_tokens: this.config.maxTokens ?? 4096,
      messages: nonSystem,
    };

    if (systemMsg) {
      body.system = systemMsg;
    }

    const res = await fetch("https://api.anthropic.com/v1/messages", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "x-api-key": this.config.apiKey!,
        "anthropic-version": "2023-06-01",
      },
      body: JSON.stringify(body),
    });

    if (!res.ok) {
      throw new Error(`Anthropic error ${res.status}: ${await res.text()}`);
    }

    const data = (await res.json()) as any;
    return {
      content: data.content?.[0]?.text ?? "",
      inputTokens: data.usage?.input_tokens,
      outputTokens: data.usage?.output_tokens,
      model: data.model,
      provider: "anthropic",
    };
  }
}
