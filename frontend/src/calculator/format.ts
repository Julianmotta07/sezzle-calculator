/** Strict decimal number: optional sign, digits with an optional dot, optional exponent. */
const NUMBER_PATTERN = /^[+-]?(\d+\.?\d*|\.\d+)(e[+-]?\d+)?$/i

export type ParseResult = { ok: true; value: number } | { ok: false; message: string }

/**
 * Validates and parses the text of a number input.
 * `fieldName` is used in messages, e.g. "first number".
 *
 * Stricter than Number(): rejects "", "0x10", "Infinity" and "1,5".
 */
export function parseOperand(raw: string, fieldName: string): ParseResult {
  const text = raw.trim()
  if (text === '') {
    return { ok: false, message: `Enter the ${fieldName}.` }
  }
  if (!NUMBER_PATTERN.test(text)) {
    return {
      ok: false,
      message: `"${text}" is not a valid number. Use digits and a dot for decimals, e.g. -12.5.`,
    }
  }
  const value = Number(text)
  if (!Number.isFinite(value)) {
    return { ok: false, message: `The ${fieldName} is too large.` }
  }
  return { ok: true, value }
}

/**
 * Formats a number for display, hiding floating-point noise
 * (0.1 + 0.2 shows as 0.3) while keeping up to 15 significant digits.
 */
export function formatNumber(n: number): string {
  if (Object.is(n, -0)) {
    return '0'
  }
  return String(Number(n.toPrecision(15)))
}
