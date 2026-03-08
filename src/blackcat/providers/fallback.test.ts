import { describe, expect, it } from "vitest";

import { FallbackChain } from "./fallback.js";
import type { LLMBackend } from "./types.js";

function makeOk(name: string): LLMBackend {
  return {
    name,
    async chat(_msgs) {
      return { content: `response from ${name}`, provider: name };
    },
  };
}

function makeFail(name: string, msg = "provider error"): LLMBackend {
  return {
    name,
    async chat(_msgs) {
      throw new Error(msg);
    },
  };
}

describe("FallbackChain", () => {
  it("returns response from first provider when it succeeds", async () => {
    const chain = new FallbackChain([makeOk("p1"), makeOk("p2")]);
    const r = await chain.chat([{ role: "user", content: "hi" }]);
    expect(r.content).toBe("response from p1");
  });

  it("falls back to second provider when first fails", async () => {
    const chain = new FallbackChain([makeFail("p1"), makeOk("p2")]);
    const r = await chain.chat([{ role: "user", content: "hi" }]);
    expect(r.content).toBe("response from p2");
  });

  it("throws when all providers fail", async () => {
    const chain = new FallbackChain([makeFail("p1", "err1"), makeFail("p2", "err2")]);
    await expect(chain.chat([{ role: "user", content: "hi" }])).rejects.toThrow("err2");
  });

  it("works with single provider", async () => {
    const chain = new FallbackChain([makeOk("solo")]);
    const r = await chain.chat([{ role: "user", content: "hi" }]);
    expect(r.content).toBe("response from solo");
  });

  it("throws when constructed with empty array", () => {
    expect(() => new FallbackChain([])).toThrow();
  });
});
