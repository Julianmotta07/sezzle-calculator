// Global test setup, loaded by Vitest before every test file.
import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach } from 'vitest'

// Unmount anything rendered by React Testing Library between tests.
// (RTL only does this automatically when Vitest globals are enabled.)
afterEach(() => {
  cleanup()
})
