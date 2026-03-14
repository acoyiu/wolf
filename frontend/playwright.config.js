import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig } from '@playwright/test'

const currentDir = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  testDir: './tests',
  timeout: 120000,
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: 'list',
  use: {
    baseURL: 'http://127.0.0.1:4173',
    headless: true,
    trace: 'retain-on-failure',
    video: 'retain-on-failure',
  },
  webServer: {
    command: 'npm --prefix frontend run build && PORT=4173 DAY_TIMEOUT_SEC=20 VOTE_TIMEOUT_SEC=20 go run main.go',
    cwd: path.resolve(currentDir, '..'),
    url: 'http://127.0.0.1:4173/healthz',
    reuseExistingServer: !process.env.CI,
    timeout: 120000,
  },
})
