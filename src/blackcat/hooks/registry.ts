import type { Hook, HookContext, HookPhase, HookResult } from "./types.js";

export class HookRegistry {
  private readonly hooks: Hook[] = [];

  /** Register a hook. Hooks run in registration order within their phase. */
  register(hook: Hook): void {
    this.hooks.push(hook);
  }

  /**
   * Run all hooks for the given phase, threading the context through each.
   * Hooks that throw are logged and skipped — they never crash the pipeline.
   */
  async run(phase: HookPhase, ctx: HookContext): Promise<HookContext> {
    let current = { ...ctx, metadata: { ...ctx.metadata } };

    for (const hook of this.hooks) {
      if (hook.phase !== phase) continue;

      try {
        const result: HookResult = await hook.execute(current);
        current = { ...result.context, metadata: { ...result.context.metadata } };

        if (result.halt) {
          break;
        }
      } catch (err) {
        // Log but continue — hooks must never crash the pipeline.
        console.error(
          `[hooks] hook "${hook.name}" (${phase}) threw:`,
          err instanceof Error ? err.message : String(err),
        );
      }
    }

    return current;
  }

  /** List registered hooks (name + phase). */
  list(): Array<{ name: string; phase: HookPhase }> {
    return this.hooks.map((h) => ({ name: h.name, phase: h.phase }));
  }
}
