import { randomUUID } from "node:crypto";
import { existsSync, mkdirSync, readFileSync, rmSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { afterEach, describe, expect, it } from "vitest";

import { Vault, type VaultFile } from "./vault.js";

function isBase64(value: string): boolean {
  return Buffer.from(value, "base64").toString("base64") === value;
}

const tempRoots: string[] = [];

function makeTempPath(): string {
  const root = join(tmpdir(), `blackcat-vault-${randomUUID()}`);
  mkdirSync(root, { recursive: true });
  tempRoots.push(root);
  return join(root, "vault.json");
}

afterEach(() => {
  while (tempRoots.length > 0) {
    const root = tempRoots.pop();
    if (root && existsSync(root)) {
      rmSync(root, { recursive: true, force: true });
    }
  }
});

describe("blackcat vault Go-compatible format", () => {
  it("round-trips TS encrypt/decrypt", async () => {
    const vaultPath = makeTempPath();
    const vault = new Vault(vaultPath, "correct horse battery staple");

    vault.set("token", "secret123");
    await vault.save();

    const reloaded = new Vault(vaultPath, "correct horse battery staple");
    await reloaded.load();

    expect(reloaded.get("token")).toBe("secret123");
  });

  it("output format matches Go vault spec", async () => {
    const vaultPath = makeTempPath();
    const vault = new Vault(vaultPath, "vault-passphrase");

    vault.set("k", "v");
    await vault.save();

    const raw = readFileSync(vaultPath, "utf-8");
    const disk = JSON.parse(raw) as VaultFile;

    expect(typeof disk.salt).toBe("string");
    expect(typeof disk.nonce).toBe("string");
    expect(typeof disk.data).toBe("string");

    expect(isBase64(disk.salt)).toBe(true);
    expect(isBase64(disk.nonce)).toBe(true);
    expect(isBase64(disk.data)).toBe(true);

    expect(Buffer.from(disk.salt, "base64").length).toBe(16);
    expect(Buffer.from(disk.nonce, "base64").length).toBe(12);
    expect(Buffer.from(disk.data, "base64").length).toBeGreaterThan(16);
  });

  it("throws clear error on wrong passphrase", async () => {
    const vaultPath = makeTempPath();
    const vault = new Vault(vaultPath, "passphrase-a");
    vault.set("apiKey", "abc123");
    await vault.save();

    const wrong = new Vault(vaultPath, "passphrase-b");
    await expect(wrong.load()).rejects.toThrow("invalid passphrase or corrupted vault data");
  });

  it("persists and restores multiple entries", async () => {
    const vaultPath = makeTempPath();
    const vault = new Vault(vaultPath, "multi-pass");
    vault.set("k1", "v1");
    vault.set("k2", "v2");
    vault.set("k3", "v3");
    await vault.save();

    const reloaded = new Vault(vaultPath, "multi-pass");
    await reloaded.load();

    expect(reloaded.get("k1")).toBe("v1");
    expect(reloaded.get("k2")).toBe("v2");
    expect(reloaded.get("k3")).toBe("v3");
    expect(reloaded.list()).toEqual(["k1", "k2", "k3"]);
  });

  it("preserves salt across multiple saves to stay Go-compatible", async () => {
    const vaultPath = makeTempPath();
    const vault = new Vault(vaultPath, "stable-salt");
    vault.set("first", "1");
    await vault.save();

    const firstDisk = JSON.parse(readFileSync(vaultPath, "utf-8")) as VaultFile;

    vault.set("second", "2");
    await vault.save();

    const secondDisk = JSON.parse(readFileSync(vaultPath, "utf-8")) as VaultFile;
    expect(secondDisk.salt).toBe(firstDisk.salt);
  });
});
