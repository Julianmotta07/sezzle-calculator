import { describe, expect, it } from 'vitest'
import { ApiError, NetworkError } from '../api/calculator'
import { formatNumber, parseOperand } from './format'
import { friendlyErrorMessage } from './messages'
import { getOperation, isOperation, OPERATIONS } from './operations'

describe('parseOperand', () => {
  it.each([
    ['42', 42],
    ['-3.5', -3.5],
    ['+7', 7],
    ['.5', 0.5],
    ['5.', 5],
    ['  12  ', 12],
    ['1e3', 1000],
    ['0', 0],
  ])('accepts %j', (raw, want) => {
    expect(parseOperand(raw, 'first number')).toEqual({ ok: true, value: want })
  })

  it.each(['abc', '1,5', '0x10', 'Infinity', '1..2', '-', '.', '1e'])('rejects %j as not a number', (raw) => {
    const result = parseOperand(raw, 'first number')
    expect(result.ok).toBe(false)
    expect(!result.ok && result.message).toMatch(/is not a valid number/)
  })

  it('rejects empty input', () => {
    expect(parseOperand('   ', 'second number')).toEqual({ ok: false, message: 'Enter the second number.' })
  })

  it('rejects numbers too large for a float64', () => {
    expect(parseOperand('1e400', 'first number')).toEqual({ ok: false, message: 'The first number is too large.' })
  })
})

describe('formatNumber', () => {
  it.each([
    [5, '5'],
    [0.1 + 0.2, '0.3'],
    [1 / 3, '0.333333333333333'],
    [-0, '0'],
    [-2.5, '-2.5'],
    [1e21, '1e+21'],
  ])('formats %d as %s', (n, want) => {
    expect(formatNumber(n)).toBe(want)
  })
})

describe('operations', () => {
  it('lists every backend operation exactly once', () => {
    expect(OPERATIONS.map((op) => op.value)).toEqual([
      'add', 'subtract', 'multiply', 'divide', 'power', 'sqrt', 'percentage',
    ])
  })

  it('only sqrt is unary', () => {
    expect(OPERATIONS.filter((op) => op.arity === 1).map((op) => op.value)).toEqual(['sqrt'])
  })

  it.each([
    ['subtract', '5', '-2', '5 − (-2)'],
    ['power', '-2', '3', '(-2) ^ 3'],
    ['percentage', '20', '50', '20% of 50'],
    ['sqrt', '-4', '', '√(-4)'],
  ] as const)('describes %s', (op, a, b, want) => {
    expect(getOperation(op).describe(a, b)).toBe(want)
  })

  it('guards operation names', () => {
    expect(isOperation('divide')).toBe(true)
    expect(isOperation('modulo')).toBe(false)
  })
})

describe('friendlyErrorMessage', () => {
  it.each([
    [new ApiError('DIVISION_BY_ZERO', 'x', 422), /divide by zero/],
    [new ApiError('NEGATIVE_SQRT', 'x', 422), /zero or greater/],
    [new ApiError('NON_FINITE_RESULT', 'x', 422), /too large or undefined/],
    [new ApiError('INTERNAL_ERROR', 'x', 500), /server had a problem/],
    [new ApiError('UNEXPECTED_RESPONSE', 'x', 200), /server had a problem/],
    [new ApiError('MISSING_OPERAND', 'operation "add" requires field "b"', 400), /rejected: operation "add" requires/],
    [new NetworkError('down'), /can't reach the calculator server/i],
    [new Error('boom'), /something unexpected/i],
  ])('maps %o', (error, want) => {
    expect(friendlyErrorMessage(error)).toMatch(want)
  })
})
