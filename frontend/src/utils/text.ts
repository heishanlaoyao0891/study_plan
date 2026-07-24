export function normalizeDisplayText(value: string): string {
  return String(value || '').trim().normalize('NFKC')
}

export function unicodeLength(value: string): number {
  return Array.from(normalizeDisplayText(value)).length
}
