import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './tests/account',
  workers: 1,
  timeout: 60000,
  use: { browserName: 'chromium', baseURL: 'http://127.0.0.1:4305', trace: 'off', screenshot: 'off' },
  webServer: [
    { command: 'node scripts/test-account-api.mjs', url: 'http://127.0.0.1:58080/readyz', reuseExistingServer: false },
    { command: 'node apps/account/.output/server/index.mjs', url: 'http://127.0.0.1:4305', env: { PORT: '4305', HOST: '127.0.0.1' }, reuseExistingServer: false },
  ],
})
