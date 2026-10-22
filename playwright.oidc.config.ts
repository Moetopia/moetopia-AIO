import { defineConfig } from '@playwright/test'
import { resolve } from 'node:path'
import { oidcAPIOrigin } from './scripts/oidc-test-config.mjs'

export default defineConfig({
  testDir: './tests/oidc', workers: 1, timeout: 90000, reporter: 'line',
  testMatch: process.env.MOETOPIA_TEST_OIDC_SUITE === 'admin' ? 'admin.spec.ts' : process.env.MOETOPIA_TEST_OIDC_SUITE === 'management' ? 'management.spec.ts' : 'interop.spec.ts',
  outputDir: resolve(process.env.MOETOPIA_TEST_OIDC_DIRECTORY ?? '.data/oidc-playwright', 'playwright'), preserveOutput: 'never',
  use: { browserName: 'chromium', baseURL: 'http://127.0.0.1:54305', trace: 'off', screenshot: 'off', video: 'off' },
  webServer: [
    { command: 'node scripts/oidc-test-server.mjs api', url: oidcAPIOrigin() + '/readyz', reuseExistingServer: false },
    { command: 'node scripts/oidc-test-server.mjs account', url: 'http://127.0.0.1:54305', reuseExistingServer: false },
    { command: 'node scripts/oidc-test-rp.mjs', url: 'http://localhost:54310/healthz', reuseExistingServer: false },
    ...(process.env.MOETOPIA_TEST_OIDC_SUITE === 'admin' ? [{ command: 'node scripts/admin-test-proxy.mjs', url: 'http://127.0.0.1:54311/healthz', reuseExistingServer: false }] : []),
  ],
})
