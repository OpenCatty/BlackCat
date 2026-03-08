import type { LLMBackend, LLMMessage, LLMResponse } from "./types.js";

export class FallbackChain implements LLMBackend {
  readonly name = "fallback";

  constructor(private readonly providers: LLMBackend[]) {
    if (providers.length === 0) {
      throw new Error("FallbackChain requires at least one provider");
    }
  }

  async chat(messages: LLMMessage[]): Promise<LLMResponse> {
    let lastError: unknown;

    for (const provider of this.providers) {
      try {
        return await provider.chat(messages);
      } catch (err) {
        lastError = err;
      }
    }

    throw lastError;
  }
}
