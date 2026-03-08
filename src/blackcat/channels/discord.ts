import { Client, Events, GatewayIntentBits } from 'discord.js'
import type { ChannelAdapter, InboundMessage, OutboundMessage } from './types.js'

export class DiscordAdapter implements ChannelAdapter {
  readonly channelType = 'discord' as const
  private client: Client
  private handler?: (msg: InboundMessage) => Promise<void>

  constructor(private readonly token: string) {
    this.client = new Client({
      intents: [
        GatewayIntentBits.Guilds,
        GatewayIntentBits.GuildMessages,
        GatewayIntentBits.MessageContent,
        GatewayIntentBits.DirectMessages,
      ],
    })
    this.client.on(Events.MessageCreate, async (message) => {
      if (message.author.bot) return
      const msg: InboundMessage = {
        id: message.id,
        channelType: 'discord',
        accountId: String(this.client.user?.id ?? ''),
        peer: message.author.id,
        text: message.content,
        timestamp: message.createdTimestamp,
        raw: message,
      }
      await this.handler?.(msg)
    })
  }

  async start(): Promise<void> {
    await this.client.login(this.token)
  }

  async stop(): Promise<void> {
    this.client.destroy()
  }

  async send(msg: OutboundMessage): Promise<void> {
    const channel = await this.client.channels.fetch(msg.peer)
    if (channel?.isTextBased()) {
      await (channel as any).send(msg.text)
    }
  }

  onMessage(handler: (msg: InboundMessage) => Promise<void>): void {
    this.handler = handler
  }
}
