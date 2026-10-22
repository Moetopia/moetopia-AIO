import { selectLanguage } from './helpers/language'
import { chooseTheme } from './helpers/theme'
import { expect, test } from '@playwright/test'

for (const [port, path, title, heading] of [[53000, '/auth/login', '账号中心', 'Welcome back'], [53001, '/', '管理控制台', 'Dashboard'], [53000, '/developer', '开发者门户', 'Connect your application to Moetopia.']] as const) {
  test(`${title} renders safely without a backend`, async ({ page }) => {
    const exceptions: string[] = []
    page.on('pageerror', (error) => exceptions.push(error.message))
    await page.route('**/api/**', route => route.fulfill({ status: 503, contentType: 'application/json', body: '{}' }))
    await page.goto(`http://127.0.0.1:${port}${path}`)
    await expect(page).toHaveTitle(`${title} · Moetopia`)
    if (port !== 53001) await expect(page.locator('.moe-status-indicator[role="status"]')).toHaveText('连接暂不可用')
    await chooseTheme(page, '切换主题', '深色')
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
    await selectLanguage(page, 'English')
    await expect(page.locator('html')).toHaveAttribute('lang', 'en')
    if (port === 53001) await expect(page.getByText('Request failed. Please retry.')).toBeVisible()
    else await expect(page.getByRole('heading', { level: 1 })).toHaveText(heading)
    await page.reload()
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
    await expect(page.locator('html')).toHaveAttribute('lang', 'en')
    expect(exceptions).toEqual([])
  })
}

test('account layout fits a narrow viewport', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 })
  await page.route('**/api/**', route => route.fulfill({ status: 503, contentType: 'application/json', body: '{}' }))
  await page.goto('http://127.0.0.1:53000/auth/login')
  await expect(page.locator('main img')).toHaveCount(0)
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
})

test('registration fits mobile English and dark mode', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 })
  await page.route('**/api/**', route => route.fulfill({ status: 503, contentType: 'application/json', body: '{}' }))
  await page.goto('http://127.0.0.1:53000/auth/login')
  await expect(page.getByRole('status')).toHaveText('连接暂不可用')
  await page.getByRole('link', { name: '立即注册', exact: true }).click()
  await selectLanguage(page, 'English')
  await chooseTheme(page, 'Change theme', 'Dark')
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('Create account')
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
})

test('password trailing control works and policy is described', async ({ page }) => {
  await page.route('**/api/**', route => route.fulfill({ status: 503, contentType: 'application/json', body: '{}' }))
  await page.goto('http://127.0.0.1:53000/auth/login')
  await expect(page.getByRole('status')).toHaveText('连接暂不可用')
  const password = page.getByLabel('密码', { exact: true })
  await password.fill('example passphrase only')
  await page.getByRole('button', { name: '显示密码', exact: true }).click()
  await expect(password).toHaveAttribute('type', 'text')
  await page.getByRole('button', { name: '隐藏密码', exact: true }).click()
  await expect(password).toHaveAttribute('type', 'password')
  await page.getByRole('link', { name: '立即注册', exact: true }).click()
  await expect(password).toHaveAccessibleDescription('至少 15 个字符，可以使用空格。')
})

test('dialog supports Escape and restores focus', async ({ page }) => {
  await page.goto('http://127.0.0.1:56006/iframe.html?id=foundation-dialog--default&viewMode=story')
  const trigger = page.getByRole('button', { name: '打开 · Open' })
  await trigger.click()
  await expect(page.getByRole('dialog')).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).not.toBeVisible()
  await expect(trigger).toBeFocused()
})

test('loading controls cannot submit and errors are associated with inputs', async ({ page }) => {
  await page.goto('http://127.0.0.1:56006/iframe.html?id=foundation-button--loading&viewMode=story')
  await expect(page.getByRole('button', { name: '继续 · Continue' })).toBeDisabled()
  await page.goto('http://127.0.0.1:56006/iframe.html?id=foundation-input--error&viewMode=story')
  await expect(page.getByRole('textbox', { name: '邮箱 · Email' })).toHaveAccessibleDescription('请输入有效邮箱。Enter a valid email address.')
})
