import { argon2Sync, createCipheriv, createDecipheriv, randomBytes } from "node:crypto";
import { existsSync, mkdirSync, readFileSync, renameSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";

const VAULT_SALT_SIZE = 16;
const VAULT_NONCE_SIZE = 12;
const VAULT_KEY_LEN = 32;
const VAULT_TIME = 1;
const VAULT_MEMORY = 64 * 1024;
const VAULT_THREADS = 4;
const GCM_TAG_LEN = 16;

export interface VaultFile {
  salt: string;
  nonce: string;
  data: string;
}

function deriveKey(passphrase: string, salt: Buffer): Buffer {
  return argon2Sync("argon2id", {
    message: passphrase,
    nonce: salt,
    parallelism: VAULT_THREADS,
    memory: VAULT_MEMORY,
    passes: VAULT_TIME,
    tagLength: VAULT_KEY_LEN,
  });
}

function parseBase64(field: "salt" | "nonce" | "data", encoded: string): Buffer {
  try {
    return Buffer.from(encoded, "base64");
  } catch {
    throw new Error(`vault has invalid base64 in field: ${field}`);
  }
}

function readExistingSalt(path: string): Buffer | undefined {
  if (!existsSync(path)) {
    return undefined;
  }

  const raw = readFileSync(path, "utf-8");
  const disk = JSON.parse(raw) as Partial<VaultFile>;
  if (typeof disk.salt !== "string") {
    throw new Error("invalid vault file: expected salt, nonce, data");
  }

  const salt = parseBase64("salt", disk.salt);
  if (salt.length !== VAULT_SALT_SIZE) {
    throw new Error(`invalid salt size: ${salt.length}`);
  }

  return salt;
}

export class Vault {
  private readonly entries = new Map<string, string>();

  constructor(
    private readonly path: string,
    private readonly passphrase: string,
  ) {}

  async load(): Promise<void> {
    if (!existsSync(this.path)) {
      this.entries.clear();
      return;
    }

    const raw = readFileSync(this.path, "utf-8");
    const disk = JSON.parse(raw) as VaultFile;

    if (typeof disk.salt !== "string" || typeof disk.nonce !== "string" || typeof disk.data !== "string") {
      throw new Error("invalid vault file: expected salt, nonce, data");
    }

    const salt = parseBase64("salt", disk.salt);
    if (salt.length !== VAULT_SALT_SIZE) {
      throw new Error(`invalid salt size: ${salt.length}`);
    }

    const nonce = parseBase64("nonce", disk.nonce);
    if (nonce.length !== VAULT_NONCE_SIZE) {
      throw new Error(`invalid nonce size: ${nonce.length}`);
    }

    const sealed = parseBase64("data", disk.data);
    if (sealed.length < GCM_TAG_LEN) {
      throw new Error("invalid vault data: ciphertext too short");
    }

    const ciphertext = sealed.subarray(0, sealed.length - GCM_TAG_LEN);
    const authTag = sealed.subarray(sealed.length - GCM_TAG_LEN);

    const key = deriveKey(this.passphrase, salt);

    try {
      const decipher = createDecipheriv("aes-256-gcm", key, nonce);
      decipher.setAuthTag(authTag);
      const plaintext = Buffer.concat([decipher.update(ciphertext), decipher.final()]);
      const parsed = plaintext.length > 0 ? (JSON.parse(plaintext.toString("utf-8")) as Record<string, string>) : {};

      this.entries.clear();
      for (const [entryKey, value] of Object.entries(parsed)) {
        this.entries.set(entryKey, value);
      }
    } catch {
      throw new Error("invalid passphrase or corrupted vault data");
    } finally {
      key.fill(0);
    }
  }

  async save(): Promise<void> {
    mkdirSync(dirname(this.path), { recursive: true, mode: 0o700 });

    const salt = readExistingSalt(this.path) ?? randomBytes(VAULT_SALT_SIZE);
    const key = deriveKey(this.passphrase, salt);
    const nonce = randomBytes(VAULT_NONCE_SIZE);
    const plaintext = Buffer.from(JSON.stringify(Object.fromEntries(this.entries)), "utf-8");

    try {
      const cipher = createCipheriv("aes-256-gcm", key, nonce);
      const ciphertext = Buffer.concat([cipher.update(plaintext), cipher.final()]);
      const authTag = cipher.getAuthTag();
      const sealed = Buffer.concat([ciphertext, authTag]);

      const disk: VaultFile = {
        salt: salt.toString("base64"),
        nonce: nonce.toString("base64"),
        data: sealed.toString("base64"),
      };

      const tmpPath = `${this.path}.tmp`;
      writeFileSync(tmpPath, JSON.stringify(disk), { mode: 0o600 });
      renameSync(tmpPath, this.path);
    } finally {
      key.fill(0);
    }
  }

  get(key: string): string | undefined {
    return this.entries.get(key);
  }

  set(key: string, value: string): void {
    this.entries.set(key, value);
  }

  delete(key: string): void {
    this.entries.delete(key);
  }

  list(): string[] {
    return [...this.entries.keys()].sort();
  }
}
