import { expect, test } from '@playwright/test'
import { verifyPortalUI } from './portal-ui'

test('real user-site pages manage applications, profile and recovery with isolated fixtures', async ({ page, request }) => {
  const email = process.env.MOETOPIA_TEST_OIDC_EMAIL
  const password = process.env.MOETOPIA_TEST_OIDC_PASSWORD
  const invitation = process.env.MOETOPIA_TEST_INVITATION
  const smtp = process.env.MOETOPIA_SMTP_ADDRESS
  if (!email || !password || !invitation || !smtp) throw new Error('An isolated management fixture is required')
  const mailpit = `http://${smtp.split(':')[0]}:58025`
  const messages: string[] = []
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.name))
  const verificationCode = async () => {
    let id = ''
    await expect.poll(async () => {
      const response = await request.get(`${mailpit}/api/v1/search`, { params: { query: `to:${email}` } })
      if (!response.ok()) return false
      const result = await response.json() as { messages?: { ID: string }[] }
      id = result.messages?.find(message => !messages.includes(message.ID))?.ID ?? ''
      return !!id
    }, { timeout: 20000 }).toBe(true)
    messages.push(id)
    const message = await (await request.get(`${mailpit}/api/v1/message/${id}`)).json() as { Text?: string }
    const code = message.Text?.match(/[0-9]{8}/)?.[0]
    if (!code) throw new Error('Expected a test verification code')
    return code
  }
  try {
    await page.goto('/auth/register')
    await page.getByLabel('邮箱', { exact: true }).fill(email)
    await page.getByLabel('昵称', { exact: true }).fill('Management reader')
    await page.getByLabel('邀请码', { exact: true }).fill(invitation)
    await page.getByLabel('密码', { exact: true }).fill(password)
    await page.getByRole('button', { name: '发送验证码', exact: true }).click()
    await page.getByLabel('邮箱验证码', { exact: true }).fill(await verificationCode())
    await page.getByRole('button', { name: '验证并创建账号', exact: true }).click()
    await expect(page.getByRole('heading', { level: 1 })).toHaveText('欢迎回来')
    const identity = await (await page.context().request.get('/api/v1/account/context')).json() as { csrf_token: string }
    const login = await page.context().request.post('/api/v1/account/login', { data: { email, password }, headers: { Origin: 'http://127.0.0.1:54305', 'X-Moetopia-CSRF': identity.csrf_token } })
    expect(login.status()).toBe(200)
    await verifyPortalUI(page, true)
    await page.goto('/auth/forgot-password')
    await page.getByLabel('邮箱', { exact: true }).fill(email)
    await page.getByRole('button', { name: '发送验证码', exact: true }).click()
    await page.getByLabel('邮箱验证码', { exact: true }).fill(await verificationCode())
    await page.getByLabel('密码', { exact: true }).fill(`${password}-reset`)
    await page.getByRole('button', { name: '设置新密码', exact: true }).click()
    await expect(page.getByRole('alert')).toHaveText('密码已重置，请重新登录。')
    const after = await (await page.context().request.get('/api/v1/account/context')).json() as { user?: unknown }
    expect(after.user).toBeUndefined()
    expect(errors).toEqual([])
  } finally {
    if (messages.length) expect((await request.delete(`${mailpit}/api/v1/messages`, { data: { IDs: messages } })).ok()).toBe(true)
  }
})
