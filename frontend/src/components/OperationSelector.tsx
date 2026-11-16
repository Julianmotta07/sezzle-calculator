import { useId } from 'react'
import type { Operation } from '../api/calculator'
import { isOperation, OPERATIONS } from '../calculator/operations'

interface OperationSelectorProps {
  value: Operation
  onChange: (operation: Operation) => void
}

/** Labelled <select> listing every supported operation. */
export function OperationSelector({ value, onChange }: OperationSelectorProps) {
  const id = useId()
  return (
    <div className="field">
      <label htmlFor={id}>Operation</label>
      <select
        id={id}
        value={value}
        onChange={(event) => {
          if (isOperation(event.target.value)) onChange(event.target.value)
        }}
      >
        {OPERATIONS.map((op) => (
          <option key={op.value} value={op.value}>
            {op.label}
          </option>
        ))}
      </select>
    </div>
  )
}
