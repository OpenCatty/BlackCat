import { Command } from "commander";
import { existsSync, readFileSync } from "node:fs";
import { resolve, dirname } from "node:path";

function isProcessRunning(pid: number): boolean {
  try {
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
}

export const statusCommand = new Command("status")
  .description("Show BlackCat daemon status")
  .option("-c, --config <path>", "Path to blackcat.json5 config file", "blackcat.json5")
  .action((opts: { config: string }) => {
    const configPath = resolve(opts.config);
    const pidPath = resolve(dirname(configPath), ".blackcat.pid");

    process.stdout.write(`Config path: ${configPath}\n`);
    process.stdout.write(`Config exists: ${existsSync(configPath) ? "yes" : "no"}\n`);
    process.stdout.write(`PID file: ${pidPath}\n`);

    if (!existsSync(pidPath)) {
      process.stdout.write("Status: stopped (no PID file)\n");
      return;
    }

    const pidStr = readFileSync(pidPath, "utf-8").trim();
    const pid = Number(pidStr);

    if (Number.isNaN(pid) || pid <= 0) {
      process.stdout.write(`Status: unknown (invalid PID: ${pidStr})\n`);
      return;
    }

    if (isProcessRunning(pid)) {
      process.stdout.write(`Status: running (PID ${pid})\n`);
    } else {
      process.stdout.write(`Status: stopped (stale PID ${pid})\n`);
    }
  });
