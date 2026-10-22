import { expect, test, type Page } from '@playwright/test'
import { createHmac } from 'node:crypto'

const origin = 'http://admin.localhost:54311'
const totp = (secret: string) => {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567'
  let bits = ''
  for (const char of secret) bits += alphabet.indexOf(char).toString(2).padStart(5, '0')
  const key = Buffer.from(Array.from({ length: Math.floor(bits.length / 8) }, (_, i) => Number.parseInt(bits.slice(i * 8, i * 8 + 8), 2)))
  const counter = Buffer.alloc(8)
  counter.writeBigUInt64BE(BigInt(Math.floor(Date.now() / 30000)))
  const digest = createHmac('sha1', key).update(counter).digest()
  return String((digest.readUInt32BE(digest[19]! & 15) & 0x7fffffff) % 1000000).padStart(6, '0')
}
const call = async (page: Page, path: string, body?: Record<string, unknown>) => page.evaluate(async ({ path, body }) => {
  const context = await (await fetch('/api/v1/admin/context')).json() as { csrf_token: string }
  const response = await fetch(path, body ? { method: 'POST', headers: { 'content-type': 'application/json', 'X-Moetopia-CSRF': context.csrf_token }, body: JSON.stringify(body) } : undefined)
  return { status: response.status, body: await response.json() as Record<string, unknown> }
}, { path, body })

test('administrator RP requires fresh login and MFA, rotates sessions and recovers without privilege bypass', async ({ page }) => {
  const email = process.env.MOETOPIA_TEST_OIDC_EMAIL
  const password = process.env.MOETOPIA_TEST_OIDC_PASSWORD
  if (!email || !password) throw new Error('Isolated administrator fixture required')
  await page.goto(origin + '/api/v1/admin/login?return_to=/applications')
  await expect(page.getByRole('heading', { name: '请登录后继续', exact: true })).toBeVisible()
  await page.getByLabel('邮箱', { exact: true }).fill(email)
  await page.getByLabel('密码', { exact: true }).fill(password)
  await page.getByRole('button', { name: '登录', exact: true }).last().click()
  await expect(page.getByRole('button', { name: '允许访问', exact: true })).toBeEnabled()
  await page.getByRole('button', { name: '允许访问', exact: true }).click()
  await page.waitForURL(origin + '/applications')
  const before = (await page.context().cookies(origin)).find(c => c.name === 'moetopia_admin')
  expect(Boolean(before?.httpOnly && before.domain === 'admin.localhost')).toBe(true)
  expect((await call(page, '/api/v1/admin/applications')).status).toBe(403)
  await page.getByLabel('当前 Moetopia 登录密码', { exact: true }).fill(password)
  await page.getByRole('button', { name: '确认', exact: true }).click()
  const secret = await page.locator('pre').innerText()
  const code = totp(secret)
  await page.getByLabel('验证器验证码', { exact: true }).fill(code)
  await page.getByRole('button', { name: '确认', exact: true }).click()
  const stored = page.getByRole('button', { name: '我已安全保存', exact: true })
  await expect(stored).toBeVisible()
  const codes = (await page.locator('pre').innerText()).trim().split('\n')
  await stored.click()
  await expect(page.getByRole('heading', { name: '应用审核', exact: true })).toBeVisible()
  const confirmed = await call(page, '/api/v1/admin/context')
  expect(confirmed.status).toBe(200)
  const factor = confirmed.body.mfa as { enrolled: boolean, verified: boolean }
  expect(factor).toMatchObject({ enrolled: true, verified: true })
  expect(codes.length).toBe(10)
  const after = (await page.context().cookies(origin)).find(c => c.name === 'moetopia_admin')
  expect(Boolean(after && before && after.value !== before.value)).toBe(true)
  expect((await call(page, '/api/v1/admin/sessions')).status).toBe(200)
  expect((await call(page, '/api/v1/admin/mfa/verify', { code })).status).toBe(422)
  const recovered = await call(page, '/api/v1/admin/mfa/recover', { current_password: password, recovery_code: codes[0] })
  expect(recovered.status).toBe(200)
  expect(recovered.body.mfa).toMatchObject({ enrolled: false, verified: false, recovery_codes_remaining: 0 })
  expect((await call(page, '/api/v1/admin/applications')).status).toBe(403)
  expect((await call(page, '/api/v1/admin/mfa/recover', { current_password: password, recovery_code: codes[1] })).status).toBe(422)
  expect((await call(page, '/api/v1/admin/logout', {})).status).toBe(200)
  const central = await page.context().request.get('http://127.0.0.1:54305/api/v1/account/context')
  expect(Boolean((await central.json() as { user?: unknown }).user)).toBe(true)
  expect((await page.context().cookies('http://127.0.0.1:54305')).some(c => c.name === 'moetopia_admin')).toBe(false)
})
