import { randomUUID } from "node:crypto";
import { existsSync, mkdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { MemoryStore } from "./store.js";

const tempRoots: string[] = [];

function makeDbPath(): string {
  const root = join(tmpdir(), `blackcat-memory-${randomUUID()}`);
  mkdirSync(root, { recursive: true });
  tempRoots.push(root);
  return join(root, "memory.sqlite");
}

afterEach(() => {
  while (tempRoots.length > 0) {
    const root = tempRoots.pop();
    if (root && existsSync(root)) {
      rmSync(root, { recursive: true, force: true });
    }
  }
});

describe("MemoryStore", () => {
  it("add + get archival memory", () => {
    const store = new MemoryStore(makeDbPath());
    const entry = store.add("User likes black coffee", ["preference"], "chat");

    const loaded = store.get(entry.id);
    expect(loaded?.content).toBe("User likes black coffee");
    expect(loaded?.tags).toEqual(["preference"]);
    expect(loaded?.source).toBe("chat");

    store.close();
  });

  it("search returns relevant results", () => {
    const store = new MemoryStore(makeDbPath());
    store.add("User lives in Jakarta", ["profile"], "onboard");
    store.add("Reminder: buy apples", ["todo"], "chat");

    const results = store.search("Jakarta", 5);
    expect(results.length).toBeGreaterThan(0);
    expect(results.some((entry) => entry.content.includes("Jakarta"))).toBe(true);

    store.close();
  });

  it("setCoreMemory + getCoreMemory", () => {
    const store = new MemoryStore(makeDbPath());
    store.setCoreMemory("timezone", "Asia/Jakarta");

    expect(store.getCoreMemory("timezone")).toBe("Asia/Jakarta");

    store.close();
  });

  it("deleteCoreMemory removes entry", () => {
    const store = new MemoryStore(makeDbPath());
    store.setCoreMemory("language", "id");
    store.deleteCoreMemory("language");

    expect(store.getCoreMemory("language")).toBeUndefined();

    store.close();
  });

  it("list respects limit", () => {
    const store = new MemoryStore(makeDbPath());
    store.add("a");
    store.add("b");
    store.add("c");

    const rows = store.list(2);
    expect(rows).toHaveLength(2);

    store.close();
  });
});
