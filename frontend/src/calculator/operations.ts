import type { Operation } from '../api/calculator'

/** UI metadata for one operation. */
export interface OperationInfo {
  value: Operation
  /** Text shown in the operation selector. */
  label: string
  /** Number of operands the operation needs. */
  arity: 1 | 2
  /** Builds a readable expression from already-formatted operands, e.g. "10 ÷ 2". */
  describe: (a: string, b: string) => string
}

/** Wraps negative operands in parentheses so "5 − (-2)" reads correctly. */
const paren = (x: string) => (x.startsWith('-') ? `(${x})` : x)

/** Operations in the order they appear in the selector. */
export const OPERATIONS: readonly OperationInfo[] = [
  { value: 'add', label: 'Add (a + b)', arity: 2, describe: (a, b) => `${a} + ${paren(b)}` },
  { value: 'subtract', label: 'Subtract (a − b)', arity: 2, describe: (a, b) => `${a} − ${paren(b)}` },
  { value: 'multiply', label: 'Multiply (a × b)', arity: 2, describe: (a, b) => `${a} × ${paren(b)}` },
  { value: 'divide', label: 'Divide (a ÷ b)', arity: 2, describe: (a, b) => `${a} ÷ ${paren(b)}` },
  { value: 'power', label: 'Power (a ^ b)', arity: 2, describe: (a, b) => `${paren(a)} ^ ${paren(b)}` },
  { value: 'sqrt', label: 'Square root (√a)', arity: 1, describe: (a) => `√${paren(a)}` },
  { value: 'percentage', label: 'Percentage (a% of b)', arity: 2, describe: (a, b) => `${a}% of ${b}` },
]

/** Returns the metadata for an operation. */
export function getOperation(value: Operation): OperationInfo {
  const info = OPERATIONS.find((op) => op.value === value)
  if (!info) {
    throw new Error(`Unknown operation: ${value}`)
  }
  return info
}

/** Type guard for strings coming from the DOM (e.g. a <select> value). */
export function isOperation(value: string): value is Operation {
  return OPERATIONS.some((op) => op.value === value)
}
