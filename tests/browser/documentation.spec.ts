import { expect, test } from '@playwright/test'
import schema from '../../packages/api-client/openapi.json'

const origin = 'http://127.0.0.1:53000'
const expected = Object.entries(schema.paths).flatMap(([path, item]) => Object.entries(item).filter(([method]) => ['get', 'post', 'put', 'patch', 'delete', 'head', 'options'].includes(method)).map(([method, value]) => ({ id: (value as { operationId: string }).operationId, method: method.toUpperCase(), path })))

test('documentation catalog and every deep link use the generated contract without a running API', async ({ request }) => {
  const catalog = await request.get(origin + '/developer/docs/catalog')
  expect(catalog.status()).toBe(200)
  const items = await catalog.json() as { id: string; method: string; path: string }[]
  expect(items.map(({ id, method, path }) => ({ id, method, path }))).toEqual(expected)
  for (let start = 0; start < items.length; start += 8) {
    await Promise.all(items.slice(start, start + 8).map(async item => {
      const response = await request.get(origin + `/developer/docs/operations/${item.id}`)
      expect(response.status(), item.id).toBe(200)
      expect((await response.json()).entry).toMatchObject(item)
      const rendered = await request.get(origin + `/developer/docs/api/${item.id}`)
      expect(rendered.status(), `${item.id} renders as a page`).toBe(200)
    }))
  }
  const download = await request.get(origin + '/developer/docs/openapi.json')
  expect(await download.json()).toEqual(schema)
  expect((await request.get(origin + '/developer/docs/operations/unknown-operation')).status()).toBe(404)
  expect((await request.get(origin + '/developer/docs/api/unknown-operation')).status()).toBe(404)
})

test('API search, filter, direct navigation and response schemas work anonymously', async ({ page }) => {
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  await page.route('**/api/v1/**', route => route.fulfill({ status: 503, json: {} }))
  await page.goto(origin + '/developer/docs/endpoints')
  await page.getByRole('searchbox', { name: '搜索接口', exact: true }).fill('oauth-token')
  await expect(page.locator('.moe-endpoint-link')).toHaveCount(1)
  await expect(page).toHaveURL(/q=oauth-token/)
  await page.locator('.moe-endpoint-link').click()
  await expect(page).toHaveURL(origin + '/developer/docs/api/oauth-token')
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('兑换、刷新或签发令牌')
  await expect(page.locator('#body')).toContainText('grant_type')
  await expect(page.locator('#body')).toContainText('application/x-www-form-urlencoded')
  await expect(page.locator('#body')).toContainText('oneOf 1')
  await page.getByRole('combobox', { name: '状态码', exact: true }).click()
  await page.getByRole('option', { name: '400', exact: true }).click()
  await expect(page.locator('#responses')).toContainText('error_description')
  await page.reload()
  await expect(page.locator('#body')).toContainText('code_verifier')
  await page.getByRole('link', { name: '返回 API 列表' }).click()
  await page.getByRole('combobox', { name: 'HTTP 方法', exact: true }).click()
  await page.getByRole('option', { name: 'DELETE', exact: true }).click()
  await expect(page.locator('.moe-endpoint-link')).toHaveCount(expected.filter(item => item.method === 'DELETE').length)
  await page.getByRole('searchbox', { name: '搜索接口', exact: true }).fill('no-such-endpoint')
  await expect(page.getByText('没有找到匹配的接口，请调整筛选条件。')).toBeVisible()
  await page.goto(origin + '/developer/docs/api/oauth-revoke')
  await expect(page.locator('#responses')).toContainText('无响应体')
  expect(errors).toEqual([])
})

test('reference distinguishes administrative Cookie workflows from OAuth and expands referenced schemas', async ({ page }) => {
  await page.route('**/api/v1/**', route => route.fulfill({ status: 503, json: {} }))
  await page.goto(origin + '/developer/docs/api/get-api-v1-admin-users')
  await expect(page.locator('#authentication')).toContainText('管理员业务另需角色与有效 MFA')
  await expect(page.locator('#responses')).toContainText('items')
  await page.locator('#responses details > summary').first().click()
  await expect(page.locator('#responses')).toContainText('[]')
})

test('guide navigation, keyboard search and four-language mobile rendering remain usable', async ({ page }) => {
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  await page.route('**/api/v1/**', route => route.fulfill({ status: 503, json: {} }))
  for (const [locale, title] of [['zh-CN', '令牌、刷新与撤销'], ['zh-TW', '權杖、刷新與撤銷'], ['en', 'Tokens, refresh and revocation'], ['ja', 'トークン、刷新と失効']]) {
    await page.context().addCookies([{ name: 'moetopia_locale', value: locale!, url: origin }])
    await page.setViewportSize({ width: 375, height: 900 })
    await page.goto(origin + '/developer/docs/tokens')
    await expect(page.getByRole('heading', { level: 1 })).toHaveText(title!)
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
    await page.screenshot({ path: `.data/previews/documentation/${locale}-mobile.png`, fullPage: true })
  }
  expect(errors).toEqual([])
})

test('API reference stays readable across four languages, themes and viewport sizes', async ({ page }) => {
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  await page.route('**/api/v1/**', route => route.fulfill({ status: 503, json: {} }))
  for (const locale of ['zh-CN', 'zh-TW', 'en', 'ja']) {
    for (const [width, theme] of [[1440, 'light'], [375, 'dark']] as const) {
      await page.context().addCookies([{ name: 'moetopia_locale', value: locale, url: origin }, { name: 'moetopia_theme', value: theme, url: origin }])
      await page.setViewportSize({ width, height: 1000 })
      await page.goto(origin + (width === 1440 ? '/developer/docs/endpoints?group=OAuth%20%2F%20OIDC' : '/developer/docs/api/oauth-token'))
      await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
      await page.screenshot({ path: `.data/previews/documentation/${locale}-${width}-${theme}.png`, fullPage: true })
    }
  }
  expect(errors).toEqual([])
})

test('catalog failure offers a real retry instead of an empty successful directory', async ({ page }) => {
  await page.route('**/api/v1/**', route => route.fulfill({ status: 503, json: {} }))
  let unavailable = true
  await page.route('**/developer/docs/catalog', route => unavailable ? route.fulfill({ status: 503, json: {} }) : route.continue())
  await page.goto(origin + '/developer/docs')
  await page.locator('.moe-docs-sidebar a[href="/developer/docs/endpoints"]').click()
  await expect(page.getByRole('alert')).toContainText('文档暂时无法加载')
  await expect(page.getByText('没有找到匹配的接口，请调整筛选条件。')).toHaveCount(0)
  unavailable = false
  await page.getByRole('button', { name: '重新加载', exact: true }).click()
  await expect(page.locator('.moe-endpoint-link')).toHaveCount(expected.length)
  await expect(page.getByRole('alert')).toHaveCount(0)
})
