export type ChannelType = 'telegram' | 'discord' | 'whatsapp'

export interface InboundMessage {
  id: string
  channelType: ChannelType
  accountId: string
  peer: string
  text: string
  mediaPath?: string
  timestamp: number
  raw?: unknown
}

export interface OutboundMessage {
  peer: string
  text: string
  replyToId?: string
}

export interface ChannelAdapter {
  readonly channelType: ChannelType
  start(): Promise<void>
  stop(): Promise<void>
  send(msg: OutboundMessage): Promise<void>
  onMessage(handler: (msg: InboundMessage) => Promise<void>): void
}
