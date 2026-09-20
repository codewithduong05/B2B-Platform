import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

const __dirname = dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '#shared': resolve(__dirname, 'shared'),
      '#shared/*': resolve(__dirname, 'shared/*'),
      '#server': resolve(__dirname, 'server'),
      '#server/*': resolve(__dirname, 'server/*'),
      '~': resolve(__dirname, 'app'),
      '~/*': resolve(__dirname, 'app/*'),
      '@': resolve(__dirname, 'app'),
      '@/*': resolve(__dirname, 'app/*'),
    },
  },
  test: {
    environment: 'node',
    include: ['tests/unit/**', 'tests/integration/**', 'tests/component/**'],
    exclude: ['tests/e2e'],
  },
})
