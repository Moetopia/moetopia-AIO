import { expect, test } from '@playwright/test'
const origin = 'http://127.0.0.1:56006/iframe.html'

test('copy feedback distinguishes clipboard success from failure', async ({ page }) => {
  await page.addInitScript(() => Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText: async () => {} } }))
  await page.goto(`${origin}?id=actions-copybutton--default&viewMode=story`)
  await page.getByRole('button', { name: '复制', exact: true }).click()
  await expect(page.getByRole('button', { name: '已复制', exact: true })).toBeVisible()
  await page.evaluate(() => Object.defineProperty(navigator, 'clipboard', { configurable: true, value: { writeText: async () => { throw new Error('denied') } } }))
  await page.getByRole('button', { name: '已复制', exact: true }).click()
  await expect(page.getByRole('status')).toHaveText('复制失败')
  await expect(page.getByRole('status')).toBeVisible()
})

test('theme menu supports keyboard selection and restores trigger focus', async ({ page }) => {
  await page.goto(`${origin}?id=navigation-themepicker--default&viewMode=story`)
  const trigger = page.getByRole('button', { name: '切换主题' })
  await trigger.click()
  await page.getByRole('menuitem', { name: '跟随系统' }).focus()
  await page.keyboard.press('Enter')
  await expect(page.getByRole('status')).toHaveText('system')
  await expect(trigger).toBeFocused()
})
