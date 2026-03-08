import { Command } from "commander";
import { startCommand } from "./commands/start.js";
import { stopCommand } from "./commands/stop.js";
import { statusCommand } from "./commands/status.js";
import { doctorCommand } from "./commands/doctor.js";
import { migrateConfigCommand } from "./commands/migrate-config.js";
import { onboardCommand } from "./commands/onboard.js";

export function createProgram(): Command {
  const program = new Command("blackcat")
    .description("BlackCat — multi-channel AI agent daemon")
    .version("0.1.0");

  program.addCommand(startCommand);
  program.addCommand(stopCommand);
  program.addCommand(statusCommand);
  program.addCommand(doctorCommand);
  program.addCommand(migrateConfigCommand);
  program.addCommand(onboardCommand);

  return program;
}

// When run directly, parse argv
const isDirectRun = process.argv[1] &&
  (process.argv[1].endsWith("/cli/index.ts") || process.argv[1].endsWith("\\cli\\index.ts"));

if (isDirectRun) {
  const program = createProgram();
  program.parseAsync(process.argv).catch((err: unknown) => {
    const msg = err instanceof Error ? err.message : String(err);
    process.stderr.write(`blackcat: ${msg}\n`);
    process.exit(1);
  });
}
