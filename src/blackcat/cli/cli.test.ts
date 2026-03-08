import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { mkdirSync, writeFileSync, rmSync, existsSync } from "node:fs";
import { join } from "node:path";
import { tmpdir } from "node:os";
import { createProgram } from "./index.js";
import { doctorCommand } from "./commands/doctor.js";
import { migrateConfigCommand } from "./commands/migrate-config.js";

// Minimal valid config for testing
const MINIMAL_CONFIG = JSON.stringify({
  llm: { provider: "openai", model: "gpt-4o" },
  roles: [
    { name: "wizard", priority: 30, keywords: ["code"] },
    { name: "oracle", priority: 100, keywords: [] },
  ],
});

function makeTmpDir(): string {
  const dir = join(tmpdir(), `blackcat-cli-test-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`);
  mkdirSync(dir, { recursive: true });
  return dir;
}

describe("CLI index", () => {
  it("--help output is non-empty", async () => {
    const program = createProgram();
    let helpText = "";
    program.configureOutput({
      writeOut: (str: string) => { helpText += str; },
      writeErr: (str: string) => { helpText += str; },
    });
    program.exitOverride();

    try {
      await program.parseAsync(["node", "blackcat", "--help"]);
    } catch {
      // Commander throws on --help with exitOverride
    }

    expect(helpText.length).toBeGreaterThan(0);
    expect(helpText).toContain("blackcat");
  });

  it("registers all expected commands", () => {
    const program = createProgram();
    const commandNames = program.commands.map((cmd) => cmd.name());

    expect(commandNames).toContain("start");
    expect(commandNames).toContain("stop");
    expect(commandNames).toContain("status");
    expect(commandNames).toContain("doctor");
    expect(commandNames).toContain("migrate-config");
    expect(commandNames).toContain("onboard");
  });

  it("version is set", () => {
    const program = createProgram();
    expect(program.version()).toBe("0.1.0");
  });
});

describe("doctor command", () => {
  it("is a registered command object", () => {
    expect(doctorCommand.name()).toBe("doctor");
    expect(doctorCommand.description()).toContain("health");
  });

  it("reports failure when config file does not exist", async () => {
    let output = "";
    const originalWrite = process.stdout.write;
    const originalErrWrite = process.stderr.write;
    const originalExitCode = process.exitCode;

    process.stdout.write = ((str: string) => { output += str; return true; }) as typeof process.stdout.write;
    process.stderr.write = ((str: string) => { output += str; return true; }) as typeof process.stderr.write;

    const program = createProgram();
    program.exitOverride();

    try {
      await program.parseAsync(["node", "blackcat", "doctor", "-c", "/nonexistent/blackcat.json5"]);
    } catch {
      // may throw on exitOverride
    }

    process.stdout.write = originalWrite;
    process.stderr.write = originalErrWrite;

    expect(output).toContain("Config file readable");
    expect(output).toContain("\u2717"); // ✗ failure marker

    process.exitCode = originalExitCode;
  });

  it("reports success with valid config", async () => {
    const tmpDir = makeTmpDir();
    const configPath = join(tmpDir, "blackcat.json5");
    writeFileSync(configPath, MINIMAL_CONFIG, "utf-8");

    let output = "";
    const originalWrite = process.stdout.write;
    const originalErrWrite = process.stderr.write;
    const originalExitCode = process.exitCode;

    process.stdout.write = ((str: string) => { output += str; return true; }) as typeof process.stdout.write;
    process.stderr.write = ((str: string) => { output += str; return true; }) as typeof process.stderr.write;

    const program = createProgram();
    program.exitOverride();

    try {
      await program.parseAsync(["node", "blackcat", "doctor", "-c", configPath]);
    } catch {
      // may throw on exitOverride
    }

    process.stdout.write = originalWrite;
    process.stderr.write = originalErrWrite;

    expect(output).toContain("Config file readable");
    expect(output).toContain("Config JSON5 valid");
    expect(output).toContain("\u2713"); // ✓ success marker

    process.exitCode = originalExitCode;
    rmSync(tmpDir, { recursive: true, force: true });
  });
});

describe("migrate-config command", () => {
  it("is a registered command object", () => {
    expect(migrateConfigCommand.name()).toBe("migrate-config");
  });

  it("fails gracefully without input file", async () => {
    let errOutput = "";
    const originalErrWrite = process.stderr.write;
    const originalExitCode = process.exitCode;

    process.stderr.write = ((str: string) => { errOutput += str; return true; }) as typeof process.stderr.write;

    const program = createProgram();
    program.exitOverride();

    try {
      await program.parseAsync(["node", "blackcat", "migrate-config"]);
    } catch {
      // may throw on exitOverride
    }

    process.stderr.write = originalErrWrite;

    expect(errOutput).toContain("no input file");

    process.exitCode = originalExitCode;
  });

  it("converts valid YAML to JSON5 output", async () => {
    const tmpDir = makeTmpDir();
    const yamlPath = join(tmpDir, "blackcat.yaml");
    writeFileSync(yamlPath, [
      "llm:",
      "  provider: openai",
      "  model: gpt-4o",
      "  api_key: sk-test123",
      "roles:",
      "  - name: oracle",
      "    priority: 100",
      "    keywords: []",
    ].join("\n"), "utf-8");

    let stdOutput = "";
    const originalWrite = process.stdout.write;
    const originalExitCode = process.exitCode;

    process.stdout.write = ((str: string) => { stdOutput += str; return true; }) as typeof process.stdout.write;

    const program = createProgram();
    program.exitOverride();

    try {
      await program.parseAsync(["node", "blackcat", "migrate-config", yamlPath]);
    } catch {
      // may throw on exitOverride
    }

    process.stdout.write = originalWrite;

    // Should have camelCase apiKey, not api_key
    expect(stdOutput).toContain("apiKey");
    expect(stdOutput).not.toContain("api_key");
    expect(stdOutput).toContain("openai");

    process.exitCode = originalExitCode;
    rmSync(tmpDir, { recursive: true, force: true });
  });

  it("writes output to file when -o specified", async () => {
    const tmpDir = makeTmpDir();
    const yamlPath = join(tmpDir, "blackcat.yaml");
    const outPath = join(tmpDir, "blackcat.json5");
    writeFileSync(yamlPath, [
      "llm:",
      "  provider: anthropic",
      "  model: claude-sonnet-4-20250514",
      "roles:",
      "  - name: oracle",
      "    priority: 100",
      "    keywords: []",
    ].join("\n"), "utf-8");

    let stdOutput = "";
    const originalWrite = process.stdout.write;
    const originalExitCode = process.exitCode;

    process.stdout.write = ((str: string) => { stdOutput += str; return true; }) as typeof process.stdout.write;

    const program = createProgram();
    program.exitOverride();

    try {
      await program.parseAsync(["node", "blackcat", "migrate-config", yamlPath, "-o", outPath]);
    } catch {
      // may throw on exitOverride
    }

    process.stdout.write = originalWrite;

    expect(existsSync(outPath)).toBe(true);
    expect(stdOutput).toContain("Migrated config written to");

    process.exitCode = originalExitCode;
    rmSync(tmpDir, { recursive: true, force: true });
  });
});

describe("status command", () => {
  it("reports stopped when no PID file", async () => {
    let output = "";
    const originalWrite = process.stdout.write;
    const originalExitCode = process.exitCode;

    process.stdout.write = ((str: string) => { output += str; return true; }) as typeof process.stdout.write;

    const program = createProgram();
    program.exitOverride();

    try {
      await program.parseAsync(["node", "blackcat", "status", "-c", "/nonexistent/blackcat.json5"]);
    } catch {
      // may throw on exitOverride
    }

    process.stdout.write = originalWrite;

    expect(output).toContain("stopped");

    process.exitCode = originalExitCode;
  });
});

describe("onboard command", () => {
  it("skips in non-interactive mode", async () => {
    let output = "";
    const originalWrite = process.stdout.write;
    const originalExitCode = process.exitCode;

    process.stdout.write = ((str: string) => { output += str; return true; }) as typeof process.stdout.write;

    const program = createProgram();
    program.exitOverride();

    try {
      await program.parseAsync(["node", "blackcat", "onboard", "--non-interactive"]);
    } catch {
      // may throw on exitOverride
    }

    process.stdout.write = originalWrite;

    expect(output).toContain("non-interactive");

    process.exitCode = originalExitCode;
  });
});

describe("stop command", () => {
  it("fails gracefully when no PID file exists", async () => {
    let errOutput = "";
    const originalErrWrite = process.stderr.write;
    const originalExitCode = process.exitCode;

    process.stderr.write = ((str: string) => { errOutput += str; return true; }) as typeof process.stderr.write;

    const program = createProgram();
    program.exitOverride();

    try {
      await program.parseAsync(["node", "blackcat", "stop", "-c", "/nonexistent/blackcat.json5"]);
    } catch {
      // may throw on exitOverride
    }

    process.stderr.write = originalErrWrite;

    expect(errOutput).toContain("no PID file");

    process.exitCode = originalExitCode;
  });
});
