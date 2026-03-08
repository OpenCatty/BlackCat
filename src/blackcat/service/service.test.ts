import { describe, it, expect, beforeEach } from "vitest";

import { ServiceGenerator } from "./generator.js";
import type { ServiceConfig } from "./types.js";

function makeConfig(overrides?: Partial<ServiceConfig>): ServiceConfig {
  return {
    name: "blackcat",
    description: "BlackCat AI Agent Daemon",
    execPath: "/opt/blackcat/blackcat.mjs",
    nodeExecPath: "/usr/bin/node",
    workingDirectory: "/opt/blackcat",
    ...overrides,
  };
}

describe("ServiceGenerator", () => {
  let generator: ServiceGenerator;

  beforeEach(() => {
    generator = new ServiceGenerator();
  });

  // ---------------------------------------------------------------------------
  // detectPlatform
  // ---------------------------------------------------------------------------

  it("detectPlatform returns a valid ServicePlatform", () => {
    const platform = generator.detectPlatform();
    expect(["systemd", "launchd", "winschtask"]).toContain(platform);
  });

  it("detectPlatform returns winschtask on win32", () => {
    // We're running on win32 in this env
    if (process.platform === "win32") {
      expect(generator.detectPlatform()).toBe("winschtask");
    }
  });

  // ---------------------------------------------------------------------------
  // generateUnit — systemd
  // ---------------------------------------------------------------------------

  it("generateUnit returns non-empty string for systemd", () => {
    const unit = generator.generateUnit(makeConfig(), "systemd");
    expect(unit.content.length).toBeGreaterThan(0);
    expect(unit.platform).toBe("systemd");
  });

  it("generateUnit contains service name in systemd output", () => {
    const config = makeConfig({ name: "my-daemon" });
    const unit = generator.generateUnit(config, "systemd");
    expect(unit.content).toContain("my-daemon");
    expect(unit.installPath).toContain("my-daemon");
  });

  it("systemd unit has [Unit], [Service], [Install] sections", () => {
    const unit = generator.generateUnit(makeConfig(), "systemd");
    expect(unit.content).toContain("[Unit]");
    expect(unit.content).toContain("[Service]");
    expect(unit.content).toContain("[Install]");
  });

  it("systemd unit contains ExecStart, Restart, and Description", () => {
    const config = makeConfig();
    const unit = generator.generateUnit(config, "systemd");
    expect(unit.content).toContain("ExecStart=/usr/bin/node /opt/blackcat/blackcat.mjs");
    expect(unit.content).toContain("Restart=on-failure");
    expect(unit.content).toContain(`Description=${config.description}`);
  });

  it("systemd unit includes User when provided", () => {
    const unit = generator.generateUnit(makeConfig({ user: "blackcat" }), "systemd");
    expect(unit.content).toContain("User=blackcat");
  });

  it("systemd unit includes log paths when provided", () => {
    const unit = generator.generateUnit(
      makeConfig({ logPath: "/var/log/blackcat.log" }),
      "systemd",
    );
    expect(unit.content).toContain("StandardOutput=append:/var/log/blackcat.log");
    expect(unit.content).toContain("StandardError=append:/var/log/blackcat.log");
  });

  // ---------------------------------------------------------------------------
  // generateUnit — launchd
  // ---------------------------------------------------------------------------

  it("launchd plist has correct XML structure", () => {
    const config = makeConfig({ name: "blackcat" });
    const unit = generator.generateUnit(config, "launchd");

    expect(unit.platform).toBe("launchd");
    expect(unit.content).toContain("<?xml");
    expect(unit.content).toContain("<plist");
    expect(unit.content).toContain("<dict>");
    expect(unit.content).toContain("<key>Label</key>");
    expect(unit.content).toContain("<string>com.startower.blackcat</string>");
  });

  it("launchd plist contains ProgramArguments with node and script", () => {
    const config = makeConfig();
    const unit = generator.generateUnit(config, "launchd");
    expect(unit.content).toContain("<key>ProgramArguments</key>");
    expect(unit.content).toContain(`<string>${config.nodeExecPath}</string>`);
    expect(unit.content).toContain(`<string>${config.execPath}</string>`);
  });

  it("launchd plist includes KeepAlive and RunAtLoad", () => {
    const unit = generator.generateUnit(makeConfig(), "launchd");
    expect(unit.content).toContain("<key>KeepAlive</key>");
    expect(unit.content).toContain("<key>RunAtLoad</key>");
  });

  it("launchd install path uses com.startower prefix", () => {
    const unit = generator.generateUnit(makeConfig({ name: "kitty" }), "launchd");
    expect(unit.installPath).toContain("com.startower.kitty.plist");
  });

  // ---------------------------------------------------------------------------
  // generateUnit — winschtask
  // ---------------------------------------------------------------------------

  it("winschtask XML has correct task structure", () => {
    const config = makeConfig();
    const unit = generator.generateUnit(config, "winschtask");

    expect(unit.platform).toBe("winschtask");
    expect(unit.content).toContain("<Task");
    expect(unit.content).toContain("<RegistrationInfo>");
    expect(unit.content).toContain("<Triggers>");
    expect(unit.content).toContain("<Actions>");
    expect(unit.content).toContain("<Exec>");
  });

  it("winschtask XML contains Description and Command", () => {
    const config = makeConfig();
    const unit = generator.generateUnit(config, "winschtask");
    expect(unit.content).toContain(`<Description>${config.description}</Description>`);
    expect(unit.content).toContain(`<Command>${config.nodeExecPath}</Command>`);
    expect(unit.content).toContain(`<Arguments>${config.execPath}</Arguments>`);
    expect(unit.content).toContain(
      `<WorkingDirectory>${config.workingDirectory}</WorkingDirectory>`,
    );
  });

  it("winschtask XML includes BootTrigger for auto-start", () => {
    const unit = generator.generateUnit(makeConfig(), "winschtask");
    expect(unit.content).toContain("<BootTrigger>");
    expect(unit.content).toContain("<Enabled>true</Enabled>");
  });

  it("winschtask XML includes restart-on-failure settings", () => {
    const unit = generator.generateUnit(makeConfig(), "winschtask");
    expect(unit.content).toContain("<RestartOnFailure>");
  });

  // ---------------------------------------------------------------------------
  // ServiceConfig validation / edge cases
  // ---------------------------------------------------------------------------

  it("generateUnit uses detected platform when no override given", () => {
    const unit = generator.generateUnit(makeConfig());
    const expected = generator.detectPlatform();
    expect(unit.platform).toBe(expected);
  });

  it("installUnit and uninstallUnit are async functions", () => {
    // Verify the API shape without actually hitting the filesystem
    const unit = generator.generateUnit(makeConfig(), "systemd");
    const installResult = generator.installUnit(unit, makeConfig());
    expect(installResult).toBeInstanceOf(Promise);

    const uninstallResult = generator.uninstallUnit(makeConfig(), "systemd");
    expect(uninstallResult).toBeInstanceOf(Promise);
  });
});
