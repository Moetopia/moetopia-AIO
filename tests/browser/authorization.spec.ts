import { selectLanguage } from './helpers/language'
import { chooseTheme } from './helpers/theme'
import { expect, test, type Page } from '@playwright/test'

const id = 'a'.repeat(43)
const origin = 'http://127.0.0.1:53000'
const url = `${origin}/?authorization_request=${id}`
const user = { id: 'preview-user', display_name: '预览用户', email: 'preview@example.com' }
const context = { csrf_token: 'example-csrf', registration_mode: 'invite', user }
const details = { client_id: 'preview-client', client_name: '同人图书馆', redirect_uri: 'http://localhost:53110/callback', scopes: ['openid', 'profile', 'email', 'library.read'], audiences: ['https://example.test/api'], expires_at: '2050-01-01T00:00:00Z', login_required: false, user }
const setup = async (page: Page) => {
  await page.route('**/api/v1/status', route => route.fulfill({ json: { service: 'moetopia' } }))
  await page.route('**/api/v1/account/context', route => route.fulfill({ json: context }))
  await page.route('**/api/v1/account/sessions', route => route.fulfill({ json: { current_id: 'preview', items: [] } }))
}

test('authorization displays real details, waits for consent and returns with full navigation', async ({ page }) => {
  await setup(page)
  let decisions = 0
  await page.route(`**/api/v1/account/authorizations/${id}`, async route => {
    if (route.request().method() === 'GET') return route.fulfill({ json: details })
    decisions++
    expect(route.request().postDataJSON()).toEqual({ decision: 'approve', subject_id: user.id })
    expect(route.request().headers()['x-moetopia-csrf']).toBe(context.csrf_token)
    return route.fulfill({ json: { redirect_to: 'http://localhost:53110/callback?code=example&state=preserved' } })
  })
  await page.route('http://localhost:53110/callback?**', route => route.fulfill({ contentType: 'text/html', body: '<h1>RP callback</h1>' }))
  const response = await page.goto(url)
  expect(response?.headers()['cache-control']).toContain('no-store')
  expect(response?.headers()['referrer-policy']).toBe('no-referrer')
  expect(response?.headers()['content-security-policy']).toContain("frame-ancestors 'none'")
  await expect(page.getByRole('heading', { name: details.client_name })).toBeVisible()
  await expect(page.getByText(details.redirect_uri, { exact: true })).toBeVisible()
  await expect(page.getByText('library.read', { exact: true })).toBeVisible()
  await expect(page.getByText(user.email, { exact: true })).toBeVisible()
  expect(decisions).toBe(0)
  await page.getByRole('button', { name: '允许访问', exact: true }).click()
  await expect(page).toHaveURL('http://localhost:53110/callback?code=example&state=preserved')
  expect(decisions).toBe(1)
})

test('deny sends an explicit decision and uses the server callback unchanged', async ({ page }) => {
  await setup(page)
  await page.route(`**/api/v1/account/authorizations/${id}`, route => {
    if (route.request().method() === 'GET') return route.fulfill({ json: details })
    expect(route.request().postDataJSON()).toEqual({ decision: 'deny', subject_id: user.id })
    return route.fulfill({ json: { redirect_to: 'http://localhost:53110/callback?error=access_denied&state=preserved' } })
  })
  await page.route('http://localhost:53110/callback?**', route => route.fulfill({ body: 'Denied' }))
  await page.goto(url)
  await page.getByRole('button', { name: '拒绝', exact: true }).click()
  await expect(page).toHaveURL('http://localhost:53110/callback?error=access_denied&state=preserved')
})

test('prompt login keeps the request and reloads details after genuine sign-in with fresh CSRF', async ({ page }) => {
  await setup(page)
  let signedIn = false
  let reads = 0
  let writes = 0
  await page.route('**/api/v1/account/login', route => { signedIn = true; return route.fulfill({ json: { ...context, csrf_token: 'fresh-csrf' } }) })
  await page.route(`**/api/v1/account/authorizations/${id}`, route => {
    if (route.request().method() === 'GET') { reads++; return route.fulfill({ json: { ...details, login_required: !signedIn } }) }
    writes++
    expect(route.request().headers()['x-moetopia-csrf']).toBe('fresh-csrf')
    return route.fulfill({ status: 503, json: { code: 'unavailable' } })
  })
  await page.goto(url)
  await expect(page.getByText('请重新验证账号', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: '允许访问', exact: true })).toHaveCount(0)
  await page.getByLabel('邮箱', { exact: true }).fill(user.email)
  await page.getByLabel('密码', { exact: true }).fill('example-password')
  await page.getByRole('button', { name: '登录', exact: true }).click()
  await expect(page.getByRole('button', { name: '允许访问', exact: true })).toBeEnabled()
  expect(reads).toBe(2)
  expect(writes).toBe(0)
  await expect(page).toHaveURL(url)
  await page.getByRole('button', { name: '允许访问', exact: true }).click()
  await expect(page.getByRole('alert')).toContainText('授权服务暂不可用')
  expect(writes).toBe(1)
})

test('account change reloads details and requires another explicit review', async ({ page }) => {
  await setup(page)
  let changed = false
  let writes = 0
  await page.route(`**/api/v1/account/authorizations/${id}`, route => {
    if (route.request().method() === 'GET') return route.fulfill({ json: { ...details, user: changed ? { ...user, id: 'other', display_name: '另一个账号' } : user } })
    writes++
    changed = true
    return route.fulfill({ status: 409, json: { code: 'authorization_account_changed' } })
  })
  await page.goto(url)
  await page.getByRole('button', { name: '允许访问', exact: true }).click()
  await expect(page.getByRole('alert')).toContainText('账号已变化')
  await expect(page.getByRole('button', { name: '允许访问', exact: true })).toBeDisabled()
  await expect(page.getByText('另一个账号', { exact: true }).first()).toBeVisible()
  expect(writes).toBe(1)
  await page.getByRole('button', { name: '确认应用访问', exact: true }).click()
  await expect(page.getByRole('button', { name: '允许访问', exact: true })).toBeEnabled()
  expect(writes).toBe(1)
})

test('switching account preserves authorization while requiring sign-in again', async ({ page }) => {
  await setup(page)
  let signedOut = false
  await page.route('**/api/v1/account/logout', route => { signedOut = true; return route.fulfill({ json: { csrf_token: 'anonymous-csrf', registration_mode: 'invite' } }) })
  await page.route(`**/api/v1/account/authorizations/${id}`, route => route.fulfill({ json: signedOut ? { ...details, user: undefined, login_required: true } : details }))
  await page.goto(url)
  await page.getByRole('button', { name: '换个账号', exact: true }).click()
  await expect(page.getByLabel('密码', { exact: true })).toBeVisible()
  await expect(page).toHaveURL(url)
  await expect(page.getByRole('button', { name: '允许访问', exact: true })).toHaveCount(0)
})

test('401 during confirmation requires sign-in instead of resubmitting', async ({ page }) => {
  await setup(page)
  let writes = 0
  await page.route(`**/api/v1/account/authorizations/${id}`, route => {
    if (route.request().method() === 'GET') return route.fulfill({ json: details })
    writes++
    return route.fulfill({ status: 401, json: { code: 'authorization_login_required' } })
  })
  await page.goto(url)
  await page.getByRole('button', { name: '允许访问', exact: true }).click()
  await expect(page.getByLabel('密码', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: '允许访问', exact: true })).toHaveCount(0)
  expect(writes).toBe(1)
})

for (const [status, message] of [[403, '无法验证此次操作'], [404, '此请求已失效'], [422, '授权请求格式无效'], [429, '操作过于频繁'], [503, '授权服务暂不可用']] as const) {
  test(`authorization error ${status} shows recovery without approving`, async ({ page }) => {
    await setup(page)
    await page.route(`**/api/v1/account/authorizations/${id}`, route => route.fulfill({ status, json: { code: 'example-problem' } }))
    await page.goto(url)
    await expect(page.getByRole('alert')).toContainText(message)
    await expect(page.getByRole('button', { name: '允许访问', exact: true })).toHaveCount(0)
  })
}

test('expired request hides confirmation and malformed handles never call the API', async ({ page }) => {
  await setup(page)
  let reads = 0
  await page.route('**/api/v1/account/authorizations/**', route => { reads++; return route.fulfill({ json: { ...details, expires_at: '2020-01-01T00:00:00Z' } }) })
  await page.goto(url)
  await expect(page.getByRole('alert')).toContainText('此请求已失效')
  await expect(page.getByRole('button', { name: '允许访问', exact: true })).toHaveCount(0)
  await page.goto(`${origin}/?authorization_request=invalid`)
  await expect(page.getByRole('alert')).toContainText('授权请求格式无效')
  expect(reads).toBe(1)
})

for (const [status, message] of [[403, '无法验证此次操作'], [404, '此请求已失效'], [422, '授权请求格式无效'], [429, '操作过于频繁'], [503, '授权服务暂不可用']] as const) {
  test(`confirmation failure ${status} never resubmits or navigates`, async ({ page }) => {
    await setup(page)
    let writes = 0
    await page.route(`**/api/v1/account/authorizations/${id}`, route => {
      if (route.request().method() === 'GET') return route.fulfill({ json: details })
      writes++
      return route.fulfill({ status, json: { code: 'example-problem' } })
    })
    await page.goto(url)
    await page.getByRole('button', { name: '允许访问', exact: true }).click()
    await expect(page.getByRole('alert')).toContainText(message)
    await expect(page).toHaveURL(url)
    const approve = page.getByRole('button', { name: '允许访问', exact: true })
    if (status === 404) await expect(approve).toHaveCount(0)
    else await expect(approve).toBeDisabled()
    expect(writes).toBe(1)
  })
}

test('English dark authorization fits a narrow screen without loading mascot art', async ({ page }) => {
  await setup(page)
  await page.route(`**/api/v1/account/authorizations/${id}`, route => route.fulfill({ json: details }))
  await page.setViewportSize({ width: 375, height: 812 })
  await page.goto(url)
  await selectLanguage(page, 'English')
  await chooseTheme(page, 'Change theme', 'Dark')
  await expect(page.getByRole('button', { name: 'Allow access', exact: true })).toBeEnabled()
  await expect(page.locator('main img')).toHaveCount(0)
  expect(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)).toBe(false)
})
