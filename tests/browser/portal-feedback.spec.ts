import { expect, test } from '@playwright/test'
import { mockPortal } from './helpers/portal'
import { chooseTheme } from './helpers/theme'

const origin = 'http://127.0.0.1:53000'

test('system theme follows media changes while an explicit choice remains stable', async ({ page }) => {
  await mockPortal(page)
  await page.emulateMedia({ colorScheme: 'dark' })
  await page.goto(`${origin}/developer`)
  await chooseTheme(page, '切换主题', '跟随系统')
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
  await page.emulateMedia({ colorScheme: 'light' })
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'light')
  await chooseTheme(page, '切换主题', '深色')
  await page.emulateMedia({ colorScheme: 'light' })
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
  await page.reload()
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
})

test('logout errors retain the current page and never reuse identity CSRF', async ({ page }) => {
  await mockPortal(page)
  let writes = 0
  await page.route('**/api/v1/portal/logout', route => {
    writes++
    expect(route.request().headers()['x-moetopia-csrf']).toBe('portal-csrf')
    return route.fulfill({ status: 403, json: {} })
  })
  await page.goto(`${origin}/profile`)
  await page.getByRole('button', { name: '你的账号', exact: true }).click()
  await page.getByRole('menuitem', { name: '退出登录', exact: true }).click()
  await expect(page.getByRole('alert')).toContainText('请求验证已失效')
  await expect(page).toHaveURL(`${origin}/profile`)
  expect(writes).toBe(1)
})

test('a failed RP session list is not displayed as an empty successful response', async ({ page }) => {
  await mockPortal(page)
  await page.route('**/api/v1/portal/sessions', route => route.fulfill({ status: 503, json: {} }))
  await page.goto(`${origin}/sessions`)
  await expect(page.getByRole('alert')).toContainText('暂时无法连接账号服务')
  await expect(page.getByRole('heading', { name: '没有可显示的会话' })).toHaveCount(0)
})
