import { Command } from "commander";
import { existsSync, readFileSync, writeFileSync, unlinkSync } from "node:fs";
import { resolve } from "node:path";
import { DatabaseSync } from "node:sqlite";
import { loadConfig } from "../../config/loader.js";
import type { BlackCatConfig } from "../../config/types.js";

interface CheckResult {
  name: string;
  ok: boolean;
  detail: string;
}

function checkConfigReadable(configPath: string): CheckResult {
  try {
    readFileSync(configPath, "utf-8");
    return { name: "Config file readable", ok: true, detail: configPath };
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    return { name: "Config file readable", ok: false, detail: msg };
  }
}

function checkConfigParseable(configPath: string): CheckResult & { config?: BlackCatConfig } {
  try {
    const config = loadConfig(configPath);
    return { name: "Config JSON5 valid", ok: true, detail: "parsed and validated", config };
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    return { name: "Config JSON5 valid", ok: false, detail: msg };
  }
}

function checkVaultAccessible(config: BlackCatConfig | undefined): CheckResult {
  const vaultPath = config?.security?.vaultPath;
  if (!vaultPath) {
    return { name: "Vault accessible", ok: true, detail: "no vault configured (skipped)" };
  }
  const resolved = resolve(vaultPath);
  if (existsSync(resolved)) {
    return { name: "Vault accessible", ok: true, detail: resolved };
  }
  return { name: "Vault accessible", ok: false, detail: `vault file not found: ${resolved}` };
}

function checkSqliteWritable(config: BlackCatConfig | undefined): CheckResult {
  const dbPath = config?.memory?.sqlitePath;
  if (!dbPath) {
    return { name: "SQLite DB writable", ok: true, detail: "no sqlitePath configured (skipped)" };
  }

  const testPath = `${dbPath}.doctor-test`;
  try {
    const db = new DatabaseSync(testPath);
    db.exec("CREATE TABLE IF NOT EXISTS _doctor_check (id INTEGER PRIMARY KEY)");
    db.exec("INSERT INTO _doctor_check (id) VALUES (1)");
    db.exec("DROP TABLE _doctor_check");
    db.close();
    try {
      unlinkSync(testPath);
    } catch {
      // best effort cleanup
    }
    return { name: "SQLite DB writable", ok: true, detail: dbPath };
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    try {
      unlinkSync(testPath);
    } catch {
      // best effort cleanup
    }
    return { name: "SQLite DB writable", ok: false, detail: msg };
  }
}

function checkApiKeys(): CheckResult {
  const keys = ["OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GEMINI_API_KEY"];
  const present = keys.filter((k) => Boolean(process.env[k]));
  if (present.length > 0) {
    return { name: "API key env vars", ok: true, detail: present.join(", ") };
  }
  return {
    name: "API key env vars",
    ok: false,
    detail: `none found (checked: ${keys.join(", ")})`,
  };
}

export const doctorCommand = new Command("doctor")
  .description("Check BlackCat health: config, vault, DB, credentials")
  .option("-c, --config <path>", "Path to blackcat.json5 config file", "blackcat.json5")
  .action((opts: { config: string }) => {
    const configPath = resolve(opts.config);
    process.stdout.write("BlackCat Doctor\n");
    process.stdout.write("===============\n\n");

    const results: CheckResult[] = [];

    // 1. Config readable
    results.push(checkConfigReadable(configPath));

    // 2. Config parseable
    const parseResult = checkConfigParseable(configPath);
    results.push(parseResult);

    // 3. Vault accessible
    results.push(checkVaultAccessible(parseResult.config));

    // 4. SQLite writable
    results.push(checkSqliteWritable(parseResult.config));

    // 5. API keys
    results.push(checkApiKeys());

    // Print results
    let allOk = true;
    for (const result of results) {
      const icon = result.ok ? "\u2713" : "\u2717";
      process.stdout.write(`  ${icon} ${result.name}: ${result.detail}\n`);
      if (!result.ok) allOk = false;
    }

    process.stdout.write("\n");
    if (allOk) {
      process.stdout.write("All checks passed.\n");
    } else {
      process.stdout.write("Some checks failed. Review the output above.\n");
      process.exitCode = 1;
    }
  });
