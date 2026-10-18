import { expect, test } from '@playwright/test'

test.beforeEach(async ({ page }) => {
  await page.goto('/iframe.html?id=overview-navigation--default&viewMode=story')
  await expect(page.getByRole('heading', { name: '导航组件工作台' })).toBeVisible()
})

test('desktop navigation, sidebar collapse and menu actions', async ({ page }) => {
  const main = page.getByRole('navigation', { name: '主导航', exact: true })
  await main.getByRole('link', { name: '账号中心' }).click()
  await expect(main.getByRole('link', { name: '账号中心' })).toHaveAttribute('aria-current', 'page')
  await expect(main.locator('.moe-nav-item').filter({ hasText: '社区' })).toHaveAttribute('aria-disabled', 'true')
  await expect(main.getByRole('link', { name: '社区' })).toHaveCount(0)
  const sidebar = page.getByRole('complementary')
  await sidebar.getByRole('button', { name: '收起侧栏' }).click()
  await expect(sidebar.getByRole('button', { name: '展开侧栏' })).toHaveAttribute('aria-expanded', 'false')
  await expect(sidebar.getByRole('link', { name: '安全与登录' })).toBeVisible()
  await sidebar.getByRole('link', { name: '安全与登录' }).click()
  await expect(sidebar.getByRole('link', { name: '安全与登录' })).toHaveAttribute('aria-current', 'page')
  const trigger = page.getByRole('button', { name: '打开用户菜单' })
  await trigger.focus()
  await page.keyboard.press('Enter')
  await expect(page.getByRole('menu', { name: '用户菜单' })).toBeVisible()
  await expect(page.getByRole('menuitem', { name: '管理控制台' })).toBeDisabled()
  await page.getByRole('menuitem', { name: '偏好设置' }).click()
  await expect(page.getByRole('status').first()).toContainText('偏好设置')
  await expect(trigger).toBeFocused()
})

test('mobile navigation traps focus, closes after selection and restores focus', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 })
  const trigger = page.getByRole('button', { name: '打开导航', exact: true })
  await trigger.click()
  const drawer = page.getByRole('dialog', { name: '主导航', exact: true })
  await expect(drawer).toBeVisible()
  await page.keyboard.press('Shift+Tab')
  expect(await drawer.evaluate(element => element.contains(document.activeElement))).toBe(true)
  await drawer.getByRole('link', { name: '开发者', exact: true }).click()
  await expect(drawer).toHaveCount(0)
  await expect(trigger).toBeFocused()
  await trigger.click()
  await page.keyboard.press('Escape')
  await expect(drawer).toHaveCount(0)
  await trigger.click()
  await page.setViewportSize({ width: 1200, height: 850 })
  await expect(drawer).toHaveCount(0)
  await expect(page.getByRole('navigation', { name: '主导航', exact: true })).toBeVisible()
})

test('pagination changes page and protects boundaries', async ({ page }) => {
  const pagination = page.getByRole('navigation', { name: '分页', exact: true })
  await expect(pagination.getByRole('button', { name: '上一页' })).toBeDisabled()
  await pagination.getByRole('button', { name: '下一页' }).click()
  await expect(page.getByRole('status').last()).toContainText('当前第 2 页')
  await pagination.getByRole('button', { name: '第 20 页', exact: true }).click()
  await expect(pagination.getByRole('button', { name: '下一页' })).toBeDisabled()
  await expect(pagination.getByRole('button', { name: '第 20 页', exact: true })).toHaveAttribute('aria-current', 'page')
  await pagination.getByRole('spinbutton', { name: '跳转到页数' }).fill('7')
  await pagination.getByRole('button', { name: '跳转', exact: true }).click()
  await expect(page.getByRole('status').last()).toContainText('当前第 7 页')
  await expect.poll(async () => {
    const [marker, target] = await Promise.all([pagination.locator('.moe-page-indicator').boundingBox(), pagination.getByRole('button', { name: '第 7 页', exact: true }).boundingBox()])
    return !!marker && !!target && Math.abs(marker.x - target.x) < 1 && Math.abs(marker.width - target.width) < 1
  }).toBe(true)
})

for (const theme of ['light', 'dark']) {
  test(`${theme} navigation fits 375px`, async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 })
    await page.goto(`/iframe.html?id=overview-navigation--default&viewMode=story&globals=theme:${theme}`)
    await expect(page.getByRole('heading', { name: '导航组件工作台' })).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  })
}
