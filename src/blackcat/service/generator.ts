import * as fs from "node:fs/promises";
import * as path from "node:path";

import type { ServiceConfig, ServicePlatform, ServiceUnit } from "./types.js";

/**
 * Generates and manages platform-specific service definitions so the BlackCat
 * daemon can be installed as a system service.
 */
export class ServiceGenerator {
  /**
   * Detect the service platform based on the current operating system.
   *
   * - `linux`  → `systemd`
   * - `darwin` → `launchd`
   * - `win32`  → `winschtask`
   */
  detectPlatform(): ServicePlatform {
    switch (process.platform) {
      case "linux":
        return "systemd";
      case "darwin":
        return "launchd";
      case "win32":
        return "winschtask";
      default:
        return "systemd"; // sensible fallback
    }
  }

  /**
   * Generate a service unit definition for the given config.
   *
   * @param config  Service configuration.
   * @param platform  Override platform detection (useful for cross-platform generation).
   */
  generateUnit(config: ServiceConfig, platform?: ServicePlatform): ServiceUnit {
    const target = platform ?? this.detectPlatform();

    switch (target) {
      case "systemd":
        return this.generateSystemd(config);
      case "launchd":
        return this.generateLaunchd(config);
      case "winschtask":
        return this.generateWinschtask(config);
    }
  }

  /**
   * Install a generated service unit to the appropriate system path.
   * Writes the unit content to disk at `unit.installPath`.
   */
  async installUnit(unit: ServiceUnit, _config: ServiceConfig): Promise<void> {
    const dir = path.dirname(unit.installPath);
    await fs.mkdir(dir, { recursive: true });
    await fs.writeFile(unit.installPath, unit.content, "utf-8");
  }

  /**
   * Remove a previously installed service unit from disk.
   */
  async uninstallUnit(config: ServiceConfig, platform?: ServicePlatform): Promise<void> {
    const target = platform ?? this.detectPlatform();
    const installPath = this.getInstallPath(config.name, target);

    try {
      await fs.unlink(installPath);
    } catch (err: unknown) {
      if ((err as NodeJS.ErrnoException).code !== "ENOENT") {
        throw err;
      }
      // Already removed — nothing to do.
    }
  }

  // ---------------------------------------------------------------------------
  // Private helpers
  // ---------------------------------------------------------------------------

  private getInstallPath(name: string, platform: ServicePlatform): string {
    switch (platform) {
      case "systemd":
        return `/etc/systemd/system/${name}.service`;
      case "launchd":
        return `/Library/LaunchDaemons/com.startower.${name}.plist`;
      case "winschtask":
        return `C:\\ProgramData\\${name}\\${name}-task.xml`;
    }
  }

  private generateSystemd(config: ServiceConfig): ServiceUnit {
    const lines: string[] = [
      "[Unit]",
      `Description=${config.description}`,
      "After=network.target",
      "",
      "[Service]",
      "Type=simple",
      `ExecStart=${config.nodeExecPath} ${config.execPath}`,
      `WorkingDirectory=${config.workingDirectory}`,
      `SyslogIdentifier=${config.name}`,
      "Restart=on-failure",
      "RestartSec=5",
    ];

    if (config.user) {
      lines.push(`User=${config.user}`);
    }
    if (config.logPath) {
      lines.push(`StandardOutput=append:${config.logPath}`);
      lines.push(`StandardError=append:${config.logPath}`);
    }

    lines.push("", "[Install]", "WantedBy=multi-user.target", "");

    return {
      platform: "systemd",
      content: lines.join("\n"),
      installPath: this.getInstallPath(config.name, "systemd"),
    };
  }

  private generateLaunchd(config: ServiceConfig): ServiceUnit {
    const logEntries = config.logPath
      ? [
          `  <key>StandardOutPath</key>`,
          `  <string>${config.logPath}</string>`,
          `  <key>StandardErrorPath</key>`,
          `  <string>${config.logPath}</string>`,
        ]
      : [];

    const userEntries = config.user
      ? [`  <key>UserName</key>`, `  <string>${config.user}</string>`]
      : [];

    const content = [
      `<?xml version="1.0" encoding="UTF-8"?>`,
      `<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`,
      `<plist version="1.0">`,
      `<dict>`,
      `  <key>Label</key>`,
      `  <string>com.startower.${config.name}</string>`,
      `  <key>ProgramArguments</key>`,
      `  <array>`,
      `    <string>${config.nodeExecPath}</string>`,
      `    <string>${config.execPath}</string>`,
      `  </array>`,
      `  <key>WorkingDirectory</key>`,
      `  <string>${config.workingDirectory}</string>`,
      `  <key>RunAtLoad</key>`,
      `  <true/>`,
      `  <key>KeepAlive</key>`,
      `  <true/>`,
      ...userEntries,
      ...logEntries,
      `</dict>`,
      `</plist>`,
      "",
    ].join("\n");

    return {
      platform: "launchd",
      content,
      installPath: this.getInstallPath(config.name, "launchd"),
    };
  }

  private generateWinschtask(config: ServiceConfig): ServiceUnit {
    const content = [
      `<?xml version="1.0" encoding="UTF-16"?>`,
      `<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">`,
      `  <RegistrationInfo>`,
      `    <Description>${config.description}</Description>`,
      `  </RegistrationInfo>`,
      `  <Triggers>`,
      `    <BootTrigger>`,
      `      <Enabled>true</Enabled>`,
      `    </BootTrigger>`,
      `  </Triggers>`,
      `  <Principals>`,
      `    <Principal id="Author">`,
      config.user
        ? `      <UserId>${config.user}</UserId>`
        : `      <LogonType>ServiceAccount</LogonType>`,
      `      <RunLevel>LeastPrivilege</RunLevel>`,
      `    </Principal>`,
      `  </Principals>`,
      `  <Settings>`,
      `    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>`,
      `    <DisallowStartIfOnBatteries>false</DisallowStartIfOnBatteries>`,
      `    <StopIfGoingOnBatteries>false</StopIfGoingOnBatteries>`,
      `    <ExecutionTimeLimit>PT0S</ExecutionTimeLimit>`,
      `    <RestartOnFailure>`,
      `      <Interval>PT1M</Interval>`,
      `      <Count>3</Count>`,
      `    </RestartOnFailure>`,
      `  </Settings>`,
      `  <Actions>`,
      `    <Exec>`,
      `      <Command>${config.nodeExecPath}</Command>`,
      `      <Arguments>${config.execPath}</Arguments>`,
      `      <WorkingDirectory>${config.workingDirectory}</WorkingDirectory>`,
      `    </Exec>`,
      `  </Actions>`,
      `</Task>`,
      "",
    ].join("\n");

    return {
      platform: "winschtask",
      content,
      installPath: this.getInstallPath(config.name, "winschtask"),
    };
  }
}
