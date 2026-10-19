import { expect, test } from '@playwright/test'
import { mockPortal } from './helpers/portal'

const origin = 'http://127.0.0.1:53000'

test('identity cookies alone never grant a protected portal page', async ({ page }) => {
  await mockPortal(page, false)
  let identityReads = 0
  await page.route('**/api/v1/account/context', route => { identityReads++; return route.fulfill({ json: { user: { id: 'identity-only' } } }) })
  await page.route('**/api/v1/portal/login?**', route => route.fulfill({ body: 'OAuth redirect', contentType: 'text/plain' }))
  await page.goto(`${origin}/profile`)
  await expect(page).toHaveURL(`${origin}/api/v1/portal/login?return_to=%2Fprofile`)
  expect(identityReads).toBe(0)
})

test('portal dependency failure remains a retryable error instead of triggering a login loop', async ({ page }) => {
  await page.route('**/api/**', route => route.fulfill({ status: 503, json: {} }))
  let starts = 0
  await page.route('**/api/v1/portal/login?**', route => { starts++; return route.fulfill({ body: 'Unexpected login' }) })
  await page.goto(`${origin}/profile`)
  await expect(page.getByRole('alert')).toContainText('暂时无法连接账号服务')
  await expect(page.getByRole('button', { name: '重新连接' })).toBeVisible()
  expect(starts).toBe(0)
})

test('public developer documentation keeps its reference sidebar and real protocol paths', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 1000 })
  await page.route('**/api/**', route => route.fulfill({ status: 503, json: {} }))
  await page.goto(`${origin}/developer/docs`)
  await expect(page.getByRole('heading', { name: '接入说明', exact: true })).toBeVisible()
  await expect(page.locator('.moe-docs-sidebar')).toBeVisible()
  await page.getByRole('navigation', { name: '文档导航' }).getByRole('link', { name: '协议端点' }).click()
  await expect(page).toHaveURL(`${origin}/developer/docs/endpoints`)
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('端点参考')
  await expect(page.getByRole('navigation', { name: '主导航' }).getByRole('link', { name: '接入准备', exact: true })).toHaveAttribute('aria-current', 'page')
  await expect(page.getByRole('heading', { name: '接入前准备', exact: true })).toHaveCount(0)
  await expect(page.getByRole('table')).toContainText('/oauth/token')
})

test('developer login opens a modal and starts OAuth with a fixed destination', async ({ page }) => {
  await mockPortal(page, false)
  await page.route('**/api/v1/portal/login?**', route => route.fulfill({ body: 'OAuth started' }))
  await page.goto(`${origin}/developer`)
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await page.getByRole('dialog').getByRole('button', { name: '使用 Moetopia 账号登录', exact: true }).click()
  await expect(page).toHaveURL(`${origin}/api/v1/portal/login?return_to=%2Fdeveloper%2Fapplications`)
})

test('application creation opens the reference form without faking a successful API', async ({ page }) => {
  await mockPortal(page)
  await page.goto(`${origin}/developer/applications`)
  await page.getByRole('button', { name: '创建应用', exact: true }).first().click()
  const dialog = page.getByRole('dialog', { name: '创建应用' })
  await expect(dialog).toBeVisible()
  await dialog.getByRole('textbox', { name: '应用名称与说明' }).fill('A review draft')
  await expect(dialog.getByRole('button', { name: '创建应用', exact: true })).toBeDisabled()
  await expect(dialog).toContainText('应用创建接口尚未开放')
  await page.keyboard.press('Escape')
  await expect(dialog).toHaveCount(0)
})

test('recovery entry preserves authorization and does not submit unsupported recovery', async ({ page }) => {
  const id = 'a'.repeat(43)
  await page.route('**/api/v1/account/context', route => route.fulfill({ json: { csrf_token: 'identity-csrf', registration_mode: 'invite' } }))
  await page.goto(`${origin}/auth/login?authorization_request=${id}`)
  await page.getByRole('link', { name: '忘记密码？' }).click()
  await expect(page).toHaveURL(`${origin}/auth/forgot-password?authorization_request=${id}`)
  await expect(page.getByRole('button', { name: '发送验证码' })).toBeDisabled()
  await expect(page.getByRole('link', { name: '返回登录' })).toHaveAttribute('href', `/auth/authorize?authorization_request=${id}`)
})

test('registration history is private and the illustrated layout never loads candidate art', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 800 })
  await page.route('**/api/**', route => route.fulfill({ status: 503, json: {} }))
  await page.goto(`${origin}/auth/login`)
  await page.getByRole('link', { name: '立即注册', exact: true }).click()
  await expect(page).toHaveURL(`${origin}/auth/register`)
  await expect(page.locator('.moe-auth-shell img')).toHaveCount(0)
  expect(await page.locator('.moe-auth-column').evaluate(element => element.scrollHeight > element.clientHeight)).toBe(true)
  await page.goBack()
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('欢迎回来')
  const response = await page.goto(`${origin}/auth/register`)
  expect(response?.headers()['cache-control']).toContain('no-store')
  await page.setViewportSize({ width: 375, height: 812 })
  await expect(page.locator('.moe-auth-mobile-brand')).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
})
