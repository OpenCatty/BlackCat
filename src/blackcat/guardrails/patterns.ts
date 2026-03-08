const defaultInputPatterns = [
  String.raw`(?i)ignore\s+previous\s+instructions`,
  String.raw`(?i)system\s+prompt`,
  String.raw`(?i)jailbreak`,
  String.raw`(?i)forget\s+your\s+instructions`,
  String.raw`(?i)act\s+as`,
  String.raw`(?i)you\s+are\s+now`,
  String.raw`(?i)pretend\s+you\s+are`,
  String.raw`(?i)bypass`,
];

const defaultToolPatterns = [
  String.raw`(?i)\brm\s+-rf\b`,
  String.raw`(?i)\brm\s+-r\s+/\b`,
  String.raw`(?i)\bgit\s+push\s+--force\b`,
  String.raw`(?i)\bgit\s+push\s+-f\b`,
  String.raw`(?i)\bDROP\s+TABLE\b`,
  String.raw`(?i)\bDROP\s+DATABASE\b`,
  String.raw`(?i)\bformat\s+c:\b`,
  String.raw`(?i)\bmkfs\b`,
  String.raw`(?i)\bdd\s+if=`,
];

function normalizeInlineFlags(pattern: string): { source: string; flags: string } {
  if (pattern.startsWith("(?i)")) {
    return { source: pattern.slice(4), flags: "i" };
  }

  return { source: pattern, flags: "" };
}

export function safeCompilePattern(pattern: string): RegExp | null {
  try {
    const normalized = normalizeInlineFlags(pattern);
    return new RegExp(normalized.source, normalized.flags);
  } catch {
    return null;
  }
}

export function compilePatterns(patterns: readonly string[]): RegExp[] {
  const compiled: RegExp[] = [];
  for (const pattern of patterns) {
    const regex = safeCompilePattern(pattern);
    if (regex) {
      compiled.push(regex);
    }
  }

  return compiled;
}

export function buildInputDenyPatterns(customPatterns?: readonly string[]): RegExp[] {
  return compilePatterns([...defaultInputPatterns, ...(customPatterns ?? [])]);
}

export function buildToolApprovalPatterns(requireApprovalPatterns?: readonly string[]): RegExp[] {
  return compilePatterns([...defaultToolPatterns, ...(requireApprovalPatterns ?? [])]);
}
