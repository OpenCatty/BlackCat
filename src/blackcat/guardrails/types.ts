export interface GuardrailResult {
  allow: boolean;
  reason?: string;
  modified?: string;
}

export interface GuardrailsConfig {
  inputEnabled?: boolean;
  toolEnabled?: boolean;
  outputEnabled?: boolean;
  denyPatterns?: string[];
  requireApprovalPatterns?: string[];
}
