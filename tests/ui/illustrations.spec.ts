import { expect, test } from '@playwright/test'

test('all illustration variants load without exposing decorative image labels', async ({ page }) => {
  await page.goto('/iframe.html?id=brand-illustration--gallery&viewMode=story')
  const images = page.locator('.moe-illustration img')
  await expect(images).toHaveCount(12)
  for (const image of await images.all()) {
    await image.scrollIntoViewIfNeeded()
    await expect.poll(() => image.evaluate((node: HTMLImageElement) => node.complete && node.naturalWidth > 1)).toBe(true)
    await expect(image).toHaveAttribute('alt', '')
  }
})

for (const [variant, code] of [['not-found', '404'], ['restricted', '403'], ['unavailable', '503']] as const) {
  test(`status page ${code} keeps content and actions on a narrow screen`, async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 })
    await page.emulateMedia({ reducedMotion: 'reduce' })
    await page.goto(`/iframe.html?id=feedback-statuspage--${variant}&viewMode=story`)
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    await expect(page.getByRole('button', { name: '返回首页' })).toBeVisible()
    await expect(page.locator('.moe-status-code')).toHaveText(code)
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  })
}
