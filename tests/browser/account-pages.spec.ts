import { expect, test, type Page } from '@playwright/test'

const anonymous = { csrf_token: 'example-csrf-token', registration_mode: 'invite' }
const user = { id: 'example-user', display_name: '预览用户', email: 'preview@example.com' }
const session = { id: 'current', created_at: '2026-10-18T02:00:00Z', expires_at: '2026-11-18T02:00:00Z' }
const mockStatus = async (page: Page) => page.route('**/api/v1/status', route => route.fulfill({ json: { service: 'moetopia' } }))

test('sign-in changes layout without remounting account state, sign-out returns to auth', async ({ page }) => {
  await mockStatus(page)
  let contextRequests = 0
  await page.route('**/api/v1/account/context', route => { contextRequests++; return route.fulfill({ json: anonymous }) })
  await page.route('**/api/v1/account/sessions', route => route.fulfill({ json: { current_id: 'current', items: [session] } }))
  await page.route('**/api/v1/account/login', async route => {
    expect(route.request().headers()['x-moetopia-csrf']).toBe(anonymous.csrf_token)
    await route.fulfill({ json: { ...anonymous, user } })
  })
  await page.route('**/api/v1/account/logout', route => route.fulfill({ json: anonymous }))
  await page.goto('http://127.0.0.1:53000')
  await page.getByLabel('邮箱', { exact: true }).fill(user.email)
  await page.getByLabel('密码', { exact: true }).fill('example-password-only')
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await expect(page.getByRole('heading', { level: 1 })).toHaveText(`欢迎回来，${user.display_name}`)
  await expect(page.getByRole('heading', { name: '基本资料' })).toBeVisible()
  await expect(page.getByText(user.email, { exact: true }).first()).toBeVisible()
  await page.getByRole('navigation', { name: '主导航' }).getByRole('link', { name: '登录会话', exact: true }).click()
  await expect(page).toHaveURL('http://127.0.0.1:53000/sessions')
  await expect(page.getByRole('button', { name: '结束会话' })).toHaveCount(1)
  expect(contextRequests).toBe(1)
  await page.getByRole('button', { name: '退出登录', exact: true }).click()
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('欢迎回来')
  await expect(page.getByLabel('密码', { exact: true })).toHaveValue('')
  await expect(page.getByRole('complementary').filter({ has: page.getByRole('navigation', { name: '工作区' }) })).toHaveCount(0)
})

test('registration retains verification flow and uses the selected locale', async ({ page }) => {
  await mockStatus(page)
  await page.route('**/api/v1/account/context', route => route.fulfill({ json: anonymous }))
  await page.route('**/api/v1/account/verification', async route => {
    expect(route.request().postDataJSON()).toEqual({ email: user.email, locale: 'en' })
    await route.fulfill({ json: { challenge_id: 'example-challenge' } })
  })
  await page.route('**/api/v1/account/register', async route => {
    const body = route.request().postDataJSON()
    expect(body.challenge_id).toBe('example-challenge')
    expect(body.invitation).toBe('example-invitation')
    expect(body.code).toBe('12345678')
    await route.fulfill({ status: 204 })
  })
  await page.goto('http://127.0.0.1:53000')
  await page.getByRole('button', { name: 'Switch to English' }).click()
  await page.getByRole('tab', { name: 'Create account', exact: true }).click()
  await page.getByLabel('Email', { exact: true }).fill(user.email)
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

test('session actions refresh real response data and ending current session returns to auth', async ({ page }) => {
  await mockStatus(page)
  let signedIn = true
  let items = [session, { ...session, id: 'other' }]
  await page.route('**/api/v1/account/context', route => route.fulfill({ json: signedIn ? { ...anonymous, user } : anonymous }))
  await page.route('**/api/v1/account/sessions', route => route.fulfill({ json: { current_id: 'current', items } }))
  await page.route('**/api/v1/account/sessions/*', async route => {
    const id = route.request().url().split('/').at(-1)
    items = items.filter(item => item.id !== id)
    if (id === 'current') signedIn = false
    await route.fulfill({ status: 204 })
  })
  await page.goto('http://127.0.0.1:53000/sessions')
  await expect(page.getByRole('button', { name: '结束会话' })).toHaveCount(2)
  await page.getByRole('button', { name: '结束会话' }).last().click()
  await expect(page.getByRole('button', { name: '结束会话' })).toHaveCount(1)
  await page.getByRole('button', { name: '结束会话' }).click()
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('欢迎回来')
})

test('security route uses shared account state and links to usable session management', async ({ page }) => {
  await mockStatus(page)
  let contextRequests = 0
  await page.route('**/api/v1/account/context', route => { contextRequests++; return route.fulfill({ json: { ...anonymous, user } }) })
  await page.route('**/api/v1/account/sessions', route => route.fulfill({ json: { current_id: 'current', items: [session] } }))
  await page.goto('http://127.0.0.1:53000/security')
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('安全设置')
  await expect(page.getByText('修改密码待开放', { exact: true })).toBeVisible()
  await expect(page.locator('main img')).toHaveCount(0)
  await page.getByRole('link', { name: '管理登录会话', exact: true }).click()
  await expect(page).toHaveURL('http://127.0.0.1:53000/sessions')
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('登录会话')
  await expect(page.getByRole('button', { name: '结束会话' })).toHaveCount(1)
  await page.getByRole('navigation', { name: '主导航' }).getByRole('link', { name: '个人资料' }).click()
  await expect(page).toHaveURL('http://127.0.0.1:53000/profile')
  await expect(page.getByRole('heading', { name: '基本资料' })).toBeVisible()
  expect(contextRequests).toBe(1)
})

for (const port of [53001]) {
  test(`workspace on ${port} fits mobile and exposes honest navigation`, async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 })
    await mockStatus(page)
    await page.goto(`http://127.0.0.1:${port}`)
    await page.getByRole('button', { name: '打开导航' }).click()
    const menu = page.getByRole('dialog', { name: '主导航' })
    await expect(menu).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(menu).toHaveCount(0)
    await expect(page.getByText('功能准备中', { exact: true }).first()).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  })
}
