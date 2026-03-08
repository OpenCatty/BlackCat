export type HookPhase = "pre" | "post";

export interface HookContext {
  /** The incoming message text (pre) or outgoing response text (post). */
  text: string;
  /** Session identifier, if available. */
  sessionId?: string;
  /** Arbitrary metadata hooks can read/write. */
  metadata: Record<string, unknown>;
}

export interface HookResult {
  /** The (potentially modified) context to pass downstream. */
  context: HookContext;
  /** If true the hook signals the pipeline should halt early. */
  halt?: boolean;
}

export interface Hook {
  /** Unique name for this hook (used for listing / debugging). */
  name: string;
  /** Which phase this hook runs in. */
  phase: HookPhase;
  /** Execute the hook. May return a modified context. */
  execute(ctx: HookContext): HookResult | Promise<HookResult>;
}
