import { type FormEvent, useRef } from 'react'
import { useCalculator } from '../hooks/useCalculator'
import { NumberField } from './NumberField'
import { OperationSelector } from './OperationSelector'
import { ResultDisplay } from './ResultDisplay'
import './Calculator.css'

/** The calculator form: operation, operands, submit button and result. */
export function Calculator() {
  const calc = useCalculator()
  const aRef = useRef<HTMLInputElement>(null)
  const bRef = useRef<HTMLInputElement>(null)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const invalidField = await calc.calculate()
    // Move focus to the first invalid input so keyboard and screen reader users land on the problem.
    if (invalidField === 'a') aRef.current?.focus()
    if (invalidField === 'b') bRef.current?.focus()
  }

  return (
    <div className="calculator">
      <h1>Calculator</h1>
      <form className="calculator-form" onSubmit={handleSubmit} noValidate aria-busy={calc.isLoading}>
        <OperationSelector value={calc.operation} onChange={calc.setOperation} />
        <div className="operands">
          <NumberField
            ref={aRef}
            label={calc.isUnary ? 'Number' : 'First number'}
            value={calc.a}
            onChange={calc.setA}
            error={calc.fieldErrors.a}
          />
          {!calc.isUnary && (
            <NumberField
              ref={bRef}
              label="Second number"
              value={calc.b}
              onChange={calc.setB}
              error={calc.fieldErrors.b}
            />
          )}
        </div>
        <button type="submit" className="calculate-button" disabled={calc.isLoading}>
          {calc.isLoading ? 'Calculating…' : 'Calculate'}
        </button>
      </form>
      <ResultDisplay status={calc.status} result={calc.result} error={calc.error} />
    </div>
  )
}
