import type { LLMBackend, LLMMessage, LLMResponse, ProviderConfig } from "./types.js";

export class GeminiBackend implements LLMBackend {
  readonly name = "gemini";

  constructor(private readonly config: ProviderConfig) {
    if (!config.apiKey) {
      throw new Error("Gemini apiKey required");
    }
  }

  async chat(messages: LLMMessage[]): Promise<LLMResponse> {
    const contents = messages
      .filter((m) => m.role !== "system")
      .map((m) => ({
        role: m.role === "assistant" ? "model" : "user",
        parts: [{ text: m.content }],
      }));

    const url = `https://generativelanguage.googleapis.com/v1beta/models/${this.config.model}:generateContent?key=${this.config.apiKey}`;
    const res = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ contents }),
    });

    if (!res.ok) {
      throw new Error(`Gemini error ${res.status}: ${await res.text()}`);
    }

    const data = (await res.json()) as any;
    return {
      content: data.candidates?.[0]?.content?.parts?.[0]?.text ?? "",
      provider: "gemini",
      model: this.config.model,
    };
  }
}
