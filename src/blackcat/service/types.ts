/** Supported service management platforms. */
export type ServicePlatform = "systemd" | "launchd" | "winschtask";

/** Configuration describing how to install the daemon as a system service. */
export interface ServiceConfig {
  /** Unique service name (e.g. "blackcat"). */
  name: string;
  /** Human-readable description of the service. */
  description: string;
  /** Absolute path to the blackcat entry script (blackcat.mjs). */
  execPath: string;
  /** Absolute path to the Node.js binary used to run the script. */
  nodeExecPath: string;
  /** Working directory for the service process. */
  workingDirectory: string;
  /** OS user to run the service as (optional, platform-dependent). */
  user?: string;
  /** Path for service log output (optional). */
  logPath?: string;
}

/** A generated service unit ready for installation. */
export interface ServiceUnit {
  /** The platform this unit targets. */
  platform: ServicePlatform;
  /** The rendered unit content (INI, plist XML, or task XML). */
  content: string;
  /** Suggested file path where the unit should be installed. */
  installPath: string;
}
