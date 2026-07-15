import { defineConfig, devices } from '@playwright/test'
import { resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const webDir = fileURLToPath(new URL('.', import.meta.url))
const localBrowser = process.env.CI ? {} : { channel: 'chrome' as const }

export default defineConfig({
  testDir: './e2e',
  outputDir: '../test_reports/playwright-artifacts',
  fullyParallel: false,
  workers: 1,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI
    ? [['line'], ['html', { outputFolder: '../test_reports/playwright-report', open: 'never' }]]
    : [['list'], ['html', { outputFolder: '../test_reports/playwright-report', open: 'never' }]],
  use: {
    baseURL: 'http://127.0.0.1:18080',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    { name: 'desktop-chromium', use: { ...devices['Desktop Chrome'], ...localBrowser } },
    { name: 'mobile-chromium', use: { ...devices['Pixel 7'], ...localBrowser } },
  ],
  webServer: [
    {
      command: 'node e2e/support/smtp-capture.mjs',
      cwd: webDir,
      url: 'http://127.0.0.1:12526/health',
      reuseExistingServer: !process.env.CI,
      timeout: 30_000,
    },
    {
      command: 'go run ./cmd/event-go',
      cwd: resolve(webDir, '..'),
      url: 'http://127.0.0.1:18080/health',
      reuseExistingServer: !process.env.CI,
      timeout: 120_000,
      env: {
        APP_ENV: 'test',
        ADMIN_TOKEN: 'e2e-admin-token',
        DATABASE_PATH: ':memory:',
        JWT_SECRET: 'e2e-jwt-secret-with-at-least-32-bytes',
        PORT: '18080',
        PUBLIC_BASE_URL: 'http://127.0.0.1:18080',
        SMTP_HOST: '127.0.0.1',
        SMTP_PORT: '12525',
        SMTP_USERNAME: 'e2e',
        SMTP_PASSWORD: 'e2e',
        SMTP_FROM: 'Soulmark E2E <no-reply@example.com>',
        LOG_LEVEL: 'error',
        VERSION: 'e2e',
      },
    },
  ],
})
