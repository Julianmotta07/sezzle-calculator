// Typed client for the Go calculator REST API.

/** Operations supported by POST /api/v1/calculate. */
export type Operation = 'add' | 'subtract' | 'multiply' | 'divide' | 'power' | 'sqrt' | 'percentage'

/** Body of POST /api/v1/calculate. `b` is omitted for single-operand operations (sqrt). */
export interface CalculateRequest {
  operation: Operation
  a: number
  b?: number
}

/** Success body of POST /api/v1/calculate. */
export interface CalculateResponse {
  operation: Operation
  result: number
}

/** Error codes the backend can return, plus UNEXPECTED_RESPONSE added by this client. */
export type ApiErrorCode =
  | 'INVALID_JSON'
  | 'UNKNOWN_FIELD'
  | 'PAYLOAD_TOO_LARGE'
  | 'MISSING_OPERATION'
  | 'UNKNOWN_OPERATION'
  | 'MISSING_OPERAND'
  | 'UNEXPECTED_OPERAND'
  | 'DIVISION_BY_ZERO'
  | 'NEGATIVE_SQRT'
  | 'NON_FINITE_RESULT'
  | 'NOT_FOUND'
  | 'METHOD_NOT_ALLOWED'
  | 'INTERNAL_ERROR'
  | 'UNEXPECTED_RESPONSE'

/** Shape shared by every backend error response. */
export interface ApiErrorBody {
  error: {
    code: ApiErrorCode
    message: string
  }
}

/** The backend answered with an error (non-2xx) or with a body this client cannot read. */
export class ApiError extends Error {
  readonly code: ApiErrorCode
  readonly status: number

  constructor(code: ApiErrorCode, message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
  }
}

/** The request never got a response: backend down, wrong URL, CORS rejection, offline. */
export class NetworkError extends Error {
  constructor(message: string, options?: ErrorOptions) {
    super(message, options)
    this.name = 'NetworkError'
  }
}

const DEV_API_URL = 'http://localhost:8080'

/**
 * Picks the API base URL from Vite env values.
 * - VITE_API_URL set (even to ""): used as-is, minus trailing slashes. "" means same origin.
 * - VITE_API_URL unset: http://localhost:8080 in development, same origin otherwise.
 */
export function resolveBaseUrl(env: { VITE_API_URL?: string; DEV: boolean }): string {
  if (env.VITE_API_URL !== undefined) {
    return env.VITE_API_URL.trim().replace(/\/+$/, '')
  }
  return env.DEV ? DEV_API_URL : ''
}

/**
 * Asks the backend to run one calculation.
 *
 * @throws {ApiError} when the backend returns a non-2xx status or an unreadable body.
 * @throws {NetworkError} when the backend cannot be reached.
 */
export async function calculate(operation: Operation, a: number, b?: number): Promise<CalculateResponse> {
  const url = `${resolveBaseUrl(import.meta.env)}/api/v1/calculate`
  const body: CalculateRequest = b === undefined ? { operation, a } : { operation, a, b }

  let response: Response
  try {
    response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
      body: JSON.stringify(body),
    })
  } catch (cause) {
    throw new NetworkError(`Could not reach the calculator API at ${url}. Is the backend running?`, { cause })
  }

  if (!response.ok) {
    throw await toApiError(response)
  }

  const data = await readJson(response)
  if (!isCalculateResponse(data)) {
    throw new ApiError('UNEXPECTED_RESPONSE', 'The server returned an unexpected response.', response.status)
  }
  return data
}

/** Builds an ApiError from an error response, falling back when the body is not our error shape. */
async function toApiError(response: Response): Promise<ApiError> {
  const data = await readJson(response)
  if (isApiErrorBody(data)) {
    return new ApiError(data.error.code, data.error.message, response.status)
  }
  return new ApiError('UNEXPECTED_RESPONSE', `Request failed with status ${response.status}.`, response.status)
}

/** Parses a JSON body, returning undefined instead of throwing on invalid JSON. */
async function readJson(response: Response): Promise<unknown> {
  try {
    return await response.json()
  } catch {
    return undefined
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}

function isCalculateResponse(value: unknown): value is CalculateResponse {
  return isRecord(value) && typeof value.operation === 'string' && typeof value.result === 'number'
}

function isApiErrorBody(value: unknown): value is ApiErrorBody {
  return (
    isRecord(value) &&
    isRecord(value.error) &&
    typeof value.error.code === 'string' &&
    typeof value.error.message === 'string'
  )
}
