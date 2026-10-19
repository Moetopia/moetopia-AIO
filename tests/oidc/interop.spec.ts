import { expect, test } from '@playwright/test'

test('real account proxy and independent RP validate the browser authorization lifecycle', async ({ page, request }) => {
  const email = process.env.MOETOPIA_TEST_OIDC_EMAIL
  const password = process.env.MOETOPIA_TEST_OIDC_PASSWORD
  const invitation = process.env.MOETOPIA_TEST_INVITATION
  const smtp = process.env.MOETOPIA_SMTP_ADDRESS
  if (!email || !password || !invitation || !smtp) throw new Error('An isolated OIDC fixture is required')
  const mailpit = `http://${smtp.split(':')[0]}:58025`
  let messageID: string | undefined
  const pageErrors: string[] = []
  page.on('pageerror', error => pageErrors.push(error.name))
  const complete = async (decision = '允许访问') => {
    await expect(page.getByRole('button', { name: decision, exact: true })).toBeEnabled()
    await page.getByRole('button', { name: decision, exact: true }).click()
    await page.waitForURL('http://localhost:54310/result')
    return JSON.parse(await page.locator('body').innerText()) as Record<string, unknown>
  }
  try {
    const proxied = await request.get('/.well-known/openid-configuration')
    const direct = await request.get('http://127.0.0.1:58490/.well-known/openid-configuration')
    expect(proxied.status()).toBe(200)
    expect(await proxied.json()).toEqual(await direct.json())
    const detailsRequest = page.waitForResponse(response => new URL(response.url()).pathname.startsWith('/api/v1/account/authorizations/') && response.request().method() === 'GET')
    await page.goto('http://localhost:54310/login')
    await expect(page.getByRole('heading', { name: '请登录后继续', exact: true })).toBeVisible()
    const detailsResponse = await detailsRequest
    const detailsBody = await detailsResponse.json() as { code?: string }
    expect({ status: detailsResponse.status(), error: detailsBody.code ?? '', scriptErrors: pageErrors }).toEqual({ status: 200, error: '', scriptErrors: [] })
    await expect(page.getByRole('heading', { name: 'Independent OIDC test RP', exact: true })).toBeVisible()
    const initial = await page.context().cookies()
    const browserCookie = initial.find(cookie => cookie.name === 'moetopia_authorization')
    const anonymousCookie = initial.find(cookie => cookie.name === 'moetopia_account')
    expect(browserCookie?.httpOnly).toBe(true)
    await page.getByRole('button', { name: '立即注册', exact: true }).click()
    await page.getByLabel('邮箱', { exact: true }).fill(email)
    await page.getByLabel('昵称', { exact: true }).fill('OIDC reader')
    await page.getByLabel('邀请码', { exact: true }).fill(invitation)
    await page.getByLabel('密码', { exact: true }).fill(password)
    await page.getByRole('button', { name: '发送验证码', exact: true }).click()
    await expect.poll(async () => {
      const response = await request.get(`${mailpit}/api/v1/search`, { params: { query: `to:${email}` } })
      if (!response.ok()) return false
      const body = await response.json() as { messages?: { ID: string }[] }
      messageID = body.messages?.[0]?.ID
      return Boolean(messageID)
    }, { timeout: 20000 }).toBe(true)
    const message = await (await request.get(`${mailpit}/api/v1/message/${messageID}`)).json() as { Text?: string }
    const code = message.Text?.match(/[0-9]{8}/)?.[0]
    if (!code) throw new Error('Expected a test verification code')
    await page.getByLabel('邮箱验证码', { exact: true }).fill(code)
    await page.getByRole('button', { name: '验证并创建账号', exact: true }).click()
    await expect(page.getByRole('alert')).toContainText('账号已创建')
    await page.getByLabel('密码', { exact: true }).fill(password)
    await page.getByRole('button', { name: '登录', exact: true }).last().click()
    await expect(page.getByRole('button', { name: '允许访问', exact: true })).toBeEnabled()
    const authenticated = await page.context().cookies()
    expect(authenticated.find(cookie => cookie.name === 'moetopia_authorization')?.value === browserCookie?.value).toBe(true)
    expect(authenticated.find(cookie => cookie.name === 'moetopia_account')?.value === anonymousCookie?.value).toBe(false)
    expect(await complete()).toMatchObject({ status: 'approved', signature_checked: true, userinfo_checked: true, identity_checked: true })
    await page.goto('http://localhost:54310/replay')
    expect(JSON.parse(await page.locator('body').innerText())).toEqual({ replay_rejected: true, prior_token_revoked: true })
    await page.goto('http://localhost:54310/login?mode=deny')
    expect(await complete('拒绝')).toMatchObject({ status: 'denied' })
    await page.goto('http://localhost:54310/login?mode=login')
    await expect(page.getByRole('alert')).toContainText('请重新验证账号')
    await page.getByLabel('邮箱', { exact: true }).fill(email)
    await page.getByLabel('密码', { exact: true }).fill(password)
    await page.getByRole('button', { name: '登录', exact: true }).last().click()
    expect(await complete()).toMatchObject({ status: 'approved' })
    for (const mode of ['bad-state', 'bad-nonce', 'bad-signature']) {
      await page.goto(`http://localhost:54310/login?mode=${mode}`)
      expect(await complete()).toMatchObject({ status: 'rejected' })
    }
    expect(pageErrors).toEqual([])
  } finally {
    if (messageID && /^[A-Za-z0-9-]+$/.test(messageID)) {
      const response = await request.delete(`${mailpit}/api/v1/messages`, { data: { IDs: [messageID] } })
      expect(response.ok()).toBe(true)
    }
  }
})
