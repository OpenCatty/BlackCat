import { describe, expect, it } from 'vitest'
import { normalizeE164 } from './whatsapp.js'

describe('normalizeE164', () => {
  it('strips non-digit characters', () => {
    expect(normalizeE164('+62-812-3456-7890')).toBe('6281234567890')
  })

  it('handles plain number', () => {
    expect(normalizeE164('6281234567890')).toBe('6281234567890')
  })

  it('handles spaces', () => {
    expect(normalizeE164('+1 415 555 0101')).toBe('14155550101')
  })

  it('empty string', () => {
    expect(normalizeE164('')).toBe('')
  })
})

describe('allowFrom filtering logic', () => {
  function shouldAllow(allowFrom: string[] | undefined, phone: string): boolean {
    if (!allowFrom || allowFrom.length === 0) return true
    const set = new Set(allowFrom.map(normalizeE164))
    return set.has(normalizeE164(phone))
  }

  it('allows all when allowFrom is undefined', () => {
    expect(shouldAllow(undefined, '6281234567890')).toBe(true)
  })

  it('allows all when allowFrom is empty', () => {
    expect(shouldAllow([], '6281234567890')).toBe(true)
  })

  it('allows whitelisted number', () => {
    expect(shouldAllow(['+6281234567890'], '6281234567890')).toBe(true)
  })

  it('blocks non-whitelisted number', () => {
    expect(shouldAllow(['+6281234567890'], '6289999999999')).toBe(false)
  })

  it('normalizes both sides before comparing', () => {
    expect(shouldAllow(['+62 812-345-67890'], '+6281234567890')).toBe(true)
  })
})
