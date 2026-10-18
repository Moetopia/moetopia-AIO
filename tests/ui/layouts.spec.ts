import { expect, test } from '@playwright/test'

test('page frame provides a keyboard skip link to the main landmark', async ({ page }) => {
  await page.goto('http://127.0.0.1:56006/iframe.html?id=layouts-pageframe--default&viewMode=story')
  await expect(page.getByRole('main')).toBeVisible()
  await page.keyboard.press('Tab')
  await expect(page.getByRole('link', { name: '跳转到主要内容' })).toBeFocused()
  await page.keyboard.press('Enter')
  await expect(page.getByRole('main')).toBeFocused()
  await expect(page.getByRole('banner')).toBeVisible()
})

test('authentication layout keeps a usable form without candidate artwork', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 })
  await page.goto('http://127.0.0.1:56006/iframe.html?id=layouts-authlayout--default&viewMode=story')
  await expect(page.locator('.moe-auth-art')).toHaveCount(0)
  await page.setViewportSize({ width: 375, height: 812 })
  await expect(page.locator('.moe-auth-art')).toHaveCount(0)
  await page.getByRole('textbox', { name: '邮箱', exact: true }).fill('demo@example.com')
  await expect(page.getByRole('textbox', { name: '邮箱', exact: true })).toHaveValue('demo@example.com')
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
})
