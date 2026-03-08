import type { LLMBackend, LLMMessage, LLMResponse, ProviderConfig } from "./types.js";

export class OpenAIBackend implements LLMBackend {
  readonly name = "openai";
  private readonly apiKey: string;
  private readonly model: string;
  private readonly baseURL: string;
  private readonly temperature: number;
  private readonly maxTokens: number;

  constructor(config: ProviderConfig) {
    if (!config.apiKey) {
      throw new Error("OpenAI apiKey required");
    }
    this.apiKey = config.apiKey;
    this.model = config.model;
    this.baseURL = config.baseURL ?? "https://api.openai.com/v1";
    this.temperature = config.temperature ?? 0.7;
    this.maxTokens = config.maxTokens ?? 4096;
  }

  async chat(messages: LLMMessage[]): Promise<LLMResponse> {
    const res = await fetch(`${this.baseURL}/chat/completions`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify({
        model: this.model,
        messages,
        temperature: this.temperature,
        max_tokens: this.maxTokens,
      }),
    });

    if (!res.ok) {
      throw new Error(`OpenAI error ${res.status}: ${await res.text()}`);
    }

    const data = (await res.json()) as any;
    return {
      content: data.choices?.[0]?.message?.content ?? "",
      inputTokens: data.usage?.prompt_tokens,
      outputTokens: data.usage?.completion_tokens,
      model: data.model,
      provider: "openai",
    };
  }
}
