import { expect, test } from '@playwright/test'
import { mockPortal } from './helpers/portal'

const account = 'http://127.0.0.1:53000'
const admin = 'http://127.0.0.1:53001'
const healthy = async (page: import('@playwright/test').Page, name: string) => {
  const image = page.locator(`[data-illustration="${name}"] img`)
  await expect(image).toBeVisible()
  await expect.poll(() => image.evaluate((node: HTMLImageElement) => node.complete && node.naturalWidth > 1)).toBe(true)
  await expect(image).toHaveAttribute('alt', '')
}

test('authentication art fits desktop including OAuth and is neither displayed nor fetched on mobile', async ({ page }) => {
  await page.route('**/api/**', route => route.fulfill({ status: 503, json: {} }))
  await page.setViewportSize({ width: 1440, height: 800 })
  await page.goto(account + '/auth/login')
  await healthy(page, 'auth-welcome')
  const brand = page.locator('.moe-auth-brand')
  expect(await brand.evaluate(node => node.scrollHeight <= node.clientHeight + 1)).toBe(true)
  await page.screenshot({ path: '.data/previews/illustrations/auth-desktop.png', fullPage: true })
  const mobile = await page.context().newPage()
  await mobile.setViewportSize({ width: 375, height: 812 })
  await mobile.route('**/api/**', route => route.fulfill({ status: 503, json: {} }))
  const fetched: string[] = []
  mobile.on('request', request => { if (request.url().includes('mascot-auth-welcome')) fetched.push(request.url()) })
  await mobile.goto(account + '/auth/login')
  await expect(mobile.locator('[data-illustration="auth-welcome"]')).toBeHidden()
  await expect(mobile.getByRole('heading', { level: 1 })).toBeVisible()
  expect(fetched).toEqual([])
  await mobile.close()
  await page.goto(account + '/auth/authorize')
  await expect(page.getByRole('alert')).toContainText('授权请求格式无效')
  await healthy(page, 'auth-welcome')
})

test('empty applications, failed loading, developer landing and guide each use the matching illustration', async ({ page }) => {
  await mockPortal(page)
  await page.goto(account + '/developer/applications')
  await healthy(page, 'no-applications')
  await expect(page.getByRole('button', { name: '创建应用', exact: true }).last()).toBeEnabled()
  await page.screenshot({ path: '.data/previews/illustrations/applications-empty.png', fullPage: true })
  await page.route('**/api/v1/developer/applications', route => route.fulfill({ status: 503, json: {} }))
  await page.reload()
  await healthy(page, 'service-unavailable')
  await expect(page.locator('[data-illustration="no-applications"]')).toHaveCount(0)
  await page.goto(account + '/developer')
  await healthy(page, 'developer-welcome')
  await expect(page.locator('.moe-code-block')).toContainText('code_challenge_method=S256')
  await page.screenshot({ path: '.data/previews/illustrations/developer-desktop.png', fullPage: true })
  await page.goto(account + '/developer/docs')
  await healthy(page, 'guide-helper')
  await page.goto(account + '/developer/docs/endpoints?q=no-such-endpoint')
  await healthy(page, 'no-results')
})

for (const [locale, title] of [['zh-CN', '好像走到未探索的地方了'], ['zh-TW', '好像走到未探索的地方了'], ['en', 'This path is still unexplored'], ['ja', 'まだ見つかっていない場所のようです']] as const) {
  test(`real SSR 404 preserves status and supports ${locale} dark mobile`, async ({ page }) => {
    await mockPortal(page)
    await page.setViewportSize({ width: 375, height: 812 })
    await page.context().addCookies([{ name: 'moetopia_locale', value: locale, url: account }, { name: 'moetopia_theme', value: 'dark', url: account }])
    const response = await page.goto(account + '/there-is-no-such-page')
    expect(response?.status()).toBe(404)
    await expect(page.getByRole('heading', { level: 1 })).toHaveText(title)
    await healthy(page, 'not-found')
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
    await page.screenshot({ path: `.data/previews/illustrations/404-${locale}-mobile.png`, fullPage: true })
    await page.locator('.moe-status-actions button').click()
    await expect(page).toHaveURL(account + '/profile')
    await expect(page.locator('.moe-status-page')).toHaveCount(0)
  })
}

test('admin SPA displays the shared 404 and can leave it', async ({ page }) => {
  await page.route('**/api/**', route => route.fulfill({ json: { authenticated: true, csrf_token: 'preview-csrf', user: { id: 'preview', display_name: 'Preview' }, mfa: { enrolled: true, verified: true }, items: [] } }))
  await page.goto(admin + '/unknown-section')
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('好像走到未探索的地方了')
  await healthy(page, 'not-found')
  await page.getByRole('button', { name: '返回首页', exact: true }).click()
  await expect(page).toHaveURL(admin + '/')
  await expect(page.locator('.moe-status-page')).toHaveCount(0)
})

test('image failure retains empty-state explanation and creation controls', async ({ page }) => {
  await mockPortal(page)
  await page.route('**/*state-no-applications*.webp', route => route.abort())
  await page.goto(account + '/developer/applications')
  await expect(page.locator('[data-illustration="no-applications"]')).toHaveAttribute('data-failed', 'true')
  await expect(page.locator('.moe-empty h3')).toBeVisible()
  await page.getByRole('button', { name: '创建应用', exact: true }).last().click()
  await expect(page.getByRole('dialog')).toBeVisible()
})
