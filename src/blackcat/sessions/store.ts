import { DatabaseSync } from "node:sqlite";

import type { Session, SessionMessage } from "./types.js";

const SCHEMA_VERSION = 1;

function now(): number {
  return Date.now();
}

function parseMessages(raw: string): SessionMessage[] {
  try {
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) {
      return [];
    }

    return parsed.flatMap((message) => {
      if (typeof message !== "object" || message === null) {
        return [];
      }

      const role = (message as { role?: unknown }).role;
      const content = (message as { content?: unknown }).content;
      const timestamp = (message as { timestamp?: unknown }).timestamp;

      if ((role !== "user" && role !== "assistant" && role !== "system") || typeof content !== "string" || typeof timestamp !== "number") {
        return [];
      }

      return [{ role, content, timestamp } satisfies SessionMessage];
    });
  } catch {
    return [];
  }
}

function toSession(row: {
  id: string;
  channel: string;
  account_id: string;
  peer: string;
  messages: string;
  created_at: number;
  updated_at: number;
}): Session {
  return {
    id: row.id,
    channel: row.channel,
    accountId: row.account_id,
    peer: row.peer,
    messages: parseMessages(row.messages),
    createdAt: row.created_at,
    updatedAt: row.updated_at,
  };
}

function makeSessionId(channel: string, accountId: string, peer: string): string {
  return `${channel}:${accountId}:${peer}`;
}

export class SessionStore {
  private readonly db: DatabaseSync;

  constructor(dbPath: string) {
    this.db = new DatabaseSync(dbPath);
    this.applyPragmas();
    this.initSchema();
  }

  getOrCreate(channel: string, accountId: string, peer: string): Session {
    const id = makeSessionId(channel, accountId, peer);
    const existing = this.load(id);
    if (existing) {
      return existing;
    }

    const timestamp = now();
    const session: Session = {
      id,
      channel,
      accountId,
      peer,
      messages: [],
      createdAt: timestamp,
      updatedAt: timestamp,
    };

    this.save(session);
    return session;
  }

  load(id: string): Session | undefined {
    const row = this.db
      .prepare(
        `
          SELECT id, channel, account_id, peer, messages, created_at, updated_at
          FROM sessions
          WHERE id = ?
        `,
      )
      .get(id) as
      | {
          id: string;
          channel: string;
          account_id: string;
          peer: string;
          messages: string;
          created_at: number;
          updated_at: number;
        }
      | undefined;

    if (!row) {
      return undefined;
    }

    return toSession(row);
  }

  save(session: Session): void {
    const existing = this.load(session.id);
    const createdAt = existing?.createdAt ?? session.createdAt ?? now();
    const updatedAt = session.updatedAt > 0 ? session.updatedAt : now();

    this.db
      .prepare(
        `
          INSERT INTO sessions (id, channel, account_id, peer, messages, created_at, updated_at)
          VALUES (?, ?, ?, ?, ?, ?, ?)
          ON CONFLICT(id) DO UPDATE SET
            channel=excluded.channel,
            account_id=excluded.account_id,
            peer=excluded.peer,
            messages=excluded.messages,
            updated_at=excluded.updated_at
        `,
      )
      .run(
        session.id,
        session.channel,
        session.accountId,
        session.peer,
        JSON.stringify(session.messages),
        createdAt,
        updatedAt,
      );
  }

  appendMessage(id: string, msg: SessionMessage): void {
    const session = this.load(id);
    if (!session) {
      throw new Error(`session not found: ${id}`);
    }

    session.messages.push(msg);
    session.updatedAt = now();
    this.save(session);
  }

  getHistory(id: string, limit?: number): SessionMessage[] {
    const session = this.load(id);
    if (!session) {
      return [];
    }

    if (typeof limit !== "number" || !Number.isFinite(limit) || limit <= 0) {
      return [...session.messages];
    }

    return session.messages.slice(-Math.trunc(limit));
  }

  deleteSession(id: string): void {
    this.db.prepare("DELETE FROM sessions WHERE id = ?").run(id);
  }

  close(): void {
    this.db.close();
  }

  private applyPragmas(): void {
    this.db.exec("PRAGMA journal_mode = WAL;");
    this.db.exec("PRAGMA foreign_keys = ON;");
    this.db.exec("PRAGMA busy_timeout = 5000;");
  }

  private initSchema(): void {
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS schema_version (
        version INTEGER PRIMARY KEY
      );

      CREATE TABLE IF NOT EXISTS sessions (
        id TEXT PRIMARY KEY,
        channel TEXT NOT NULL,
        account_id TEXT NOT NULL,
        peer TEXT NOT NULL,
        messages TEXT NOT NULL DEFAULT '[]',
        created_at INTEGER NOT NULL,
        updated_at INTEGER NOT NULL
      );

      CREATE INDEX IF NOT EXISTS idx_sessions_channel ON sessions(channel);
    `);

    const row = this.db.prepare("SELECT MAX(version) AS version FROM schema_version").get() as { version: number | null };
    const currentVersion = row.version ?? 0;

    if (currentVersion === 0) {
      this.db.prepare("INSERT INTO schema_version(version) VALUES (?)").run(SCHEMA_VERSION);
      return;
    }

    if (currentVersion !== SCHEMA_VERSION) {
      throw new Error(`unsupported sessions schema version: ${currentVersion}`);
    }
  }
}

export { makeSessionId };
