import { describe, expect, it, vi } from 'vitest'
import { ApiError, calculate, NetworkError, resolveBaseUrl } from './calculator'

/** Builds a fetch Response with a JSON body. */
function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

/** Replaces global fetch with a mock for the current test (restored by unstubGlobals). */
function mockFetch(impl: () => Promise<Response>) {
  const fetchMock = vi.fn<typeof fetch>(impl)
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('calculate', () => {
  it('posts the request and returns the result on success', async () => {
    const fetchMock = mockFetch(async () => jsonResponse({ operation: 'divide', result: 5 }))

    await expect(calculate('divide', 10, 2)).resolves.toEqual({ operation: 'divide', result: 5 })

    expect(fetchMock).toHaveBeenCalledOnce()
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://localhost:8080/api/v1/calculate')
    expect(init?.method).toBe('POST')
    expect(init?.headers).toMatchObject({ 'Content-Type': 'application/json' })
    expect(JSON.parse(init?.body as string)).toEqual({ operation: 'divide', a: 10, b: 2 })
  })

  it('omits b for single-operand operations', async () => {
    const fetchMock = mockFetch(async () => jsonResponse({ operation: 'sqrt', result: 3 }))

    await calculate('sqrt', 9)

    const body = JSON.parse(fetchMock.mock.calls[0][1]?.body as string)
    expect(body).toEqual({ operation: 'sqrt', a: 9 })
    expect(body).not.toHaveProperty('b')
  })

  it('throws ApiError with code, message and status on a 422', async () => {
    mockFetch(async () =>
      jsonResponse({ error: { code: 'DIVISION_BY_ZERO', message: 'cannot divide by zero' } }, 422),
    )

    const error = await calculate('divide', 1, 0).catch((e: unknown) => e)

    expect(error).toBeInstanceOf(ApiError)
    expect(error).toMatchObject({
      name: 'ApiError',
      code: 'DIVISION_BY_ZERO',
      message: 'cannot divide by zero',
      status: 422,
    })
  })

  it('throws ApiError UNEXPECTED_RESPONSE when an error body is not JSON', async () => {
    mockFetch(async () => new Response('<html>Bad Gateway</html>', { status: 502 }))

    await expect(calculate('add', 1, 2)).rejects.toMatchObject({
      code: 'UNEXPECTED_RESPONSE',
      status: 502,
    })
  })

  it('throws ApiError UNEXPECTED_RESPONSE when a success body has the wrong shape', async () => {
    mockFetch(async () => jsonResponse({ value: 3 }))

    await expect(calculate('add', 1, 2)).rejects.toMatchObject({
      name: 'ApiError',
      code: 'UNEXPECTED_RESPONSE',
      status: 200,
    })
  })

  it('throws NetworkError when the backend cannot be reached', async () => {
    const cause = new TypeError('Failed to fetch')
    mockFetch(async () => {
      throw cause
    })

    const error = await calculate('add', 1, 2).catch((e: unknown) => e)

    expect(error).toBeInstanceOf(NetworkError)
    expect(error).not.toBeInstanceOf(ApiError)
    expect((error as NetworkError).message).toMatch(/could not reach the calculator api/i)
    expect((error as NetworkError).cause).toBe(cause)
  })

  it('calls the same origin when VITE_API_URL is empty', async () => {
    vi.stubEnv('VITE_API_URL', '')
    const fetchMock = mockFetch(async () => jsonResponse({ operation: 'add', result: 3 }))

    await calculate('add', 1, 2)

    expect(fetchMock.mock.calls[0][0]).toBe('/api/v1/calculate')
  })
})

describe('resolveBaseUrl', () => {
  it.each([
    { name: 'unset in development', env: { DEV: true }, want: 'http://localhost:8080' },
    { name: 'unset in production', env: { DEV: false }, want: '' },
    { name: 'empty means same origin', env: { VITE_API_URL: '', DEV: true }, want: '' },
    { name: 'explicit URL', env: { VITE_API_URL: 'https://api.example.com', DEV: false }, want: 'https://api.example.com' },
    { name: 'trailing slashes removed', env: { VITE_API_URL: 'https://api.example.com//', DEV: false }, want: 'https://api.example.com' },
  ])('$name', ({ env, want }) => {
    expect(resolveBaseUrl(env)).toBe(want)
  })
})
