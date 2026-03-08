import { describe, expect, it } from "vitest";

import { classifyMessage } from "./router.js";
import { DEFAULT_ROLES } from "./roles.js";

describe("classifyMessage parity with Go router", () => {
  const fixtures = [
    // exact role matches
    { msg: "deploy nginx to k8s cluster", expected: "phantom" },
    { msg: "what is btc doing today", expected: "astrology" },
    { msg: "refactor this function in Go", expected: "wizard" },
    { msg: "write a tweet about our product", expected: "artist" },
    { msg: "write a blog post draft", expected: "scribe" },
    { msg: "research the best LLM options", expected: "explorer" },
    { msg: "hello there how are you", expected: "oracle" },
    // priority edge cases
    { msg: "deploy this code fix", expected: "phantom" },
    { msg: "CODE review for docker setup", expected: "phantom" },
    { msg: "analyze the eth blockchain data", expected: "astrology" },
    // fallback
    { msg: "", expected: "oracle" },
    { msg: "random message no keywords", expected: "oracle" },
  ] as const;

  for (const fx of fixtures) {
    it(`classifies: ${JSON.stringify(fx.msg)} -> ${fx.expected}`, () => {
      const role = classifyMessage(fx.msg, DEFAULT_ROLES);
      expect(role.name).toBe(fx.expected);
    });
  }

  it("falls back to defaults when roles is empty", () => {
    const role = classifyMessage("unknown input", []);
    expect(role.name).toBe("oracle");
  });
});
