import { describe, expect, it } from "vitest";
import { FallbackChain } from "../../src/blackcat/providers/fallback.js";
import type { LLMBackend, LLMMessage, LLMResponse } from "../../src/blackcat/providers/types.js";

function mockBackend(name: string, response?: string, shouldFail?: boolean): LLMBackend {
  return {
    name,
    async chat(_messages: LLMMessage[]): Promise<LLMResponse> {
      if (shouldFail) {
        throw new Error(`${name} failed`);
      }
      return { content: response ?? `response from ${name}` };
    },
  };
}

describe("Gate F — FallbackChain Providers", () => {
  it("first provider success returns immediately", async () => {
    const chain = new FallbackChain([
      mockBackend("openai", "openai response"),
      mockBackend("anthropic", "anthropic response"),
    ]);

    const result = await chain.chat([{ role: "user", content: "hello" }]);
    expect(result.content).toBe("openai response");
  });

  it("first fails → second succeeds", async () => {
    const chain = new FallbackChain([
      mockBackend("openai", undefined, true), // fails
      mockBackend("anthropic", "fallback response"),
    ]);

    const result = await chain.chat([{ role: "user", content: "hello" }]);
    expect(result.content).toBe("fallback response");
  });

  it("all fail → throws last error", async () => {
    const chain = new FallbackChain([
      mockBackend("openai", undefined, true),
      mockBackend("anthropic", undefined, true),
    ]);

    await expect(chain.chat([{ role: "user", content: "hello" }]))
      .rejects.toThrow("anthropic failed");
  });

  it("empty providers array throws at construction", () => {
    expect(() => new FallbackChain([])).toThrow(/at least one provider/i);
  });

  it("single provider success", async () => {
    const chain = new FallbackChain([
      mockBackend("only", "sole response"),
    ]);

    const result = await chain.chat([{ role: "user", content: "test" }]);
    expect(result.content).toBe("sole response");
  });

  it("single provider failure throws", async () => {
    const chain = new FallbackChain([
      mockBackend("only", undefined, true),
    ]);

    await expect(chain.chat([{ role: "user", content: "test" }]))
      .rejects.toThrow("only failed");
  });

  it("three providers: first two fail, third succeeds", async () => {
    const chain = new FallbackChain([
      mockBackend("p1", undefined, true),
      mockBackend("p2", undefined, true),
      mockBackend("p3", "third time's a charm"),
    ]);

    const result = await chain.chat([{ role: "user", content: "hello" }]);
    expect(result.content).toBe("third time's a charm");
  });

  it("chain has name 'fallback'", () => {
    const chain = new FallbackChain([mockBackend("test")]);
    expect(chain.name).toBe("fallback");
  });
});
