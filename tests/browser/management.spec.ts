import { expect, test } from '@playwright/test'
import { mockPortal, previewUser } from './helpers/portal'
const origin = 'http://127.0.0.1:53000'
const id = 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa'
const application = () => ({ id, client_id: 'test-client', public: false, enabled: true, revision: 1, active_version: 0, created_at: '2026-10-19T00:00:00Z', latest: { version: 1, status: 'draft', review_reason: '', config: { name: 'Test application', description: 'Test app', homepage_url: 'https://example.com', redirect_uris: ['https://example.com/callback'], scopes: ['openid', 'profile'] } } })

test('developer navigation keeps stable transition containers across all layouts', async ({ page }) => {
  await mockPortal(page)
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.message))
  await page.goto(`${origin}/developer`)
  const steps = [
    ['/developer/docs', '快速开始'], ['/developer/docs/endpoints', 'API 参考'], ['/developer/applications', '我的应用'], ['/developer', '让你的应用，连接 Moetopia。'],
  ]
  for (let repeat = 0; repeat < 3; repeat++) for (const [path, heading] of steps) {
    await page.locator(`a[href="${path}"]:visible`).first().click()
    await expect(page).toHaveURL(`${origin}${path}`)
    await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
    await page.waitForTimeout(240)
  }
  expect(errors).toEqual([])
})

test('direct application details require RP login even with an unrelated authorization query', async ({ page }) => {
  await mockPortal(page, false)
  await page.route('**/api/v1/portal/login?**', route => route.fulfill({ body: 'OAuth started' }))
  await page.goto(`${origin}/developer/applications/${id}?authorization_request=${'a'.repeat(43)}`)
  await expect(page).toHaveURL(`${origin}/api/v1/portal/login?return_to=%2Fdeveloper%2Fapplications`)
})

test('profile handles revision conflicts, avatar upload and removal', async ({ page }) => {
  await mockPortal(page)
  let user = { ...previewUser, avatar_url: '' }
  let conflict = true
  await page.route('**/api/v1/portal/context', route => route.fulfill({ json: { authenticated: true, csrf_token: 'portal-csrf', user } }))
  await page.route('**/api/v1/portal/profile', route => {
    expect(route.request().headers()['x-moetopia-csrf']).toBe('portal-csrf')
    const body = route.request().postDataJSON()
    if (conflict) { conflict = false; user.revision++; return route.fulfill({ status: 409, json: { code: 'account_revision_conflict' } }) }
    expect(body.revision).toBe(user.revision)
    user = { ...user, ...body, revision: user.revision + 1 }
    return route.fulfill({ json: user })
  })
  const uploads: string[] = []
  await page.route('**/api/v1/portal/avatar', route => {
    const body = route.request().postDataJSON()
    uploads.push(body.image_base64)
    expect(body.revision).toBe(user.revision)
    user = { ...user, avatar_url: body.image_base64 ? '/api/v1/avatars/test?v=2' : '', revision: user.revision + 1 }
    return route.fulfill({ json: user })
  })
  await page.goto(`${origin}/profile`)
  await page.getByLabel('昵称', { exact: true }).fill('Changed')
  await page.getByRole('button', { name: '保存修改' }).click()
  await expect(page.getByRole('alert')).toContainText('资料或状态已变化')
  await page.getByLabel('昵称', { exact: true }).fill('Updated')
  await page.getByRole('button', { name: '保存修改' }).click()
  await expect(page.locator('.moe-profile-identity')).toContainText('Updated')
  await page.locator('input[type=file]').setInputFiles({ name: 'avatar.png', mimeType: 'image/png', buffer: Buffer.from('synthetic-avatar') })
  await page.getByRole('button', { name: '移除头像' }).click()
  await expect.poll(() => uploads.length).toBe(2)
  expect(uploads[0]).not.toContain('data:')
  expect(uploads[1]).toBe('')
})

test('email change binds its challenge and clears sessions after confirmation', async ({ page }) => {
  await mockPortal(page)
  await page.route('**/api/v1/account/context', route => route.fulfill({ json: { csrf_token: 'identity-csrf' } }))
  await page.route('**/api/v1/portal/email-changes', route => {
    expect(route.request().postDataJSON()).toEqual({ current_password: 'current password', email: 'new@example.com', locale: 'zh-CN' })
    return route.fulfill({ json: { challenge_id: 'email-challenge' } })
  })
  await page.route('**/api/v1/portal/email-changes/confirm', route => {
    expect(route.request().headers()['x-moetopia-csrf']).toBe('portal-csrf')
    expect(route.request().postDataJSON()).toEqual({ current_password: 'current password', challenge_id: 'email-challenge', code: '12345678' })
    return route.fulfill({ status: 204 })
  })
  await page.goto(`${origin}/profile`)
  const form = page.locator('section').filter({ has: page.getByRole('heading', { name: '邮箱地址', exact: true }) }).locator('form')
  await form.getByLabel('当前密码', { exact: true }).fill('current password')
  await form.getByLabel('新邮箱', { exact: true }).fill('new@example.com')
  await form.getByRole('button', { name: '发送验证码', exact: true }).click()
  await form.getByLabel('邮箱验证码').fill('12345678')
  await form.getByRole('button', { name: '确认更换邮箱' }).click()
  await expect(page).toHaveURL(`${origin}/auth/login`)
})

test('password change rejects mismatch and wrong current password before signing out', async ({ page }) => {
  await mockPortal(page)
  let attempts = 0
  await page.route('**/api/v1/account/context', route => route.fulfill({ json: { csrf_token: 'identity-csrf' } }))
  await page.route('**/api/v1/portal/password', route => { attempts++; return attempts === 1 ? route.fulfill({ status: 401, json: { code: 'reauthentication_failed' } }) : route.fulfill({ status: 204 }) })
  await page.goto(`${origin}/security`)
  const form = page.locator('section').filter({ has: page.getByRole('heading', { name: '修改密码', exact: true }) }).locator('form')
  await form.getByLabel('当前密码', { exact: true }).fill('wrong current password')
  await form.getByLabel('新密码', { exact: true }).fill('a new secure passphrase')
  await form.getByLabel('确认新密码', { exact: true }).fill('a different passphrase')
  await form.getByRole('button', { name: '修改密码', exact: true }).click()
  await expect(form.getByRole('alert')).toContainText('不一致')
  expect(attempts).toBe(0)
  await form.getByLabel('确认新密码', { exact: true }).fill('a new secure passphrase')
  await form.getByRole('button', { name: '修改密码', exact: true }).click()
  await expect(form.getByRole('alert')).toContainText('当前密码不正确')
  await form.getByLabel('当前密码', { exact: true }).fill('correct current password')
  await form.getByRole('button', { name: '修改密码', exact: true }).click()
  await expect(page).toHaveURL(`${origin}/auth/login`)
})

test('password recovery uses identity CSRF and never automatically signs in', async ({ page }) => {
  await page.route('**/api/v1/account/context', route => route.fulfill({ json: { csrf_token: 'identity-csrf' } }))
  await page.route('**/api/v1/account/password-resets', route => {
    expect(route.request().headers()['x-moetopia-csrf']).toBe('identity-csrf')
    return route.fulfill({ json: { challenge_id: 'reset-challenge' } })
  })
  await page.route('**/api/v1/account/password-resets/confirm', route => {
    expect(route.request().postDataJSON()).toEqual({ challenge_id: 'reset-challenge', code: '12345678', new_password: 'new secure passphrase' })
    return route.fulfill({ status: 204 })
  })
  await page.goto(`${origin}/auth/forgot-password`)
  await page.getByLabel('邮箱', { exact: true }).fill('unknown@example.com')
  await page.getByRole('button', { name: '发送验证码' }).click()
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('设置新密码')
  await page.getByLabel('邮箱验证码').fill('12345678')
  await page.getByLabel('密码', { exact: true }).fill('new secure passphrase')
  await page.getByRole('button', { name: '设置新密码' }).click()
  await expect(page.getByRole('alert')).toContainText('密码已重置')
  await expect(page).toHaveURL(`${origin}/auth/forgot-password`)
})

test('authorization records paginate and revoke individual grant families', async ({ page }) => {
  await mockPortal(page)
  const grant = { id: 'one', client_id: 'client', client_name: 'Example', created_at: '2026-10-18T02:00:00Z', expires_at: '2030-11-18T02:00:00Z', scopes: ['openid'], audiences: [] }
  await page.route('**/api/v1/portal/authorizations*', route => route.fulfill({ json: { items: [{ ...grant, id: route.request().url().includes('cursor=') ? 'two' : 'one' }], next_cursor: route.request().url().includes('cursor=') ? '' : 'next' } }))
  await page.route('**/api/v1/portal/authorizations/one', route => { expect(route.request().method()).toBe('DELETE'); return route.fulfill({ status: 204 }) })
  await page.goto(`${origin}/security`)
  await page.getByRole('button', { name: '加载更多' }).click()
  await expect(page.getByRole('button', { name: '撤销授权', exact: true })).toHaveCount(2)
  await page.getByRole('button', { name: '撤销授权', exact: true }).first().click()
  await page.getByRole('dialog').getByRole('button', { name: '撤销授权', exact: true }).click()
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(page.getByRole('button', { name: '撤销授权', exact: true })).toHaveCount(1)
})

test('developer creates confidential app, edits draft, submits, rotates and deletes', async ({ page }) => {
  await mockPortal(page)
  let app = application()
  const writes: string[] = []
  await page.route('**/api/v1/developer/applications', route => {
    if (route.request().method() === 'GET') return route.fulfill({ json: { items: [], next_cursor: '' } })
    const body = route.request().postDataJSON()
    expect(body.public).toBe(false)
    app.latest.config = body.config
    return route.fulfill({ json: { application: app, client_secret: 'synthetic-create-secret' } })
  })
  await page.route(`**/api/v1/developer/applications/${id}`, route => route.request().method() === 'DELETE' ? route.fulfill({ status: 204 }) : route.fulfill({ json: app }))
  await page.route(`**/api/v1/developer/applications/${id}/*`, route => {
    const operation = route.request().url().split('/').at(-1)!
    if (route.request().method() === 'GET') return route.fulfill({ json: operation === 'test-members' ? { revision: app.revision, limit: 20, items: [] } : { items: [{ ...app.latest, created_at: app.created_at }], next_before: 0, active_version: app.active_version, latest_version: app.latest.version } })
    writes.push(operation)
    const body = route.request().postDataJSON()
    expect(route.request().headers()['x-moetopia-csrf']).toBe('portal-csrf')
    expect(body.revision).toBe(app.revision)
    app.revision++
    if (operation === 'draft') app.latest.config = body.config
    if (operation === 'submit') app.latest.status = 'pending'
    if (operation === 'availability') app.enabled = body.enabled
    return route.fulfill({ json: operation === 'secret' ? { client_secret: 'synthetic-rotated-secret' } : app })
  })
  await page.goto(`${origin}/developer/applications`)
  await page.getByRole('button', { name: '创建应用', exact: true }).first().click()
  const create = page.getByRole('dialog', { name: '创建应用' })
  await create.getByLabel('应用名称', { exact: true }).fill('My app')
  await create.getByLabel('回调地址 1', { exact: true }).fill('https://example.com/callback')
  await create.getByRole('checkbox', { name: '公开客户端（浏览器或原生应用）' }).uncheck()
  await create.getByRole('button', { name: '创建应用', exact: true }).click()
  const reveal = page.getByRole('dialog', { name: '客户端凭据' })
  await expect(reveal.getByLabel('client_secret')).toHaveValue('synthetic-create-secret')
  await reveal.getByRole('button', { name: '关闭', exact: true }).last().click()
  await page.getByRole('link', { name: /^应用详情:/ }).click()
  await expect(page).toHaveURL(`${origin}/developer/applications/${id}`)
  await page.getByRole('button', { name: '编辑应用', exact: true }).click()
  const edit = page.getByRole('dialog', { name: '编辑应用' })
  await edit.getByLabel('应用名称', { exact: true }).fill('Edited app')
  await edit.getByRole('button', { name: '保存草稿' }).click()
  await expect(page.getByRole('heading', { level: 2, name: 'Edited app' })).toBeVisible()
  await page.getByRole('button', { name: '提交审核', exact: true }).click()
  await page.getByRole('dialog').getByRole('button', { name: '确认操作' }).click()
  await expect(page.getByRole('button', { name: '编辑应用', exact: true })).toBeDisabled()
  await page.getByRole('button', { name: '轮换密钥', exact: true }).click()
  await page.getByRole('dialog').getByLabel('当前密码', { exact: true }).fill('current secure passphrase')
  await page.getByRole('dialog').getByRole('button', { name: '确认操作' }).click()
  await expect(reveal.getByLabel('client_secret')).toHaveValue('synthetic-rotated-secret')
  await reveal.getByRole('button', { name: '关闭', exact: true }).last().click()
  await expect(page.getByLabel('client_secret')).toHaveCount(0)
  await page.getByRole('button', { name: '停用应用', exact: true }).click()
  await page.getByRole('dialog').getByRole('button', { name: '确认操作' }).click()
  await page.getByRole('button', { name: '删除应用', exact: true }).click()
  await page.getByRole('dialog').getByRole('button', { name: '确认操作' }).click()
  await expect(page).toHaveURL(`${origin}/developer/applications`)
  expect(writes).toEqual(['draft', 'submit', 'secret', 'availability'])
})
