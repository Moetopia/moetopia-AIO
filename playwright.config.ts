import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './tests/browser',
  fullyParallel: true,
  use: { browserName: 'chromium', trace: 'retain-on-failure' },
  webServer: [
    { command: 'node apps/account/.output/server/index.mjs', url: 'http://127.0.0.1:53000', env: { PORT: '53000', HOST: '127.0.0.1' }, reuseExistingServer: false },
    { command: 'node scripts/static-server.mjs apps/admin/.output/public 53001', url: 'http://127.0.0.1:53001', reuseExistingServer: false },
    { command: 'node scripts/static-server.mjs apps/developer/.output/public 53002', url: 'http://127.0.0.1:53002', reuseExistingServer: false },
    { command: 'node scripts/static-server.mjs packages/ui/storybook-static 56006', url: 'http://127.0.0.1:56006', reuseExistingServer: false },
  ],
})
