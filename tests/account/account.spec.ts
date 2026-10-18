import { randomUUID } from 'node:crypto'
import { expect, test } from '@playwright/test'

test('invitation registration, Mailpit verification, login and logout', async ({ page, request }) => {
  const invitation = process.env.MOETOPIA_TEST_INVITATION
  const smtp = process.env.MOETOPIA_SMTP_ADDRESS
  if (!invitation || !smtp) throw new Error('Dedicated account test environment is required')
  const mailpit = `http://${smtp.split(':')[0]}:58025`
  const email = `browser-${randomUUID()}@moetopia.test`
  let messageID: string | undefined
  const exceptions: string[] = []
  page.on('pageerror', error => exceptions.push(error.message))
  await page.route('**/api/**', async route => {
    const url = new URL(route.request().url())
    const response = await route.fetch({ url: `http://127.0.0.1:58080${url.pathname}${url.search}` })
    await route.fulfill({ response })
  })
  try {
    await page.goto('/')
    await expect(page.getByRole('button', { name: '登录', exact: true }).last()).toBeEnabled()
    await page.getByRole('tab', { name: '创建账号', exact: true }).click()
    await page.getByLabel('邮箱', { exact: true }).fill(email)
    await page.getByLabel('昵称', { exact: true }).fill('Browser reader')
    await page.getByLabel('邀请码', { exact: true }).fill(invitation)
    await page.getByLabel('密码', { exact: true }).fill('a long browser test passphrase')
    await page.getByRole('button', { name: '发送验证码', exact: true }).click()
    await expect(page.getByRole('alert')).toContainText('验证码将发送')
    await expect.poll(async () => {
      const response = await request.get(`${mailpit}/api/v1/search`, { params: { query: `to:${email}` } })
      if (!response.ok()) return false
      const body = await response.json() as { messages?: { ID: string }[] }
      messageID = body.messages?.[0]?.ID
      return !!messageID
    }, { timeout: 20000 }).toBe(true)
    const response = await request.get(`${mailpit}/api/v1/message/${messageID}`)
    const message = await response.json() as { Text?: string }
    const code = message.Text?.match(/[0-9]{8}/)?.[0]
    if (!code) throw new Error('Verification email did not contain a code')
    await page.getByLabel('邮箱验证码', { exact: true }).fill(code)
    await page.getByRole('button', { name: '验证并创建账号', exact: true }).click()
    await expect(page.getByRole('alert')).toContainText('账号已创建')
    await page.getByLabel('密码', { exact: true }).fill('a long browser test passphrase')
    await page.getByRole('button', { name: '登录', exact: true }).last().click()
    await expect(page.getByRole('heading', { name: '欢迎回来，Browser reader' })).toBeVisible()
    await expect(page.getByRole('link', { name: '管理登录会话', exact: true })).toBeVisible()
    await page.reload()
    await expect(page.getByRole('heading', { name: '欢迎回来，Browser reader' })).toBeVisible()
    expect((await page.context().cookies()).find(cookie => cookie.name === 'moetopia_account')?.httpOnly).toBe(true)
    await page.getByRole('button', { name: '退出登录', exact: true }).click()
    await expect(page.getByRole('heading', { name: '欢迎回来', exact: true })).toBeVisible()
    await page.reload()
    await expect(page.getByRole('heading', { name: '欢迎回来', exact: true })).toBeVisible()
    expect(exceptions).toEqual([])
  } finally {
    if (messageID && /^[A-Za-z0-9-]+$/.test(messageID)) {
      const deletion = await request.delete(`${mailpit}/api/v1/messages`, { data: { IDs: [messageID] } })
      expect(deletion.ok()).toBe(true)
    }
  }
})
