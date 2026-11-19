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

test('system dark theme applies before first paint and hydration without flashing light', async ({ page }) => {
  let releaseContext!: () => void
  const gate = new Promise<void>(resolve => { releaseContext = resolve })
  await page.emulateMedia({ colorScheme: 'dark' })
  await page.context().addCookies([{ name: 'moetopia_theme', value: 'system', url: origin }])
  await page.route('**/api/v1/portal/context', async route => {
    await gate
    return route.fulfill({ json: { authenticated: true, csrf_token: 'portal-csrf', user: { id: 'preview', display_name: '预览用户', email: 'preview@example.com' } } })
  })
  await page.route('**/api/v1/status', route => route.fulfill({ json: { service: 'moetopia' } }))
  await page.addInitScript(() => {
    const seen: string[] = []
    const record = () => {
      const value = document.documentElement?.getAttribute('data-theme')
      if (value) seen.push(value)
    }
    const attach = () => {
      if (!document.documentElement) return
      record()
      new MutationObserver(record).observe(document.documentElement, { attributes: true, attributeFilter: ['data-theme'] })
    }
    document.addEventListener('readystatechange', attach)
    document.addEventListener('DOMContentLoaded', record)
    ;(window as unknown as { __themes?: string[] }).__themes = seen
  })
  try {
    await page.goto(`${origin}/profile`, { waitUntil: 'domcontentloaded' })
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')
    const beforeApi = await page.evaluate(() => (window as unknown as { __themes?: string[] }).__themes ?? [])
    expect(beforeApi.length).toBeGreaterThan(0)
    expect(beforeApi.every(value => value === 'dark')).toBe(true)
  } finally { releaseContext() }
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
