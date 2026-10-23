import { expect, test } from '@playwright/test'
import { mockPortal, previewUser } from './helpers/portal'
const account = 'http://127.0.0.1:53000'
const admin = 'http://127.0.0.1:53001'

test('brand image failure preserves its text fallback and accessible site name', async ({ page }) => {
  await mockPortal(page)
  await page.route('**/*mascot-face-neutral*.webp', route => route.abort())
  await page.goto(account + '/developer')
  const brand = page.locator('.moe-navbar-brand')
  await expect(brand.locator('.moe-brand-mark')).toHaveText('M')
  await expect(brand).toHaveAccessibleName('Moetopia 开发者门户')
  await page.setViewportSize({ width: 375, height: 812 })
  await expect(brand.locator('.moe-brand-mark')).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
})

test('OAuth uses the mascot in the desktop brand panel and mobile provider mark without deciding consent', async ({ page }) => {
  await mockPortal(page)
  const id = 'c'.repeat(43)
  let decisions = 0
  await page.route('**/api/v1/account/context', route => route.fulfill({ json: { user: previewUser, csrf_token: 'preview-csrf', registration_mode: 'invite' } }))
  await page.route(`**/api/v1/account/authorizations/${id}`, route => {
    if (route.request().method() !== 'GET') decisions++
    return route.fulfill({ json: { user: previewUser, client_id: 'example', client_name: '同人图书馆', redirect_uri: 'https://example.test/callback', scopes: ['openid', 'profile'], audiences: [], expires_at: '2050-01-01T00:00:00Z', login_required: false } })
  })
  await page.setViewportSize({ width: 1440, height: 1000 })
  await page.goto(`${account}/auth/authorize?authorization_request=${id}`)
  const provider = page.locator('.moe-authorization-provider .moe-brand-mark img')
  await expect(provider).toBeVisible()
  await expect.poll(() => provider.evaluate((image: HTMLImageElement) => image.complete && image.naturalWidth > 1)).toBe(true)
  await expect(page.locator('.moe-auth-brand [data-illustration="auth-welcome"]')).toBeVisible()
  await expect(page.getByRole('button', { name: '允许访问', exact: true })).toBeEnabled()
  await expect(page.getByRole('button', { name: '拒绝', exact: true })).toBeEnabled()
  await page.screenshot({ path: '.data/previews/illustrations/oauth-desktop.png', fullPage: true })
  await page.setViewportSize({ width: 375, height: 812 })
  await expect(provider).toBeVisible()
  await expect(page.locator('.moe-auth-brand')).toBeHidden()
  await expect(page.getByRole('button', { name: '允许访问', exact: true })).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  await page.screenshot({ path: '.data/previews/illustrations/oauth-mobile.png', fullPage: true })
  expect(decisions).toBe(0)
})

for (const origin of [account, admin]) {
  test(`${origin} serves shared site icons with valid dimensions and ICO frames`, async ({ page, request }) => {
    await page.route('**/api/**', route => route.fulfill({ status: 503, json: {} }))
    await page.goto(origin + (origin === account ? '/auth/login' : '/'))
    await expect(page.locator('head link[rel="icon"][href="/brand/favicon-32.png"]')).toHaveCount(1)
    await expect(page.locator('head link[rel="apple-touch-icon"]')).toHaveAttribute('href', '/brand/apple-touch-icon.png')
    for (const [file, size] of [['favicon-32.png', 32], ['favicon-48.png', 48], ['apple-touch-icon.png', 180]] as const) {
      const response = await request.get(`${origin}/brand/${file}`)
      expect(response.status()).toBe(200)
      expect(response.headers()['content-type']).toContain('image/png')
      const png = await response.body()
      expect(png.subarray(1, 4).toString()).toBe('PNG')
      expect(png.readUInt32BE(16)).toBe(size)
      expect(png.readUInt32BE(20)).toBe(size)
    }
    const response = await request.get(origin + '/favicon.ico')
    expect(response.status()).toBe(200)
    expect(response.headers()['content-type']).toMatch(/image\/(?:x-icon|vnd.microsoft.icon)/)
    const ico = await response.body()
    expect(ico.readUInt16LE(2)).toBe(1)
    expect(ico.readUInt16LE(4)).toBe(4)
    expect([0, 1, 2, 3].map(index => ico[6 + index * 16])).toEqual([16, 32, 48, 64])
    await page.setViewportSize({ width: 375, height: 812 })
    const mark = page.locator(origin === account ? '.moe-auth-header .moe-brand-mark img' : '.moe-navbar .moe-brand-mark img')
    await expect(mark).toBeVisible()
  })
}
