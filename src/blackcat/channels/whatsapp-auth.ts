import { useMultiFileAuthState } from '@whiskeysockets/baileys'

export async function createAuthState(sessionDir: string) {
  return useMultiFileAuthState(sessionDir)
}
