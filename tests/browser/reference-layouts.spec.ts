import { expect, test, type Page } from '@playwright/test'

const origin = 'http://127.0.0.1:53000'
const user = { id: 'preview', display_name: '预览用户', email: 'preview@example.com' }
const setup = async (page: Page) => {
  await page.route('**/api/v1/status', route => route.fulfill({ json: { service: 'moetopia' } }))
  await page.route('**/api/v1/account/context', route => route.fulfill({ json: { user, csrf_token: 'preview-csrf', registration_mode: 'invite' } }))
  await page.route('**/api/v1/portal/context', route => route.fulfill({ json: { authenticated: true, user, csrf_token: 'portal-csrf', login_url: '/api/v1/portal/login' } }))
  await page.route('**/api/v1/portal/sessions', route => route.fulfill({ json: { items: [], current_id: 'preview' } }))
}

test('developer console follows the reference dimensions and account placement', async ({ page }) => {
  await setup(page)
  await page.setViewportSize({ width: 1440, height: 1000 })
  await page.goto(`${origin}/developer/applications`)
  await expect(page.locator('.moe-console-sidebar')).toBeVisible()
  await expect(page.locator('.moe-navbar')).toHaveCSS('height', '64px')
  await expect(page.locator('.moe-page-main')).toHaveCSS('max-width', '1280px')
  await expect(page.locator('.moe-console-sidebar')).toHaveCSS('flex-basis', '240px')
  await expect(page.locator('.moe-console-layout')).toHaveCSS('gap', '32px')
  await expect(page.locator('.moe-console-sidebar .moe-sidebar')).toHaveCount(0)
  await expect(page.locator('.moe-console-sidebar').getByRole('button', { name: '退出登录' })).toBeVisible()
  await page.getByRole('button', { name: '创建应用', exact: true }).first().click()
  await expect(page.getByRole('dialog').getByRole('button', { name: '创建应用', exact: true })).toBeDisabled()
  await page.keyboard.press('Escape')
  await page.setViewportSize({ width: 820, height: 900 })
  await expect(page.locator('.moe-console-sidebar')).toBeHidden()
  await page.getByRole('button', { name: '打开导航' }).click()
  const drawer = page.getByRole('dialog', { name: '主导航' })
  await drawer.getByRole('link', { name: '接入准备', exact: true }).click()
  await expect(drawer).toHaveCount(0)
  await expect(page).toHaveURL(`${origin}/developer/docs`)
})

test('profile uses the reference width, grouped panels and avatar account navigation', async ({ page }) => {
  await setup(page)
  await page.setViewportSize({ width: 1440, height: 1000 })
  await page.goto(`${origin}/profile`)
  await expect(page.locator('.moe-page-frame')).toHaveCSS('opacity', '1')
  await expect(page.locator('.moe-navbar')).toHaveCSS('height', '64px')
  await expect(page.locator('.moe-page-main')).toHaveCSS('max-width', '1024px')
  await expect(page.locator('.moe-profile-layout')).toHaveCSS('gap', '24px')
  expect((await page.locator('.moe-profile-summary').boundingBox())?.width).toBe(304)
  await expect(page.getByLabel('昵称', { exact: true })).toHaveValue(user.display_name)
  await expect(page.getByLabel('昵称', { exact: true })).toHaveAttribute('readonly', '')
  await expect(page.locator('.moe-card-form-heading').first()).toHaveCSS('border-bottom-width', '0px')
  await page.getByRole('button', { name: '你的账号', exact: true }).click()
  await page.getByRole('menuitem', { name: '登录会话', exact: true }).click()
  await expect(page).toHaveURL(`${origin}/sessions`)
})

test('dedicated authorization route rejects missing requests without displaying a fake consent', async ({ page }) => {
  await setup(page)
  const response = await page.goto(`${origin}/auth/authorize`)
  expect(response?.headers()['cache-control']).toContain('no-store')
  expect(response?.headers()['content-security-policy']).toContain("frame-ancestors 'none'")
  await expect(page.getByRole('alert')).toContainText('授权请求格式无效')
  await expect(page.getByRole('button', { name: '允许访问' })).toHaveCount(0)
})

test('dedicated consent displays only claims requested by scopes', async ({ page }) => {
  await setup(page)
  const id = 'b'.repeat(43)
  let decisions = 0
  await page.route(`**/api/v1/account/authorizations/${id}`, route => {
    if (route.request().method() !== 'GET') decisions++
    return route.fulfill({ json: { client_id: 'preview', client_name: '同人图书馆', redirect_uri: 'http://localhost:53110/callback', scopes: ['openid'], audiences: [], expires_at: '2050-01-01T00:00:00Z', login_required: false, user } })
  })
  await page.goto(`${origin}/auth/authorize?authorization_request=${id}`)
  await expect(page.getByRole('button', { name: '允许访问' })).toBeEnabled()
  await expect(page.locator('.moe-authorization-handshake')).toBeVisible()
  await expect(page.locator('.moe-consent-details')).toContainText(user.id)
  await expect(page.locator('.moe-consent-details')).not.toContainText(user.email)
  expect(decisions).toBe(0)
})

for (const path of ['/profile', '/developer/applications']) {
  test(`signed-in ${path} fits a Japanese mobile viewport`, async ({ page }) => {
    await setup(page)
    await page.setViewportSize({ width: 375, height: 812 })
    await page.context().addCookies([{ name: 'moetopia_locale', value: 'ja', url: origin }])
    await page.goto(`${origin}${path}`)
    await expect(page.getByRole('button', { name: 'アカウント', exact: true })).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  })
}
