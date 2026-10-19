import { expect, test } from '@playwright/test'

const origin = 'http://127.0.0.1:53000'
const user = { id: 'preview-user', display_name: '预览用户', email: 'preview@example.com' }

test('account profile uses a summary column and top navigation instead of a dashboard sidebar', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 })
  await page.route('**/api/v1/status', route => route.fulfill({ json: { service: 'moetopia' } }))
  let contexts = 0
  await page.route('**/api/v1/account/context', route => { contexts++; return route.fulfill({ json: { user, csrf_token: 'example-csrf', registration_mode: 'invite' } }) })
  await page.route('**/api/v1/account/sessions', route => route.fulfill({ json: { items: [], current_id: 'preview' } }))
  await page.goto(`${origin}/profile`)
  await expect(page.getByRole('heading', { name: '个人资料', exact: true })).toBeVisible()
  await expect(page.locator('.moe-page-frame')).toHaveCSS('opacity', '1')
  await expect(page.locator('.moe-workspace-sidebar')).toHaveCount(0)
  const summary = await page.locator('.moe-profile-summary').boundingBox()
  const content = await page.locator('.moe-profile-content').boundingBox()
  expect(summary && content && summary.x < content.x).toBeTruthy()
  await page.getByRole('navigation', { name: '主导航' }).getByRole('link', { name: '开发者门户', exact: true }).click()
  await expect(page).toHaveURL(`${origin}/developer`)
  await page.getByRole('link', { name: '查看我的应用', exact: true }).click()
  await expect(page).toHaveURL(`${origin}/developer/applications`)
  await expect(page.getByText(user.email, { exact: true }).first()).toBeVisible()
  expect(contexts).toBe(1)
})

test('developer guide and applications work in the user site without fabricated application data', async ({ page }) => {
  await page.route('**/api/**', route => route.fulfill({ status: 503, json: {} }))
  await page.goto(`${origin}/developer/docs`)
  await expect(page.getByRole('heading', { name: '接入说明', exact: true })).toBeVisible()
  await page.getByRole('navigation', { name: '主导航' }).getByRole('link', { name: '控制台', exact: true }).click()
  await expect(page).toHaveURL(`${origin}/developer/applications`)
  await expect(page.getByRole('heading', { name: '应用接入尚未开放', exact: true })).toBeVisible()
  await page.getByRole('link', { name: '前往登录', exact: true }).click()
  await expect(page).toHaveURL(`${origin}/auth/login?continue=developer-applications`)
  await expect(page.getByLabel('密码', { exact: true })).toBeVisible()
})

test('dedicated registration route supports history and is private', async ({ page }) => {
  await page.route('**/api/**', route => route.fulfill({ status: 503, json: {} }))
  await page.goto(`${origin}/auth/login`)
  await page.getByRole('link', { name: '立即注册', exact: true }).click()
  await expect(page).toHaveURL(`${origin}/auth/register`)
  await expect(page.getByLabel('昵称', { exact: true })).toBeVisible()
  await page.goBack()
  await expect(page.getByRole('heading', { name: '欢迎回来', exact: true })).toBeVisible()
  const response = await page.goto(`${origin}/auth/register`)
  expect(response?.headers()['cache-control']).toContain('no-store')
  expect(response?.headers()['referrer-policy']).toBe('no-referrer')
})

test('authentication shell has independent form scrolling and no candidate artwork', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 800 })
  await page.route('**/api/**', route => route.fulfill({ status: 503, json: {} }))
  await page.goto(`${origin}/auth/register`)
  await expect(page.locator('.moe-auth-brand')).toBeVisible()
  await expect(page.locator('.moe-auth-shell img')).toHaveCount(0)
  expect(await page.locator('.moe-auth-column').evaluate(element => element.scrollHeight > element.clientHeight)).toBe(true)
  expect(await page.evaluate(() => document.documentElement.scrollHeight <= innerHeight)).toBe(true)
  await page.setViewportSize({ width: 375, height: 812 })
  await expect(page.locator('.moe-auth-brand')).toBeHidden()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
})

test('developer sign-in returns to applications and ignores arbitrary continuation URLs', async ({ page }) => {
  await page.route('**/api/v1/status', route => route.fulfill({ json: { service: 'moetopia' } }))
  await page.route('**/api/v1/account/context', route => route.fulfill({ json: { csrf_token: 'example-csrf', registration_mode: 'invite' } }))
  await page.route('**/api/v1/account/login', route => route.fulfill({ json: { user, csrf_token: 'fresh-csrf', registration_mode: 'invite' } }))
  await page.route('**/api/v1/account/sessions', route => route.fulfill({ json: { items: [], current_id: 'preview' } }))
  for (const [continuation, destination] of [['developer-applications', '/developer/applications'], ['https://external.example/redirect', '/profile']]) {
    await page.goto(`${origin}/auth/login?continue=${encodeURIComponent(continuation)}`)
    await page.getByLabel('邮箱', { exact: true }).fill(user.email)
    await page.getByLabel('密码', { exact: true }).fill('example passphrase only')
    await page.getByRole('button', { name: '登录', exact: true }).click()
    await expect(page).toHaveURL(`${origin}${destination}`)
  }
})
