import type { LLMBackend, LLMMessage, LLMResponse, ProviderConfig } from "./types.js";

export class OllamaBackend implements LLMBackend {
  readonly name = "ollama";
  private readonly baseURL: string;

  constructor(private readonly config: ProviderConfig) {
    this.baseURL = config.baseURL ?? "http://localhost:11434";
  }

  async chat(messages: LLMMessage[]): Promise<LLMResponse> {
    const res = await fetch(`${this.baseURL}/api/chat`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        model: this.config.model,
        messages,
        stream: false,
      }),
    });

    if (!res.ok) {
      throw new Error(`Ollama error ${res.status}: ${await res.text()}`);
    }

    const data = (await res.json()) as any;
    return {
      content: data.message?.content ?? "",
      provider: "ollama",
      model: this.config.model,
    };
  }
}
