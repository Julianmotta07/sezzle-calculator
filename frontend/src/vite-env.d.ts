/// <reference types="vite/client" />

interface ImportMetaEnv {
  /**
   * Base URL of the Go backend, without a trailing slash.
   * Unset in development: http://localhost:8080. Empty: same origin.
   */
  readonly VITE_API_URL?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
