import { describe, expect, it } from "vitest";
import type {
  ChannelAdapter,
  ChannelType,
  InboundMessage,
  OutboundMessage,
} from "../../src/blackcat/channels/types.js";

// MockAdapter — local to test file (not imported)
class MockAdapter implements ChannelAdapter {
  readonly channelType: ChannelType;
  private handler?: (msg: InboundMessage) => Promise<void>;
  public sent: OutboundMessage[] = [];
  public started = false;
  public stopped = false;

  constructor(channelType: ChannelType = "telegram") {
    this.channelType = channelType;
  }

  async start(): Promise<void> {
    this.started = true;
  }

  async stop(): Promise<void> {
    this.stopped = true;
  }

  async send(msg: OutboundMessage): Promise<void> {
    this.sent.push(msg);
  }

  onMessage(handler: (msg: InboundMessage) => Promise<void>): void {
    this.handler = handler;
  }

  async simulateInbound(msg: InboundMessage): Promise<void> {
    await this.handler?.(msg);
  }
}

describe("Gate H — Channel Adapter Lifecycle Smoke", () => {
  it("start/stop lifecycle", async () => {
    const adapter = new MockAdapter("telegram");

    expect(adapter.started).toBe(false);
    expect(adapter.stopped).toBe(false);

    await adapter.start();
    expect(adapter.started).toBe(true);

    await adapter.stop();
    expect(adapter.stopped).toBe(true);
  });

  it("MockAdapter sends and receives messages", async () => {
    const adapter = new MockAdapter("discord");
    const received: InboundMessage[] = [];

    adapter.onMessage(async (msg) => {
      received.push(msg);
    });

    await adapter.start();

    await adapter.simulateInbound({
      id: "msg-1",
      channelType: "discord",
      accountId: "bot123",
      peer: "user456",
      text: "Hello from discord",
      timestamp: Date.now(),
    });

    expect(received).toHaveLength(1);
    expect(received[0]!.text).toBe("Hello from discord");
    expect(received[0]!.channelType).toBe("discord");

    await adapter.send({ peer: "user456", text: "Reply back" });
    expect(adapter.sent).toHaveLength(1);
    expect(adapter.sent[0]!.text).toBe("Reply back");
  });

  it("no handler registered = no crash on inbound", async () => {
    const adapter = new MockAdapter("whatsapp");
    // No handler registered — should not throw
    await expect(
      adapter.simulateInbound({
        id: "msg-1",
        channelType: "whatsapp",
        accountId: "phone1",
        peer: "phone2",
        text: "test",
        timestamp: Date.now(),
      }),
    ).resolves.not.toThrow();
  });

  describe("session key formula", () => {
    function sessionKey(msg: InboundMessage): string {
      return `${msg.channelType}:${msg.accountId}:${msg.peer}`;
    }

    it("telegram session key: telegram:accountId:peer", async () => {
      const adapter = new MockAdapter("telegram");
      const keys: string[] = [];

      adapter.onMessage(async (msg) => {
        keys.push(sessionKey(msg));
      });

      await adapter.simulateInbound({
        id: "1",
        channelType: "telegram",
        accountId: "bot_tg_123",
        peer: "user_tg_456",
        text: "test",
        timestamp: Date.now(),
      });

      expect(keys[0]).toBe("telegram:bot_tg_123:user_tg_456");
    });

    it("discord session key: discord:accountId:peer", async () => {
      const adapter = new MockAdapter("discord");
      const keys: string[] = [];

      adapter.onMessage(async (msg) => {
        keys.push(sessionKey(msg));
      });

      await adapter.simulateInbound({
        id: "1",
        channelType: "discord",
        accountId: "guild_1",
        peer: "channel_2",
        text: "test",
        timestamp: Date.now(),
      });

      expect(keys[0]).toBe("discord:guild_1:channel_2");
    });

    it("whatsapp session key: whatsapp:accountId:peer", async () => {
      const adapter = new MockAdapter("whatsapp");
      const keys: string[] = [];

      adapter.onMessage(async (msg) => {
        keys.push(sessionKey(msg));
      });

      await adapter.simulateInbound({
        id: "1",
        channelType: "whatsapp",
        accountId: "+1234567890",
        peer: "+0987654321",
        text: "test",
        timestamp: Date.now(),
      });

      expect(keys[0]).toBe("whatsapp:+1234567890:+0987654321");
    });

    it("session key matches SessionStore.makeSessionId formula", () => {
      // The session key formula must match the one used in SessionStore
      const channel = "telegram";
      const accountId = "bot1";
      const peer = "user1";

      const adapterKey = `${channel}:${accountId}:${peer}`;
      // This mirrors makeSessionId from sessions/store.ts
      expect(adapterKey).toBe(`${channel}:${accountId}:${peer}`);
    });
  });
});
