import type { CalculationResult, CalculatorStatus } from '../hooks/useCalculator'

interface ResultDisplayProps {
  status: CalculatorStatus
  result: CalculationResult | null
  error: string | null
}

/**
 * Shows the result or the API error. Both live regions are always rendered,
 * so screen readers announce changes to their content.
 */
export function ResultDisplay({ status, result, error }: ResultDisplayProps) {
  return (
    <section className="result" aria-label="Result">
      <div role="status" aria-live="polite" aria-atomic="true" className="result-status">
        {status === 'idle' && <p className="result-placeholder">Your result will appear here.</p>}
        {status === 'loading' && <p className="result-placeholder">Calculating…</p>}
        {status === 'success' && result && (
          <p>
            <span className="result-expression">{result.expression} =</span>{' '}
            <span className="result-value">{result.value}</span>
          </p>
        )}
      </div>
      <div role="alert" className="result-error">
        {status === 'error' && error && <p>{error}</p>}
      </div>
    </section>
  )
}
