import { describe, it, expect, beforeEach, vi } from "vitest";

import { HookRegistry } from "./registry.js";
import type { Hook, HookContext } from "./types.js";

function makeCtx(text = "hello", sessionId?: string): HookContext {
  return { text, sessionId, metadata: {} };
}

function makeHook(
  name: string,
  phase: "pre" | "post",
  fn?: (ctx: HookContext) => HookContext,
): Hook {
  return {
    name,
    phase,
    execute: (ctx) => ({
      context: fn ? fn(ctx) : ctx,
    }),
  };
}

describe("HookRegistry", () => {
  let registry: HookRegistry;

  beforeEach(() => {
    registry = new HookRegistry();
  });

  it("register and list returns registered hooks", () => {
    registry.register(makeHook("a", "pre"));
    registry.register(makeHook("b", "post"));

    const listed = registry.list();
    expect(listed).toEqual([
      { name: "a", phase: "pre" },
      { name: "b", phase: "post" },
    ]);
  });

  it("list returns empty array when nothing registered", () => {
    expect(registry.list()).toEqual([]);
  });

  it("run pre-phase hooks only", async () => {
    const calls: string[] = [];
    registry.register({
      name: "pre-hook",
      phase: "pre",
      execute: (ctx) => {
        calls.push("pre");
        return { context: ctx };
      },
    });
    registry.register({
      name: "post-hook",
      phase: "post",
      execute: (ctx) => {
        calls.push("post");
        return { context: ctx };
      },
    });

    await registry.run("pre", makeCtx());
    expect(calls).toEqual(["pre"]);
  });

  it("run post-phase hooks only", async () => {
    const calls: string[] = [];
    registry.register({
      name: "pre-hook",
      phase: "pre",
      execute: (ctx) => {
        calls.push("pre");
        return { context: ctx };
      },
    });
    registry.register({
      name: "post-hook",
      phase: "post",
      execute: (ctx) => {
        calls.push("post");
        return { context: ctx };
      },
    });

    await registry.run("post", makeCtx());
    expect(calls).toEqual(["post"]);
  });

  it("runs multiple hooks in registration order", async () => {
    const order: number[] = [];

    for (const i of [1, 2, 3]) {
      registry.register({
        name: `hook-${i}`,
        phase: "pre",
        execute: (ctx) => {
          order.push(i);
          return { context: ctx };
        },
      });
    }

    await registry.run("pre", makeCtx());
    expect(order).toEqual([1, 2, 3]);
  });

  it("hook can modify context text", async () => {
    registry.register(
      makeHook("upper", "pre", (ctx) => ({
        ...ctx,
        text: ctx.text.toUpperCase(),
      })),
    );

    const result = await registry.run("pre", makeCtx("hello"));
    expect(result.text).toBe("HELLO");
  });

  it("hook can modify context metadata", async () => {
    registry.register({
      name: "annotate",
      phase: "pre",
      execute: (ctx) => ({
        context: { ...ctx, metadata: { ...ctx.metadata, tagged: true } },
      }),
    });

    const result = await registry.run("pre", makeCtx());
    expect(result.metadata.tagged).toBe(true);
  });

  it("modifications chain through multiple hooks", async () => {
    registry.register(
      makeHook("append-1", "pre", (ctx) => ({
        ...ctx,
        text: ctx.text + "-a",
      })),
    );
    registry.register(
      makeHook("append-2", "pre", (ctx) => ({
        ...ctx,
        text: ctx.text + "-b",
      })),
    );

    const result = await registry.run("pre", makeCtx("start"));
    expect(result.text).toBe("start-a-b");
  });

  it("hook throwing error is caught gracefully and does not crash pipeline", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    registry.register({
      name: "bad-hook",
      phase: "pre",
      execute: () => {
        throw new Error("boom");
      },
    });
    registry.register(
      makeHook("good-hook", "pre", (ctx) => ({
        ...ctx,
        text: "survived",
      })),
    );

    const result = await registry.run("pre", makeCtx("original"));
    expect(result.text).toBe("survived");
    expect(consoleSpy).toHaveBeenCalledOnce();

    consoleSpy.mockRestore();
  });

  it("halt stops further hooks in phase", async () => {
    const calls: string[] = [];

    registry.register({
      name: "halter",
      phase: "pre",
      execute: (ctx) => {
        calls.push("halter");
        return { context: ctx, halt: true };
      },
    });
    registry.register({
      name: "after-halt",
      phase: "pre",
      execute: (ctx) => {
        calls.push("after-halt");
        return { context: ctx };
      },
    });

    await registry.run("pre", makeCtx());
    expect(calls).toEqual(["halter"]);
  });

  it("run with no matching hooks returns original context", async () => {
    registry.register(makeHook("post-only", "post"));

    const ctx = makeCtx("unchanged");
    const result = await registry.run("pre", ctx);
    expect(result.text).toBe("unchanged");
  });

  it("supports async hook execution", async () => {
    registry.register({
      name: "async-hook",
      phase: "pre",
      execute: async (ctx) => {
        await new Promise((r) => setTimeout(r, 1));
        return { context: { ...ctx, text: "async-done" } };
      },
    });

    const result = await registry.run("pre", makeCtx());
    expect(result.text).toBe("async-done");
  });
});
