export interface LLMMessage {
  role: "user" | "assistant" | "system";
  content: string;
}

export interface LLMResponse {
  content: string;
  inputTokens?: number;
  outputTokens?: number;
  model?: string;
  provider?: string;
}

export interface ProviderConfig {
  provider: string;
  model: string;
  apiKey?: string;
  baseURL?: string;
  temperature?: number;
  maxTokens?: number;
}

export interface LLMBackend {
  readonly name: string;
  chat(messages: LLMMessage[]): Promise<LLMResponse>;
}
