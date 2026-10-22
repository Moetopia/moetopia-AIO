import { expect, test } from '@playwright/test'
import { mockPortal } from './helpers/portal'

const origin = 'http://127.0.0.1:53000'
test('new passwords count Unicode characters and UTF-8 bytes while strength remains advisory', async ({ page }) => {
  await page.route('**/api/**', route => route.fulfill({ json: { csrf_token: 'example-csrf', registration_mode: 'open', service: 'moetopia', items: [] } }))
  await page.goto(origin + '/auth/register')
  const password = page.getByLabel('密码', { exact: true })
  const feedback = page.getByRole('region', { name: '密码要求' })
  await password.fill('😀'.repeat(9))
  await expect(feedback).toContainText('9/15')
  await expect(feedback).toContainText('36/256')
  expect(await password.evaluate(input => (input as HTMLInputElement).checkValidity())).toBe(false)
  await password.fill('界'.repeat(90))
  await expect(feedback).toContainText('270/256')
  expect(await password.evaluate(input => (input as HTMLInputElement).checkValidity())).toBe(false)
  await password.fill('a'.repeat(15))
  await expect(feedback.getByRole('status')).toContainText('很弱')
  expect(await password.evaluate(input => (input as HTMLInputElement).checkValidity())).toBe(true)
  await password.fill('lantern meadow orbit walnut river')
  await expect(feedback.getByRole('meter')).toHaveAttribute('value', '4')
  await expect(feedback).toContainText('不强制大小写、数字或符号')
})

test('password confirmation mismatch updates while typing and does not send a request', async ({ page }) => {
  await mockPortal(page)
  let writes = 0
  await page.route('**/api/v1/portal/password', route => { writes++; return route.fulfill({ status: 204 }) })
  await page.goto(origin + '/security')
  const form = page.locator('section').filter({ has: page.getByRole('heading', { name: '修改密码', exact: true }) }).locator('form')
  await form.getByLabel('新密码', { exact: true }).fill('lantern meadow orbit walnut river')
  await form.getByLabel('确认新密码', { exact: true }).fill('different')
  await expect(form.getByRole('alert')).toContainText('不一致')
  await form.getByLabel('确认新密码', { exact: true }).fill('lantern meadow orbit walnut river')
  await expect(form.getByRole('alert')).toHaveCount(0)
  expect(writes).toBe(0)
})

for (const [locale, title] of [['zh-CN', '密码要求'], ['zh-TW', '密碼要求'], ['en', 'Password requirements'], ['ja', 'パスワードの要件']]) {
  test(`password guidance is translated and fits a phone in ${locale}`, async ({ page }) => {
    await page.context().addCookies([{ name: 'moetopia_locale', value: locale!, url: origin }])
    await page.setViewportSize({ width: 375, height: 812 })
    await page.route('**/api/**', route => route.fulfill({ json: { csrf_token: 'example-csrf', registration_mode: 'open', service: 'moetopia', items: [] } }))
    await page.goto(origin + '/auth/register')
    await expect(page.getByRole('region', { name: title! })).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
    await page.screenshot({ path: `.data/previews/security/password-${locale}.png`, fullPage: true })
  })
}
