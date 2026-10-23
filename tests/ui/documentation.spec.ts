import { expect, test } from '@playwright/test'

test('endpoint links remain keyboard accessible and fit long paths on mobile', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 800 })
  await page.goto('/iframe.html?id=documentation-endpointlist--default&viewMode=story')
  await expect(page.getByRole('list', { name: 'OAuth / OIDC' })).toBeVisible()
  const link = page.getByRole('link').first()
  await link.focus()
  await page.keyboard.press('Enter')
  await expect(page).toHaveURL(/#token$/)
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
})

test('schema children expand with the keyboard without losing required markers', async ({ page }) => {
  await page.goto('/iframe.html?id=documentation-schemafields--nested&viewMode=story')
  const summary = page.locator('summary').first()
  await expect(page.getByText('redirect_uris', { exact: true })).not.toBeVisible()
  await summary.focus()
  await page.keyboard.press('Enter')
  await expect(page.getByText('redirect_uris', { exact: true })).toBeVisible()
  await expect(page.locator('.moe-schema-field').filter({ has: page.locator('dt').filter({ hasText: /^grant_type/ }) }).first()).toContainText('必填')
})

test('mobile document outline expands and links to the requested section', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 800 })
  await page.goto('/iframe.html?id=documentation-documentlayout--default&viewMode=story')
  await page.locator('.moe-document-mobile-outline > summary').click()
  await page.getByRole('navigation', { name: '本页内容' }).getByRole('link', { name: '验证回调' }).click()
  await expect(page).toHaveURL(/#callback$/)
})

test('folder navigation supports disclosure keys and selected-operation links', async ({ page }) => {
  await page.goto('/iframe.html?id=documentation-navigationtree--default&viewMode=story')
  const folder = page.getByRole('button', { name: /OAuth \/ OIDC/ })
  await expect(folder).toHaveAttribute('aria-expanded', 'true')
  await folder.focus(); await page.keyboard.press('ArrowLeft')
  await expect(page.getByRole('link', { name: /兑换令牌/ })).not.toBeVisible()
  await page.keyboard.press('ArrowRight')
  await expect(page.getByRole('link', { name: /兑换令牌/ })).toHaveAttribute('aria-current', 'page')
})
