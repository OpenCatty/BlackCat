import { describe, expect, it } from 'vitest'
import type {
  ChannelAdapter,
  ChannelType,
  InboundMessage,
  OutboundMessage,
} from './types.js'

class MockAdapter implements ChannelAdapter {
  readonly channelType: ChannelType = 'telegram'
  private handler?: (msg: InboundMessage) => Promise<void>
  public sent: OutboundMessage[] = []
  started = false
  stopped = false

  async start() {
    this.started = true
  }

  async stop() {
    this.stopped = true
  }

  async send(msg: OutboundMessage) {
    this.sent.push(msg)
  }

  onMessage(h: (msg: InboundMessage) => Promise<void>) {
    this.handler = h
  }

  async simulateInbound(msg: InboundMessage) {
    await this.handler?.(msg)
  }
}

describe('ChannelAdapter interface', () => {
  it('registers message handler and fires on inbound', async () => {
    const adapter = new MockAdapter()
    const received: InboundMessage[] = []

    adapter.onMessage(async (msg) => {
      received.push(msg)
    })

    await adapter.start()
    await adapter.simulateInbound({
      id: '1',
      channelType: 'telegram',
      accountId: 'bot123',
      peer: 'user456',
      text: 'hello',
      timestamp: Date.now(),
    })

    expect(received).toHaveLength(1)
    expect(received[0]!.text).toBe('hello')
  })

  it('sends outbound message', async () => {
    const adapter = new MockAdapter()
    await adapter.send({ peer: 'user456', text: 'hi back' })

    expect(adapter.sent).toHaveLength(1)
    expect(adapter.sent[0]!.text).toBe('hi back')
  })

  it('start and stop lifecycle', async () => {
    const adapter = new MockAdapter()
    await adapter.start()
    expect(adapter.started).toBe(true)
    await adapter.stop()
    expect(adapter.stopped).toBe(true)
  })

  it('session key = channel:accountId:peer', async () => {
    const adapter = new MockAdapter()
    const keys: string[] = []

    adapter.onMessage(async (msg) => {
      keys.push(`${msg.channelType}:${msg.accountId}:${msg.peer}`)
    })

    await adapter.simulateInbound({
      id: '1',
      channelType: 'telegram',
      accountId: 'bot1',
      peer: 'user1',
      text: 'test',
      timestamp: 0,
    })

    expect(keys[0]).toBe('telegram:bot1:user1')
  })

  it('no handler = no crash on inbound', async () => {
    const adapter = new MockAdapter()

    await expect(
      adapter.simulateInbound({
        id: '1',
        channelType: 'telegram',
        accountId: 'bot1',
        peer: 'user1',
        text: 'test',
        timestamp: 0,
      }),
    ).resolves.not.toThrow()
  })
})
