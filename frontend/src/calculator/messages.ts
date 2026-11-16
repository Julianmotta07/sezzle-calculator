import { ApiError, type ApiErrorCode, NetworkError } from '../api/calculator'

/** User-facing wording for errors the user can act on. */
const API_MESSAGES: Partial<Record<ApiErrorCode, string>> = {
  DIVISION_BY_ZERO: "You can't divide by zero. Try a different second number.",
  NEGATIVE_SQRT: 'The square root needs a number that is zero or greater.',
  NON_FINITE_RESULT: 'The result is too large or undefined. Try smaller numbers.',
}

/** Turns anything thrown by the API client into a friendly message. */
export function friendlyErrorMessage(error: unknown): string {
  if (error instanceof NetworkError) {
    return "Can't reach the calculator server. Check your connection and that the backend is running, then try again."
  }
  if (error instanceof ApiError) {
    const known = API_MESSAGES[error.code]
    if (known) {
      return known
    }
    if (error.status >= 500 || error.code === 'UNEXPECTED_RESPONSE') {
      return 'The server had a problem. Please try again in a moment.'
    }
    return `The calculation was rejected: ${error.message}`
  }
  return 'Something unexpected went wrong. Please try again.'
}
