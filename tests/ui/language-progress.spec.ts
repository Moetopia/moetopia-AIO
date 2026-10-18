import { expect, test } from '@playwright/test'

test('language picker exposes selected options and returns keyboard focus', async ({ page }) => {
  await page.goto('/iframe.html?id=navigation-languagepicker--default&viewMode=story')
  const trigger = page.getByRole('combobox', { name: '语言 / Language', exact: true })
  await trigger.click()
  await expect(page.getByRole('option', { name: '简体中文', exact: true })).toHaveAttribute('aria-selected', 'true')
  await page.getByRole('option', { name: '日本語', exact: true }).click()
  await expect(trigger).toContainText('日本語')
  await expect(trigger).toBeFocused()
  await page.keyboard.press('Enter')
  await page.keyboard.press('Escape')
  await expect(trigger).toBeFocused()
})

test('navigation progress is controlled and does not announce a fabricated percentage', async ({ page }) => {
  await page.goto('/iframe.html?id=feedback-navigationprogress--controlled&viewMode=story')
  await expect(page.getByRole('progressbar')).toHaveCount(0)
  await page.getByRole('button', { name: '开始加载', exact: true }).click()
  await expect(page.getByRole('progressbar')).toBeVisible()
  await expect(page.getByRole('progressbar')).not.toHaveAttribute('aria-valuenow')
  await page.getByRole('button', { name: '完成', exact: true }).click()
  await expect(page.getByRole('progressbar')).toHaveCount(0)
})

test('landing and centered account summary remain readable at mobile width', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 })
  for (const id of ['layouts-landinglayout--default', 'layouts-profilesummary--default']) {
    await page.goto(`/iframe.html?id=${id}&viewMode=story`)
    await expect(page.getByRole('heading').first()).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  }
})
