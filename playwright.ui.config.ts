import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './tests/ui',
  fullyParallel: true,
  use: { baseURL: 'http://127.0.0.1:56006', browserName: 'chromium', trace: 'retain-on-failure' },
  webServer: { command: 'node scripts/static-server.mjs packages/ui/storybook-static 56006', url: 'http://127.0.0.1:56006', reuseExistingServer: false },
})
