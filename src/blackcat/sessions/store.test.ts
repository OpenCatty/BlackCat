import { randomUUID } from "node:crypto";
import { existsSync, mkdirSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { afterEach, describe, expect, it } from "vitest";

import { makeSessionId, SessionStore } from "./store.js";

const tempRoots: string[] = [];

function makeDbPath(): string {
  const root = join(tmpdir(), `blackcat-sessions-${randomUUID()}`);
  mkdirSync(root, { recursive: true });
  tempRoots.push(root);
  return join(root, "sessions.sqlite");
}

afterEach(() => {
  while (tempRoots.length > 0) {
    const root = tempRoots.pop();
    if (root && existsSync(root)) {
      rmSync(root, { recursive: true, force: true });
    }
  }
});

describe("SessionStore", () => {
  it("getOrCreate creates a new session", () => {
    const store = new SessionStore(makeDbPath());
    const session = store.getOrCreate("telegram", "acc1", "peerA");

    expect(session.id).toBe("telegram:acc1:peerA");
    expect(session.messages).toEqual([]);
    expect(session.createdAt).toBeGreaterThan(0);
    expect(session.updatedAt).toBeGreaterThan(0);

    store.close();
  });

  it("getOrCreate with same key returns same session", () => {
    const store = new SessionStore(makeDbPath());
    const first = store.getOrCreate("discord", "acc-01", "peer-01");
    const second = store.getOrCreate("discord", "acc-01", "peer-01");

    expect(second.id).toBe(first.id);
    expect(second.createdAt).toBe(first.createdAt);

    store.close();
  });

  it("appendMessage appends message into session history", () => {
    const store = new SessionStore(makeDbPath());
    const session = store.getOrCreate("whatsapp", "acc", "peer");

    store.appendMessage(session.id, {
      role: "user",
      content: "Hello",
      timestamp: Date.now(),
    });

    const history = store.getHistory(session.id);
    expect(history).toHaveLength(1);
    expect(history[0]?.content).toBe("Hello");

    store.close();
  });

  it("getHistory respects limit", () => {
    const store = new SessionStore(makeDbPath());
    const session = store.getOrCreate("telegram", "acc", "peer");

    store.appendMessage(session.id, { role: "user", content: "m1", timestamp: 1 });
    store.appendMessage(session.id, { role: "assistant", content: "m2", timestamp: 2 });
    store.appendMessage(session.id, { role: "assistant", content: "m3", timestamp: 3 });

    const limited = store.getHistory(session.id, 2);
    expect(limited.map((m) => m.content)).toEqual(["m2", "m3"]);

    store.close();
  });

  it("session key formula is stable channel:accountId:peer", () => {
    expect(makeSessionId("telegram", "account-x", "peer-y")).toBe("telegram:account-x:peer-y");
  });
});
