import { expect, test } from '@playwright/test'
import { selectLanguage } from './helpers/language'
import { mockPortal, previewUser } from './helpers/portal'

const origin = 'http://127.0.0.1:53000'
const anonymous = { csrf_token: 'identity-csrf', registration_mode: 'invite' }

test('identity sign-in starts OAuth instead of granting a user-site session', async ({ page }) => {
  await page.route('**/api/v1/status', route => route.fulfill({ json: { service: 'moetopia' } }))
  await page.route('**/api/v1/account/context', route => route.fulfill({ json: anonymous }))
  await page.route('**/api/v1/account/login', route => {
    expect(route.request().headers()['x-moetopia-csrf']).toBe(anonymous.csrf_token)
    return route.fulfill({ json: { ...anonymous, user: previewUser } })
  })
  await page.route('**/api/v1/portal/login?**', route => route.fulfill({ body: '<h1>OAuth started</h1>', contentType: 'text/html' }))
  await page.goto(`${origin}/auth/login?continue=developer-applications`)
  await page.getByLabel('邮箱', { exact: true }).fill(previewUser.email)
  await page.getByLabel('密码', { exact: true }).fill('example passphrase only')
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await expect(page).toHaveURL(`${origin}/api/v1/portal/login?return_to=%2Fdeveloper%2Fapplications`)
  await expect(page.getByRole('heading')).toHaveText('OAuth started')
})

test('registration retains verification flow and supported email locale', async ({ page }) => {
  await page.route('**/api/v1/status', route => route.fulfill({ json: { service: 'moetopia' } }))
  await page.route('**/api/v1/account/context', route => route.fulfill({ json: anonymous }))
  await page.route('**/api/v1/account/verification', route => {
    expect(route.request().postDataJSON()).toEqual({ email: previewUser.email, locale: 'en' })
    return route.fulfill({ json: { challenge_id: 'example-challenge' } })
  })
  await page.route('**/api/v1/account/register', route => {
    expect(route.request().postDataJSON()).toMatchObject({ challenge_id: 'example-challenge', invitation: 'example-invitation', code: '12345678' })
    return route.fulfill({ status: 204 })
  })
  await page.goto(`${origin}/auth/register`)
  await selectLanguage(page, 'English')
  await page.getByLabel('Email', { exact: true }).fill(previewUser.email)
  await page.getByLabel('Display name', { exact: true }).fill('Preview user')
  await page.getByLabel('Invitation code', { exact: true }).fill('example-invitation')
  await page.getByLabel('Password', { exact: true }).fill('example passphrase only')
  await page.getByRole('button', { name: 'Send code', exact: true }).click()
  await page.getByLabel('Email verification code', { exact: true }).fill('12345678')
  await page.getByRole('button', { name: 'Verify and create account' }).click()
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Welcome back')
  await expect(page.getByRole('alert')).toContainText('Your account was created')
  await expect(page.getByLabel('Password', { exact: true })).toHaveValue('')
})

test('session revocation confirms once and uses portal CSRF, including current-session expiry', async ({ page }) => {
  const state = await mockPortal(page)
  await page.route('**/api/v1/portal/login?**', route => route.fulfill({ body: 'OAuth restart' }))
  await page.goto(`${origin}/sessions`)
  await expect(page.getByRole('button', { name: '结束会话', exact: true })).toHaveCount(2)
  await page.getByRole('button', { name: '结束会话', exact: true }).last().click()
  expect(state.writes).toEqual([])
  await page.getByRole('dialog').getByRole('button', { name: '结束会话', exact: true }).click()
  await expect(page.getByRole('button', { name: '结束会话', exact: true })).toHaveCount(1)
  expect(state.writes).toEqual(['portal-csrf'])
  await page.getByRole('button', { name: '结束会话', exact: true }).click()
  await page.getByRole('dialog').getByRole('button', { name: '结束会话', exact: true }).click()
  await expect(page).toHaveURL(`${origin}/api/v1/portal/login?return_to=%2Fsessions`)
  expect(state.writes).toEqual(['portal-csrf', 'portal-csrf'])
})

test('profile navigation reuses portal context and unavailable changes cannot submit', async ({ page }) => {
  const state = await mockPortal(page)
  await page.goto(`${origin}/profile`)
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('个人资料')
  await expect(page.getByRole('button', { name: '保存修改' })).toBeDisabled()
  await expect(page.getByRole('button', { name: '修改密码', exact: true })).toBeDisabled()
  await page.getByRole('link', { name: '管理登录会话', exact: true }).click()
  await expect(page).toHaveURL(`${origin}/sessions`)
  await page.getByRole('button', { name: '你的账号', exact: true }).click()
  await page.getByRole('menuitem', { name: '安全设置', exact: true }).click()
  await expect(page).toHaveURL(`${origin}/security`)
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('安全设置')
  expect(state.reads).toBe(1)
})

test('local logout uses the RP endpoint and explains the remaining identity session', async ({ page }) => {
  const state = await mockPortal(page)
  let identityWrites = 0
  await page.route('**/api/v1/account/logout', route => { identityWrites++; return route.fulfill({ status: 500 }) })
  await page.goto(`${origin}/profile`)
  await page.getByRole('button', { name: '你的账号', exact: true }).click()
  await page.getByRole('menuitem', { name: '退出登录', exact: true }).click()
  await expect(page).toHaveURL(`${origin}/auth/logout`)
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('已退出用户站')
  await expect(page.getByText(/中央身份会话仍保留/)).toBeVisible()
  expect(state.writes).toEqual(['portal-csrf'])
  expect(identityWrites).toBe(0)
})

test('administration drawer remains usable without claiming a user RP is an administrator', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 })
  await page.route('**/api/**', route => route.fulfill({ status: 503, json: {} }))
  await page.goto('http://127.0.0.1:53001')
  await page.getByRole('button', { name: '打开导航' }).click()
  const drawer = page.getByRole('dialog', { name: '主导航' })
  await expect(drawer).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(drawer).toHaveCount(0)
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
})
