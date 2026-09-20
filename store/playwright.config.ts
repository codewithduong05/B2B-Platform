import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 30_000,
  use: {
    baseURL: 'http://127.0.0.1:3000',
  },
  webServer: {
    command: 'npm run preview',
    url: 'http://127.0.0.1:3000/api/health',
    timeout: 120_000,
    reuseExistingServer: !process.env.CI,
  },
})
