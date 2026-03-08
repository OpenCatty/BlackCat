import { AnthropicBackend } from "./anthropic.js";
import { FallbackChain } from "./fallback.js";
import { GeminiBackend } from "./gemini.js";
import { OllamaBackend } from "./ollama.js";
import { OpenAIBackend } from "./openai.js";
import type { LLMBackend, ProviderConfig } from "./types.js";

export function createProvider(config: ProviderConfig): LLMBackend {
  switch (config.provider) {
    case "openai":
      return new OpenAIBackend(config);
    case "anthropic":
      return new AnthropicBackend(config);
    case "gemini":
      return new GeminiBackend(config);
    case "ollama":
      return new OllamaBackend(config);
    default:
      throw new Error(`Unknown provider: ${config.provider}`);
  }
}

export function createFallbackChain(configs: ProviderConfig[]): LLMBackend {
  const providers = configs.map(createProvider);
  return providers.length === 1 ? providers[0]! : new FallbackChain(providers);
}
