import { Bot } from 'grammy'
import type { ChannelAdapter, InboundMessage, OutboundMessage } from './types.js'

export class TelegramAdapter implements ChannelAdapter {
  readonly channelType = 'telegram' as const
  private bot: Bot
  private handler?: (msg: InboundMessage) => Promise<void>

  constructor(private readonly token: string) {
    this.bot = new Bot(token)
    this.bot.on('message:text', async (ctx) => {
      const msg: InboundMessage = {
        id: String(ctx.message.message_id),
        channelType: 'telegram',
        accountId: String(ctx.me.id),
        peer: String(ctx.message.from.id),
        text: ctx.message.text,
        timestamp: ctx.message.date * 1000,
        raw: ctx.message,
      }
      await this.handler?.(msg)
    })
  }

  async start(): Promise<void> {
    // grammY bot.start() is non-blocking in newer versions; fire and don't await
    void this.bot.start({ drop_pending_updates: true })
  }

  async stop(): Promise<void> {
    await this.bot.stop()
  }

  async send(msg: OutboundMessage): Promise<void> {
    await this.bot.api.sendMessage(msg.peer, msg.text)
  }

  onMessage(handler: (msg: InboundMessage) => Promise<void>): void {
    this.handler = handler
  }
}
