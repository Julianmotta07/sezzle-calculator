import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError, calculate, type CalculateResponse, NetworkError } from '../api/calculator'
import { Calculator } from './Calculator'

// Keep the real ApiError/NetworkError classes; replace only the network call.
vi.mock('../api/calculator', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../api/calculator')>()),
  calculate: vi.fn(),
}))

const calculateMock = vi.mocked(calculate)

/** Renders the calculator and returns a user-event instance plus common elements. */
function setup() {
  const user = userEvent.setup()
  render(<Calculator />)
  return {
    user,
    operation: () => screen.getByLabelText('Operation'),
    first: () => screen.getByLabelText(/^(first number|number)$/i),
    second: () => screen.queryByLabelText('Second number'),
    button: () => screen.getByRole('button', { name: /calculat/i }),
    status: () => screen.getByRole('status'),
    alert: () => screen.getByRole('alert'),
  }
}

describe('Calculator', () => {
  beforeEach(() => {
    calculateMock.mockReset()
  })

  it('calculates and shows the result', async () => {
    calculateMock.mockResolvedValue({ operation: 'divide', result: 5 })
    const ui = setup()

    await ui.user.selectOptions(ui.operation(), 'divide')
    await ui.user.type(ui.first(), '10')
    await ui.user.type(ui.second()!, '2')
    await ui.user.click(ui.button())

    expect(calculateMock).toHaveBeenCalledExactlyOnceWith('divide', 10, 2)
    expect(await screen.findByText('5')).toBeInTheDocument()
    expect(ui.status()).toHaveTextContent('10 ÷ 2 = 5')
    expect(ui.alert()).toBeEmptyDOMElement()
  })

  it('submits with the Enter key and accepts negatives and decimals', async () => {
    calculateMock.mockResolvedValue({ operation: 'add', result: -1.25 })
    const ui = setup()

    await ui.user.type(ui.first(), '-1.5')
    await ui.user.type(ui.second()!, '0.25{Enter}')

    expect(calculateMock).toHaveBeenCalledExactlyOnceWith('add', -1.5, 0.25)
    expect(await screen.findByText('-1.25')).toBeInTheDocument()
  })

  it('disables the button while the request is in progress', async () => {
    let resolve!: (value: CalculateResponse) => void
    calculateMock.mockReturnValue(new Promise((r) => (resolve = r)))
    const ui = setup()

    await ui.user.type(ui.first(), '2')
    await ui.user.type(ui.second()!, '3')
    await ui.user.click(ui.button())

    expect(ui.button()).toBeDisabled()
    expect(ui.button()).toHaveTextContent('Calculating…')
    expect(ui.status()).toHaveTextContent('Calculating…')

    resolve({ operation: 'add', result: 5 })

    expect(await screen.findByText('5')).toBeInTheDocument()
    expect(ui.button()).toBeEnabled()
    expect(ui.button()).toHaveTextContent('Calculate')
  })

  describe('client-side validation', () => {
    it('reports empty inputs and does not call the API', async () => {
      const ui = setup()

      await ui.user.click(ui.button())

      expect(calculateMock).not.toHaveBeenCalled()
      expect(ui.first()).toHaveAccessibleDescription('Enter the first number.')
      expect(ui.second()).toHaveAccessibleDescription('Enter the second number.')
      expect(ui.first()).toHaveAttribute('aria-invalid', 'true')
      expect(ui.first()).toHaveFocus()
    })

    it('reports non-numeric input and focuses the invalid field', async () => {
      const ui = setup()

      await ui.user.type(ui.first(), '4')
      await ui.user.type(ui.second()!, 'abc')
      await ui.user.click(ui.button())

      expect(calculateMock).not.toHaveBeenCalled()
      expect(ui.second()).toHaveAccessibleDescription(/"abc" is not a valid number/)
      expect(ui.second()).toHaveFocus()
      expect(ui.first()).not.toHaveAttribute('aria-invalid')
    })

    it('clears a field error once the user edits that field', async () => {
      const ui = setup()

      await ui.user.click(ui.button())
      await ui.user.type(ui.first(), '1')

      expect(ui.first()).not.toHaveAttribute('aria-invalid')
      expect(screen.queryByText('Enter the first number.')).not.toBeInTheDocument()
      expect(screen.getByText('Enter the second number.')).toBeInTheDocument()
    })
  })

  describe('API errors', () => {
    it('shows a friendly message for division by zero', async () => {
      calculateMock.mockRejectedValue(new ApiError('DIVISION_BY_ZERO', 'cannot divide by zero', 422))
      const ui = setup()

      await ui.user.selectOptions(ui.operation(), 'divide')
      await ui.user.type(ui.first(), '1')
      await ui.user.type(ui.second()!, '0')
      await ui.user.click(ui.button())

      expect(await screen.findByText(/you can't divide by zero/i)).toBeInTheDocument()
      expect(ui.alert()).toHaveTextContent("You can't divide by zero. Try a different second number.")
      expect(ui.status()).not.toHaveTextContent('=')
      expect(ui.button()).toBeEnabled()
    })

    it('shows a friendly message when the server is unreachable', async () => {
      calculateMock.mockRejectedValue(new NetworkError('Failed to fetch'))
      const ui = setup()

      await ui.user.type(ui.first(), '1')
      await ui.user.type(ui.second()!, '2')
      await ui.user.click(ui.button())

      expect(await screen.findByRole('alert')).toHaveTextContent(/can't reach the calculator server/i)
    })
  })

  describe('square root', () => {
    it('hides the second input and sends a single operand', async () => {
      calculateMock.mockResolvedValue({ operation: 'sqrt', result: 3 })
      const ui = setup()
      expect(ui.second()).toBeInTheDocument()

      await ui.user.selectOptions(ui.operation(), 'sqrt')

      expect(ui.second()).not.toBeInTheDocument()
      expect(screen.getByLabelText('Number')).toBeInTheDocument()

      await ui.user.type(ui.first(), '9')
      await ui.user.click(ui.button())

      expect(calculateMock).toHaveBeenCalledExactlyOnceWith('sqrt', 9)
      expect(await screen.findByText('3')).toBeInTheDocument()
      expect(ui.status()).toHaveTextContent('√9 = 3')
    })

    it('shows the second input again, keeping its value, when switching back', async () => {
      const ui = setup()

      await ui.user.type(ui.second()!, '7')
      await ui.user.selectOptions(ui.operation(), 'sqrt')
      await ui.user.selectOptions(ui.operation(), 'power')

      expect(ui.second()).toHaveValue('7')
    })
  })
})
