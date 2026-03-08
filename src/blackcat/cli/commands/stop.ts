import { Command } from "commander";
import { existsSync, readFileSync, unlinkSync } from "node:fs";
import { resolve, dirname } from "node:path";

function findPidFile(configPath: string): string {
  return resolve(dirname(configPath), ".blackcat.pid");
}

function isProcessRunning(pid: number): boolean {
  try {
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
}

export const stopCommand = new Command("stop")
  .description("Stop the BlackCat daemon")
  .option("-c, --config <path>", "Path to blackcat.json5 config file", "blackcat.json5")
  .action((opts: { config: string }) => {
    const configPath = resolve(opts.config);
    const pidPath = findPidFile(configPath);

    if (!existsSync(pidPath)) {
      process.stderr.write("blackcat: no PID file found — daemon may not be running\n");
      process.exitCode = 1;
      return;
    }

    const pidStr = readFileSync(pidPath, "utf-8").trim();
    const pid = Number(pidStr);

    if (Number.isNaN(pid) || pid <= 0) {
      process.stderr.write(`blackcat: invalid PID in ${pidPath}: ${pidStr}\n`);
      unlinkSync(pidPath);
      process.exitCode = 1;
      return;
    }

    if (!isProcessRunning(pid)) {
      process.stdout.write(`blackcat: daemon (PID ${pid}) is not running — cleaning up stale PID file\n`);
      unlinkSync(pidPath);
      return;
    }

    try {
      process.kill(pid, "SIGTERM");
      process.stdout.write(`blackcat: sent SIGTERM to daemon (PID ${pid})\n`);
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      process.stderr.write(`blackcat: failed to stop daemon (PID ${pid}): ${msg}\n`);
      process.exitCode = 1;
      return;
    }

    // Clean up PID file after sending signal
    try {
      unlinkSync(pidPath);
    } catch {
      // PID file may have been removed by the daemon itself
    }
  });
