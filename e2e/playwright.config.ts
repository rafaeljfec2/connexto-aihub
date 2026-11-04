import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig, devices } from '@playwright/test'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const hubPort = process.env.AIHUB_E2E_PORT ?? '18080'
const baseURL = `http://127.0.0.1:${hubPort}`

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: [['list']],
  use: {
    baseURL,
    trace: 'on-first-retry',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: {
    command: `bash ${path.join(root, 'scripts/e2e-hub.sh')} ${hubPort}`,
    cwd: root,
    url: `${baseURL}/healthz`,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
})
