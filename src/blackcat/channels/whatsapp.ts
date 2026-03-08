import makeWASocket, {
  DisconnectReason,
  fetchLatestBaileysVersion,
} from '@whiskeysockets/baileys'
import { createAuthState } from './whatsapp-auth.js'
import type { ChannelAdapter, InboundMessage, OutboundMessage } from './types.js'

export interface WhatsAppConfig {
  sessionDir: string
  allowFrom?: string[]
  qrTerminal?: boolean
}

export function normalizeE164(phone: string): string {
  return phone.replace(/\D/g, '')
}

export function createAllowFromSet(allowFrom?: string[]): Set<string> | undefined {
  if (!allowFrom?.length) {
    return undefined
  }
  const normalized = allowFrom.map(normalizeE164).filter(Boolean)
  if (normalized.length === 0) {
    return undefined
  }
  return new Set(normalized)
}

export function isPhoneAllowed(phone: string, allowFromSet?: Set<string>): boolean {
  if (!allowFromSet) {
    return true
  }
  return allowFromSet.has(normalizeE164(phone))
}

function makeSilentLogger() {
  const noop = () => {}
  return {
    level: 'silent',
    trace: noop,
    debug: noop,
    info: noop,
    warn: noop,
    error: noop,
    fatal: noop,
    child: () => makeSilentLogger(),
  }
}

export class WhatsAppAdapter implements ChannelAdapter {
  readonly channelType = 'whatsapp' as const
  private sock?: Awaited<ReturnType<typeof makeWASocket>>
  private handler?: (msg: InboundMessage) => Promise<void>
  private allowFromSet?: Set<string>
  private running = false
  private reconnecting = false

  constructor(private readonly config: WhatsAppConfig) {
    this.allowFromSet = createAllowFromSet(config.allowFrom)
  }

  async start(): Promise<void> {
    this.running = true
    await this.connect()
  }

  private async connect(): Promise<void> {
    const { state, saveCreds } = await createAuthState(this.config.sessionDir)
    const { version } = await fetchLatestBaileysVersion()

    this.sock = makeWASocket({
      version,
      auth: state,
      printQRInTerminal: this.config.qrTerminal ?? true,
      logger: makeSilentLogger() as never,
    })

    this.sock.ev.on('creds.update', saveCreds)

    this.sock.ev.on('connection.update', ({ connection, lastDisconnect, qr }) => {
      if (qr && this.config.qrTerminal !== false) {
        console.log('[BlackCat WhatsApp] Scan QR code to link device')
      }
      if (connection === 'close') {
        const code =
          (lastDisconnect?.error as { output?: { statusCode?: number } } | undefined)?.output
            ?.statusCode
        if (code !== DisconnectReason.loggedOut && this.running) {
          console.log('[BlackCat WhatsApp] Reconnecting...')
          if (!this.reconnecting) {
            this.reconnecting = true
            void this.connect().finally(() => {
              this.reconnecting = false
            })
          }
        } else {
          console.log(
            '[BlackCat WhatsApp] Logged out. Delete session dir and restart to relink.',
          )
        }
      }
      if (connection === 'open') {
        console.log('[BlackCat WhatsApp] Connected.')
      }
    })

    this.sock.ev.on('messages.upsert', async ({ messages, type }) => {
      if (type !== 'notify') return

      for (const m of messages) {
        if (m.key.fromMe) continue

        const text =
          m.message?.conversation ??
          m.message?.extendedTextMessage?.text ??
          m.message?.imageMessage?.caption ??
          ''
        if (!text) continue

        const sender = m.key.participant ?? m.key.remoteJid ?? ''
        const phone = sender.split('@')[0] ?? sender

        if (!isPhoneAllowed(phone, this.allowFromSet)) {
          continue
        }

        const timestampSeconds =
          typeof m.messageTimestamp === 'number'
            ? m.messageTimestamp
            : Number(m.messageTimestamp ?? 0)

        const msg: InboundMessage = {
          id: m.key.id ?? '',
          channelType: 'whatsapp',
          accountId: this.sock?.user?.id ?? '',
          peer: phone,
          text,
          timestamp: timestampSeconds * 1000,
          raw: m,
        }

        await this.handler?.(msg)
      }
    })
  }

  async stop(): Promise<void> {
    this.running = false
    this.sock?.end(undefined)
  }

  async send(msg: OutboundMessage): Promise<void> {
    const jid = msg.peer.includes('@') ? msg.peer : `${msg.peer}@s.whatsapp.net`
    await this.sock?.sendMessage(jid, { text: msg.text })
  }

  onMessage(handler: (msg: InboundMessage) => Promise<void>): void {
    this.handler = handler
  }
}
