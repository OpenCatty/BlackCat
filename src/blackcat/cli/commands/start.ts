import { Command } from "commander";
import { readFileSync, writeFileSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { loadConfig } from "../../config/loader.js";

function writePid(configDir: string): string {
  const pidPath = resolve(configDir, ".blackcat.pid");
  writeFileSync(pidPath, String(process.pid), "utf-8");
  return pidPath;
}

export const startCommand = new Command("start")
  .description("Start the BlackCat daemon")
  .option("-c, --config <path>", "Path to blackcat.json5 config file", "blackcat.json5")
  .action(async (opts: { config: string }) => {
    const configPath = resolve(opts.config);
    let config;
    try {
      config = loadConfig(configPath);
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      process.stderr.write(`blackcat: failed to load config: ${msg}\n`);
      process.exitCode = 1;
      return;
    }

    const configDir = dirname(configPath);
    const pidPath = writePid(configDir);

    process.stdout.write(`BlackCat starting... (PID ${process.pid})\n`);
    process.stdout.write(`Config: ${configPath}\n`);
    process.stdout.write(`PID file: ${pidPath}\n`);
    process.stdout.write(`LLM provider: ${config.llm.provider} / ${config.llm.model}\n`);

    const channels: string[] = [];
    if (config.channels?.telegram?.enabled) channels.push("telegram");
    if (config.channels?.discord?.enabled) channels.push("discord");
    if (config.channels?.whatsapp?.enabled) channels.push("whatsapp");

    if (channels.length > 0) {
      process.stdout.write(`Channels: ${channels.join(", ")}\n`);
    } else {
      process.stdout.write("Channels: none configured\n");
    }

    // Graceful shutdown handler
    const shutdown = () => {
      process.stdout.write("\nBlackCat shutting down...\n");
      try {
        const { unlinkSync } = require("node:fs") as typeof import("node:fs");
        unlinkSync(pidPath);
      } catch {
        // PID file may already be removed
      }
      process.exit(0);
    };

    process.on("SIGINT", shutdown);
    process.on("SIGTERM", shutdown);

    // TODO: Initialize channel adapters and start message loop
    // For now, the daemon skeleton is complete. In production this would:
    // 1. Set up Telegram/Discord/WhatsApp adapters
    // 2. Initialize the Supervisor with router + subagents
    // 3. Start the event loop
    process.stdout.write("BlackCat daemon ready. Press Ctrl+C to stop.\n");
  });
