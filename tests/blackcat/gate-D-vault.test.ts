import { mkdtemp, rm } from "node:fs/promises";
import { readFileSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { Vault } from "../../src/blackcat/security/vault.js";

let tmpDir: string;

beforeEach(async () => {
  tmpDir = await mkdtemp(path.join(os.tmpdir(), "gate-d-vault-"));
});

afterEach(async () => {
  await rm(tmpDir, { recursive: true, force: true });
});

describe("Gate D — Vault", () => {
  it("set/get roundtrip", async () => {
    const vaultPath = path.join(tmpDir, "vault.json");
    const vault = new Vault(vaultPath, "test-passphrase");

    vault.set("api_key", "sk-secret-123");
    await vault.save();

    // Re-read from disk
    const vault2 = new Vault(vaultPath, "test-passphrase");
    await vault2.load();
    expect(vault2.get("api_key")).toBe("sk-secret-123");
  });

  it("persistence across re-open (multiple keys)", async () => {
    const vaultPath = path.join(tmpDir, "vault.json");

    // First session: write multiple keys
    const v1 = new Vault(vaultPath, "pass");
    v1.set("key1", "value1");
    v1.set("key2", "value2");
    await v1.save();

    // Second session: re-open and verify
    const v2 = new Vault(vaultPath, "pass");
    await v2.load();
    expect(v2.get("key1")).toBe("value1");
    expect(v2.get("key2")).toBe("value2");
  });

  it("missing key returns undefined", async () => {
    const vaultPath = path.join(tmpDir, "vault.json");
    const vault = new Vault(vaultPath, "pass");
    await vault.load(); // no file exists yet
    expect(vault.get("nonexistent")).toBeUndefined();
  });

  it("wrong passphrase throws on load", async () => {
    const vaultPath = path.join(tmpDir, "vault.json");

    const v1 = new Vault(vaultPath, "correct-pass");
    v1.set("secret", "data");
    await v1.save();

    const v2 = new Vault(vaultPath, "wrong-pass");
    await expect(v2.load()).rejects.toThrow(/passphrase|corrupt/i);
  });

  it("AES-256-GCM format verifiable on disk", async () => {
    const vaultPath = path.join(tmpDir, "vault.json");
    const vault = new Vault(vaultPath, "test-pass");
    vault.set("hello", "world");
    await vault.save();

    const raw = readFileSync(vaultPath, "utf-8");
    const disk = JSON.parse(raw) as { salt: string; nonce: string; data: string };

    // Verify structure
    expect(disk).toHaveProperty("salt");
    expect(disk).toHaveProperty("nonce");
    expect(disk).toHaveProperty("data");

    // Verify base64 encoding (should be valid base64)
    const salt = Buffer.from(disk.salt, "base64");
    const nonce = Buffer.from(disk.nonce, "base64");
    const data = Buffer.from(disk.data, "base64");

    // Salt = 16 bytes, Nonce = 12 bytes
    expect(salt.length).toBe(16);
    expect(nonce.length).toBe(12);
    // Data = ciphertext + 16 byte GCM tag (at least)
    expect(data.length).toBeGreaterThanOrEqual(16);
  });

  it("delete removes a key", async () => {
    const vaultPath = path.join(tmpDir, "vault.json");
    const vault = new Vault(vaultPath, "pass");
    vault.set("key1", "val1");
    vault.set("key2", "val2");
    vault.delete("key1");
    await vault.save();

    const v2 = new Vault(vaultPath, "pass");
    await v2.load();
    expect(v2.get("key1")).toBeUndefined();
    expect(v2.get("key2")).toBe("val2");
  });

  it("list returns sorted keys", async () => {
    const vaultPath = path.join(tmpDir, "vault.json");
    const vault = new Vault(vaultPath, "pass");
    vault.set("charlie", "3");
    vault.set("alpha", "1");
    vault.set("bravo", "2");
    expect(vault.list()).toEqual(["alpha", "bravo", "charlie"]);
  });
});
