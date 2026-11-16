import { useRef, useState } from 'react'
import { calculate as requestCalculation, type Operation } from '../api/calculator'
import { formatNumber, parseOperand } from '../calculator/format'
import { friendlyErrorMessage } from '../calculator/messages'
import { getOperation } from '../calculator/operations'

export type CalculatorStatus = 'idle' | 'loading' | 'success' | 'error'

/** A finished calculation, ready to display. */
export interface CalculationResult {
  /** Readable expression, e.g. "10 ÷ 2". */
  expression: string
  /** Formatted result, e.g. "5". */
  value: string
}

/** Client-side validation messages per input. */
export interface FieldErrors {
  a?: string
  b?: string
}

export type Field = keyof FieldErrors

/**
 * Holds the calculator form state and talks to the API client.
 * Inputs are kept as raw strings so the user can type freely;
 * they are validated only when `calculate` runs.
 */
export function useCalculator(initialOperation: Operation = 'add') {
  const [operation, setOperationState] = useState<Operation>(initialOperation)
  const [a, setAState] = useState('')
  const [b, setBState] = useState('')
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
  const [status, setStatus] = useState<CalculatorStatus>('idle')
  const [result, setResult] = useState<CalculationResult | null>(null)
  const [error, setError] = useState<string | null>(null)
  // Id of the latest request, so a slow older response can't overwrite a newer one.
  const latestRequest = useRef(0)

  const info = getOperation(operation)
  const isUnary = info.arity === 1

  function setOperation(next: Operation) {
    setOperationState(next)
    // The second input may be hidden now; don't leave a stale message for it.
    setFieldErrors((prev) => ({ a: prev.a }))
  }

  function setA(value: string) {
    setAState(value)
    setFieldErrors((prev) => ({ ...prev, a: undefined }))
  }

  function setB(value: string) {
    setBState(value)
    setFieldErrors((prev) => ({ ...prev, b: undefined }))
  }

  /**
   * Validates the inputs and, if they are valid, calls the API.
   * Resolves to the first invalid field (so the UI can focus it), or null.
   */
  async function calculate(): Promise<Field | null> {
    const parsedA = parseOperand(a, 'first number')
    const parsedB = isUnary ? null : parseOperand(b, 'second number')

    const errors: FieldErrors = {}
    if (!parsedA.ok) errors.a = parsedA.message
    if (parsedB && !parsedB.ok) errors.b = parsedB.message
    setFieldErrors(errors)
    if (!parsedA.ok || (parsedB && !parsedB.ok)) {
      return errors.a ? 'a' : 'b'
    }

    const x = parsedA.value
    const y = parsedB?.ok ? parsedB.value : undefined
    const requestId = ++latestRequest.current
    setStatus('loading')
    setError(null)

    try {
      const response = y === undefined
        ? await requestCalculation(operation, x)
        : await requestCalculation(operation, x, y)
      if (requestId !== latestRequest.current) return null
      setResult({
        expression: info.describe(formatNumber(x), y === undefined ? '' : formatNumber(y)),
        value: formatNumber(response.result),
      })
      setStatus('success')
    } catch (err) {
      if (requestId !== latestRequest.current) return null
      setResult(null)
      setError(friendlyErrorMessage(err))
      setStatus('error')
    }
    return null
  }

  return {
    operation,
    a,
    b,
    isUnary,
    fieldErrors,
    status,
    isLoading: status === 'loading',
    result,
    error,
    setOperation,
    setA,
    setB,
    calculate,
  }
}
