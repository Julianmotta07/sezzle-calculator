import { type Ref, useId } from 'react'

interface NumberFieldProps {
  label: string
  value: string
  onChange: (value: string) => void
  error?: string
  ref?: Ref<HTMLInputElement>
}

/**
 * Text input for a number, with its label and validation message.
 * Uses type="text" + inputMode="decimal" (numeric keypad on mobile) rather
 * than type="number", so invalid text reaches our validation and gets a clear message.
 */
export function NumberField({ label, value, onChange, error, ref }: NumberFieldProps) {
  const id = useId()
  const errorId = `${id}-error`
  return (
    <div className="field">
      <label htmlFor={id}>{label}</label>
      <input
        id={id}
        ref={ref}
        type="text"
        inputMode="decimal"
        autoComplete="off"
        spellCheck={false}
        placeholder="0"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? errorId : undefined}
      />
      {error && (
        <p id={errorId} className="field-error">
          {error}
        </p>
      )}
    </div>
  )
}
