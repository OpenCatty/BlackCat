import { Command } from "commander";
import { createInterface } from "node:readline";
import { writeFileSync } from "node:fs";
import { resolve } from "node:path";

interface OnboardAnswers {
  provider: string;
  apiKey: string;
  model: string;
  configPath: string;
}

function prompt(rl: ReturnType<typeof createInterface>, question: string, defaultValue?: string): Promise<string> {
  const suffix = defaultValue ? ` [${defaultValue}]` : "";
  return new Promise((res) => {
    rl.question(`${question}${suffix}: `, (answer: string) => {
      res(answer.trim() || defaultValue || "");
    });
  });
}

async function runInteractiveOnboard(): Promise<OnboardAnswers> {
  const rl = createInterface({
    input: process.stdin,
    output: process.stdout,
  });

  try {
    process.stdout.write("\nBlackCat Onboarding Wizard\n");
    process.stdout.write("=========================\n\n");

    const provider = await prompt(rl, "LLM provider (openai, anthropic, gemini)", "openai");
    const apiKey = await prompt(rl, `API key for ${provider}`);
    const model = await prompt(rl, "Model name", provider === "anthropic" ? "claude-sonnet-4-20250514" : "gpt-4o");
    const configPath = await prompt(rl, "Config output path", "blackcat.json5");

    return { provider, apiKey, model, configPath };
  } finally {
    rl.close();
  }
}

function generateConfig(answers: OnboardAnswers): string {
  const config = {
    llm: {
      provider: answers.provider,
      model: answers.model,
      ...(answers.apiKey ? { apiKey: answers.apiKey } : {}),
    },
    roles: [
      { name: "wizard", priority: 30, keywords: ["code", "coding", "debug", "build", "test"] },
      { name: "explorer", priority: 60, keywords: ["research", "search", "find", "analyze"] },
      { name: "oracle", priority: 100, keywords: [] },
    ],
  };

  // Use JSON with 2-space indent (compatible with JSON5)
  return JSON.stringify(config, null, 2);
}

export const onboardCommand = new Command("onboard")
  .description("Interactive first-run setup wizard")
  .option("--non-interactive", "Skip interactive prompts (for CI/testing)")
  .action(async (opts: { nonInteractive?: boolean }) => {
    const isInteractive = !opts.nonInteractive && process.stdin.isTTY;

    if (!isInteractive) {
      process.stdout.write("blackcat: non-interactive mode — skipping onboard wizard\n");
      process.stdout.write("To run interactively, use a TTY terminal without --non-interactive\n");
      return;
    }

    const answers = await runInteractiveOnboard();

    if (!answers.apiKey) {
      process.stderr.write("blackcat: API key is required\n");
      process.exitCode = 1;
      return;
    }

    const configContent = generateConfig(answers);
    const outputPath = resolve(answers.configPath);

    try {
      writeFileSync(outputPath, configContent, "utf-8");
      process.stdout.write(`\nConfig written to ${outputPath}\n`);
      process.stdout.write("Run 'blackcat doctor' to verify your setup.\n");
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err);
      process.stderr.write(`blackcat: failed to write config: ${msg}\n`);
      process.exitCode = 1;
    }
  });
