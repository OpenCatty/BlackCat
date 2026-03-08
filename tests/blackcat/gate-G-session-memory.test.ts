import { mkdtemp, rm } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { SessionStore, makeSessionId } from "../../src/blackcat/sessions/store.js";
import { MemoryStore } from "../../src/blackcat/memory/store.js";

let tmpDir: string;

beforeEach(async () => {
  tmpDir = await mkdtemp(path.join(os.tmpdir(), "gate-g-"));
});

afterEach(async () => {
  await rm(tmpDir, { recursive: true, force: true });
});

describe("Gate G — SessionStore CRUD", () => {
  it("getOrCreate creates a new session", () => {
    const store = new SessionStore(path.join(tmpDir, "sessions.db"));
    const session = store.getOrCreate("telegram", "bot1", "user1");

    expect(session.id).toBe("telegram:bot1:user1");
    expect(session.channel).toBe("telegram");
    expect(session.accountId).toBe("bot1");
    expect(session.peer).toBe("user1");
    expect(session.messages).toEqual([]);
    expect(session.createdAt).toBeGreaterThan(0);

    store.close();
  });

  it("getOrCreate returns existing session", () => {
    const store = new SessionStore(path.join(tmpDir, "sessions.db"));
    const s1 = store.getOrCreate("telegram", "bot1", "user1");
    const s2 = store.getOrCreate("telegram", "bot1", "user1");

    expect(s1.id).toBe(s2.id);
    expect(s1.createdAt).toBe(s2.createdAt);

    store.close();
  });

  it("load returns undefined for nonexistent session", () => {
    const store = new SessionStore(path.join(tmpDir, "sessions.db"));
    expect(store.load("nonexistent:id:here")).toBeUndefined();
    store.close();
  });

  it("appendMessage adds messages to session", () => {
    const store = new SessionStore(path.join(tmpDir, "sessions.db"));
    const session = store.getOrCreate("discord", "bot2", "user2");

    store.appendMessage(session.id, {
      role: "user",
      content: "Hello!",
      timestamp: Date.now(),
    });

    store.appendMessage(session.id, {
      role: "assistant",
      content: "Hi there!",
      timestamp: Date.now(),
    });

    const history = store.getHistory(session.id);
    expect(history).toHaveLength(2);
    expect(history[0]!.role).toBe("user");
    expect(history[1]!.role).toBe("assistant");

    store.close();
  });

  it("getHistory with limit returns last N messages", () => {
    const store = new SessionStore(path.join(tmpDir, "sessions.db"));
    const session = store.getOrCreate("telegram", "bot1", "user1");

    for (let i = 0; i < 5; i++) {
      store.appendMessage(session.id, {
        role: "user",
        content: `message ${i}`,
        timestamp: Date.now() + i,
      });
    }

    const last2 = store.getHistory(session.id, 2);
    expect(last2).toHaveLength(2);
    expect(last2[0]!.content).toBe("message 3");
    expect(last2[1]!.content).toBe("message 4");

    store.close();
  });

  it("deleteSession removes session", () => {
    const store = new SessionStore(path.join(tmpDir, "sessions.db"));
    store.getOrCreate("telegram", "bot1", "user1");

    store.deleteSession("telegram:bot1:user1");
    expect(store.load("telegram:bot1:user1")).toBeUndefined();

    store.close();
  });

  it("makeSessionId formula: channel:accountId:peer", () => {
    expect(makeSessionId("telegram", "bot123", "user456")).toBe("telegram:bot123:user456");
    expect(makeSessionId("discord", "srv1", "user1")).toBe("discord:srv1:user1");
    expect(makeSessionId("whatsapp", "num1", "num2")).toBe("whatsapp:num1:num2");
  });
});

describe("Gate G — MemoryStore", () => {
  it("add and search roundtrip", () => {
    const store = new MemoryStore(path.join(tmpDir, "memory.db"));

    store.add("The quick brown fox jumps over the lazy dog", ["test"], "unit-test");
    store.add("A completely different sentence about cats", ["test"], "unit-test");

    const results = store.search("fox");
    expect(results.length).toBeGreaterThanOrEqual(1);
    expect(results[0]!.content).toContain("fox");

    store.close();
  });

  it("add returns MemoryEntry with id and createdAt", () => {
    const store = new MemoryStore(path.join(tmpDir, "memory.db"));
    const entry = store.add("test content", ["tag1"], "src");

    expect(entry.id).toBeTruthy();
    expect(entry.content).toBe("test content");
    expect(entry.tags).toEqual(["tag1"]);
    expect(entry.source).toBe("src");
    expect(entry.createdAt).toBeGreaterThan(0);

    store.close();
  });

  it("get by id", () => {
    const store = new MemoryStore(path.join(tmpDir, "memory.db"));
    const entry = store.add("findable content");
    const found = store.get(entry.id);

    expect(found).toBeDefined();
    expect(found!.content).toBe("findable content");

    store.close();
  });

  it("delete removes entry", () => {
    const store = new MemoryStore(path.join(tmpDir, "memory.db"));
    const entry = store.add("to be deleted");

    const deleted = store.delete(entry.id);
    expect(deleted).toBe(true);
    expect(store.get(entry.id)).toBeUndefined();

    store.close();
  });

  it("list returns all entries ordered DESC by created_at", () => {
    const store = new MemoryStore(path.join(tmpDir, "memory.db"));
    store.add("first");
    store.add("second");
    store.add("third");

    const all = store.list();
    expect(all).toHaveLength(3);
    // All three entries should be present
    const contents = all.map((e) => e.content);
    expect(contents).toContain("first");
    expect(contents).toContain("second");
    expect(contents).toContain("third");
    // DESC order: most recent first (if timestamps differ)
    // When timestamps are equal (same ms), order is by rowid DESC
    // Either way, "third" was inserted last so it should be first or near first
    expect(all[0]!.createdAt).toBeGreaterThanOrEqual(all[2]!.createdAt);

    store.close();
  });

  it("search uses FTS5 or LIKE fallback", () => {
    const store = new MemoryStore(path.join(tmpDir, "memory.db"));
    store.add("TypeScript migration complete");
    store.add("Go implementation pending");

    // Either FTS5 or LIKE should work
    const results = store.search("TypeScript");
    expect(results.length).toBeGreaterThanOrEqual(1);
    expect(results[0]!.content).toContain("TypeScript");

    store.close();
  });

  it("core memory set/get", () => {
    const store = new MemoryStore(path.join(tmpDir, "memory.db"));
    store.setCoreMemory("user_name", "Alice");

    expect(store.getCoreMemory("user_name")).toBe("Alice");
    expect(store.getCoreMemory("nonexistent")).toBeUndefined();

    store.close();
  });

  it("core memory list and delete", () => {
    const store = new MemoryStore(path.join(tmpDir, "memory.db"));
    store.setCoreMemory("key_a", "val_a");
    store.setCoreMemory("key_b", "val_b");

    const coreList = store.listCoreMemory();
    expect(coreList).toHaveLength(2);
    expect(coreList[0]!.key).toBe("key_a");

    store.deleteCoreMemory("key_a");
    expect(store.getCoreMemory("key_a")).toBeUndefined();

    store.close();
  });
});
