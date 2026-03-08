import { randomUUID } from "node:crypto";
import { DatabaseSync } from "node:sqlite";

import type { CoreMemoryEntry, MemoryEntry } from "./types.js";

const SCHEMA_VERSION = 1;

function now(): number {
  return Date.now();
}

function toMemoryEntry(row: { id: string; content: string; tags: string; source: string; created_at: number }): MemoryEntry {
  let parsedTags: string[] = [];
  try {
    const parsed = JSON.parse(row.tags) as unknown;
    if (Array.isArray(parsed)) {
      parsedTags = parsed.filter((value): value is string => typeof value === "string");
    }
  } catch {
    parsedTags = [];
  }

  return {
    id: row.id,
    content: row.content,
    tags: parsedTags,
    source: row.source,
    createdAt: row.created_at,
  };
}

export class MemoryStore {
  private readonly db: DatabaseSync;

  private readonly hasFts: boolean;

  constructor(dbPath: string) {
    this.db = new DatabaseSync(dbPath);
    this.applyPragmas();
    this.initSchemaVersion();
    this.initTables();
    this.hasFts = this.tryInitFts();
  }

  add(content: string, tags: string[] = [], source = ""): MemoryEntry {
    const id = randomUUID();
    const createdAt = now();

    this.db
      .prepare(
        `
          INSERT INTO memories (id, content, tags, source, created_at)
          VALUES (?, ?, ?, ?, ?)
        `,
      )
      .run(id, content, JSON.stringify(tags), source, createdAt);

    return { id, content, tags, source, createdAt };
  }

  search(query: string, limit = 10): MemoryEntry[] {
    const safeLimit = Math.max(1, Math.trunc(limit));

    if (this.hasFts) {
      const rows = this.db
        .prepare(
          `
            SELECT m.id, m.content, m.tags, m.source, m.created_at
            FROM memories_fts f
            JOIN memories m ON m.rowid = f.rowid
            WHERE memories_fts MATCH ?
            ORDER BY bm25(memories_fts), m.created_at DESC
            LIMIT ?
          `,
        )
        .all(query, safeLimit) as Array<{
        id: string;
        content: string;
        tags: string;
        source: string;
        created_at: number;
      }>;

      return rows.map(toMemoryEntry);
    }

    const like = `%${query}%`;
    const rows = this.db
      .prepare(
        `
          SELECT id, content, tags, source, created_at
          FROM memories
          WHERE content LIKE ?
          ORDER BY created_at DESC
          LIMIT ?
        `,
      )
      .all(like, safeLimit) as Array<{
      id: string;
      content: string;
      tags: string;
      source: string;
      created_at: number;
    }>;

    return rows.map(toMemoryEntry);
  }

  get(id: string): MemoryEntry | undefined {
    const row = this.db
      .prepare(
        `
          SELECT id, content, tags, source, created_at
          FROM memories
          WHERE id = ?
        `,
      )
      .get(id) as
      | {
          id: string;
          content: string;
          tags: string;
          source: string;
          created_at: number;
        }
      | undefined;

    return row ? toMemoryEntry(row) : undefined;
  }

  delete(id: string): boolean {
    const result = this.db.prepare("DELETE FROM memories WHERE id = ?").run(id) as { changes: number };
    return result.changes > 0;
  }

  list(limit = 100): MemoryEntry[] {
    const safeLimit = Math.max(1, Math.trunc(limit));
    const rows = this.db
      .prepare(
        `
          SELECT id, content, tags, source, created_at
          FROM memories
          ORDER BY created_at DESC
          LIMIT ?
        `,
      )
      .all(safeLimit) as Array<{
      id: string;
      content: string;
      tags: string;
      source: string;
      created_at: number;
    }>;

    return rows.map(toMemoryEntry);
  }

  getCoreMemory(key: string): string | undefined {
    const row = this.db.prepare("SELECT value FROM core_memory WHERE key = ?").get(key) as { value: string } | undefined;
    return row?.value;
  }

  setCoreMemory(key: string, value: string): void {
    this.db
      .prepare(
        `
          INSERT INTO core_memory(key, value, updated_at)
          VALUES (?, ?, ?)
          ON CONFLICT(key) DO UPDATE SET
            value = excluded.value,
            updated_at = excluded.updated_at
        `,
      )
      .run(key, value, now());
  }

  deleteCoreMemory(key: string): void {
    this.db.prepare("DELETE FROM core_memory WHERE key = ?").run(key);
  }

  listCoreMemory(): CoreMemoryEntry[] {
    const rows = this.db
      .prepare(
        `
          SELECT key, value, updated_at
          FROM core_memory
          ORDER BY key ASC
        `,
      )
      .all() as Array<{ key: string; value: string; updated_at: number }>;

    return rows.map((row) => ({ key: row.key, value: row.value, updatedAt: row.updated_at }));
  }

  close(): void {
    this.db.close();
  }

  private applyPragmas(): void {
    this.db.exec("PRAGMA journal_mode = WAL;");
    this.db.exec("PRAGMA foreign_keys = ON;");
    this.db.exec("PRAGMA busy_timeout = 5000;");
  }

  private initSchemaVersion(): void {
    this.db.exec("CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY);");
    const row = this.db.prepare("SELECT MAX(version) AS version FROM schema_version").get() as { version: number | null };
    const currentVersion = row.version ?? 0;

    if (currentVersion === 0) {
      this.db.prepare("INSERT INTO schema_version(version) VALUES (?)").run(SCHEMA_VERSION);
      return;
    }

    if (currentVersion !== SCHEMA_VERSION) {
      throw new Error(`unsupported memory schema version: ${currentVersion}`);
    }
  }

  private initTables(): void {
    this.db.exec(`
      CREATE TABLE IF NOT EXISTS memories (
        id TEXT PRIMARY KEY,
        content TEXT NOT NULL,
        tags TEXT DEFAULT '[]',
        source TEXT DEFAULT '',
        created_at INTEGER NOT NULL
      );

      CREATE TABLE IF NOT EXISTS core_memory (
        key TEXT PRIMARY KEY,
        value TEXT NOT NULL,
        updated_at INTEGER NOT NULL
      );

      CREATE INDEX IF NOT EXISTS idx_memories_created_at ON memories(created_at DESC);
    `);
  }

  private tryInitFts(): boolean {
    try {
      this.db.exec(`
        CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts
        USING fts5(content, content='memories', content_rowid='rowid');

        CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
          INSERT INTO memories_fts(rowid, content) VALUES (new.rowid, new.content);
        END;

        CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
          INSERT INTO memories_fts(memories_fts, rowid, content) VALUES ('delete', old.rowid, old.content);
        END;

        CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
          INSERT INTO memories_fts(memories_fts, rowid, content) VALUES ('delete', old.rowid, old.content);
          INSERT INTO memories_fts(rowid, content) VALUES (new.rowid, new.content);
        END;
      `);

      this.db.exec(
        "INSERT INTO memories_fts(memories_fts) VALUES ('rebuild');",
      );

      return true;
    } catch {
      return false;
    }
  }
}
