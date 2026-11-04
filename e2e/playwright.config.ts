import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig, devices } from '@playwright/test'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const e2eHost = process.env.AIHUB_E2E_HOST ?? '127.0.0.1'
const loopbackPort = process.env.AIHUB_E2E_PORT ?? '18080'
const passphrasePort = process.env.AIHUB_E2E_PASSPHRASE_PORT ?? '18081'
const loopbackBaseURL = `http://${e2eHost}:${loopbackPort}`
const passphraseBaseURL = `http://${e2eHost}:${passphrasePort}`

export default defineConfig({
  testDir: './tests',
  fullyParallel: false,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: [['list']],
  projects: [
    {
      name: 'loopback-chromium',
      testMatch: /smoke\.spec\.ts|gate-zero-screenshots\.spec\.ts/,
      use: { ...devices['Desktop Chrome'], baseURL: loopbackBaseURL },
    },
    {
      name: 'passphrase-chromium',
      testMatch: /operator-auth\.spec\.ts/,
      use: { ...devices['Desktop Chrome'], baseURL: passphraseBaseURL },
    },
  ],
  webServer: process.env.AIHUB_E2E_EXTERNAL
    ? undefined
    : [
        {
          command: `bash ${path.join(root, 'scripts/e2e-hub.sh')} ${loopbackPort}`,
          cwd: root,
          url: `${loopbackBaseURL}/healthz`,
          reuseExistingServer: !process.env.CI,
          timeout: 120_000,
        },
        {
          command: `bash ${path.join(root, 'scripts/e2e-hub-passphrase.sh')} ${passphrasePort}`,
          cwd: root,
          url: `${passphraseBaseURL}/healthz`,
          reuseExistingServer: !process.env.CI,
          timeout: 120_000,
        },
      ],
})
