import { Command } from "commander";
import { readFileSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";
import { migrateYamlToJson5 } from "../../config/migrator.js";

export const migrateConfigCommand = new Command("migrate-config")
  .description("Convert Go YAML blackcat.yaml to JSON5 blackcat.json5")
  .argument("[input]", "Path to blackcat.yaml input file")
  .option("-o, --output <path>", "Output path for blackcat.json5")
  .action((input: string | undefined, opts: { output?: string }) => {
    if (!input) {
      process.stderr.write("blackcat: no input file specified\n");
      process.stderr.write("Usage: blackcat migrate-config <blackcat.yaml> [-o output.json5]\n");
      process.exitCode = 1;
      return;
    }

    const inputPath = resolve(input);
    let yamlContent: string;
    try {
      yamlContent = readFileSync(inputPath, "utf-8");
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      process.stderr.write(`blackcat: cannot read input file: ${msg}\n`);
      process.exitCode = 1;
      return;
    }

    let json5Output: string;
    try {
      json5Output = migrateYamlToJson5(yamlContent);
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      process.stderr.write(`blackcat: migration failed: ${msg}\n`);
      process.exitCode = 1;
      return;
    }

    if (opts.output) {
      const outputPath = resolve(opts.output);
      try {
        writeFileSync(outputPath, json5Output, "utf-8");
        process.stdout.write(`Migrated config written to ${outputPath}\n`);
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        process.stderr.write(`blackcat: cannot write output file: ${msg}\n`);
        process.exitCode = 1;
      }
    } else {
      // Print to stdout if no output file specified
      process.stdout.write(json5Output);
      process.stdout.write("\n");
    }
  });
