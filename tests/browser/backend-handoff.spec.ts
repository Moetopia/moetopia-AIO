import { expect, test, type Page } from '@playwright/test'
import { mockPortal, previewUser } from './helpers/portal'
import jsQR from 'jsqr'

const admin = 'http://127.0.0.1:53001'
const account = 'http://127.0.0.1:53000'
const application = { id: '11111111-1111-4111-8111-111111111111', client_id: 'example-client', public: true, enabled: true, revision: 3, active_version: 0, created_at: '2026-10-22T00:00:00Z', latest: { version: 1, status: 'pending', review_reason: '', config: { name: '交接测试应用', description: '测试资料', homepage_url: 'https://example.com', redirect_uris: ['https://example.com/callback'], post_logout_redirect_uris: ['https://example.com/logout'], scopes: ['openid', 'profile'] } } }
const adminSetup = async (page: Page, verified = true, enrolled = true, emergency = false) => {
  const state = { verified, enrolled, emergency, csrf: 'admin-example-csrf', writes: [] as { path: string; csrf?: string; body: Record<string, unknown> }[] }
  const context = () => ({ authenticated: true, csrf_token: state.csrf, login_url: '/api/v1/admin/login', user: { ...previewUser, has_password: true }, mfa: { enrolled: state.enrolled, verified: state.verified, emergency_required: state.emergency, recovery_codes_remaining: 10 } })
  await page.route('**/api/**', async route => {
    const url = new URL(route.request().url())
    const path = url.pathname
    const write = route.request().method() !== 'GET'
    const body = write && route.request().postData() ? route.request().postDataJSON() as Record<string, unknown> : {}
    if (write) state.writes.push({ path, csrf: route.request().headers()['x-moetopia-csrf'], body })
    if (path === '/api/v1/status') return route.fulfill({ json: { service: 'moetopia' } })
    if (path.endsWith('/context')) return route.fulfill({ json: context() })
    if (path.endsWith('/mfa/enrollment')) return route.fulfill({ json: { secret: 'EXAMPLE-ONLY-SETUP-KEY', otpauth_uri: 'otpauth://example', expires_at: '2030-01-01T00:00:00Z' } })
    if (path.endsWith('/mfa/enrollment/confirm')) { state.verified = true; state.enrolled = true; state.csrf = 'rotated-example-csrf'; return route.fulfill({ json: { ...context(), recovery_codes: ['EXAMPLE-ONLY-RECOVERY'] } }) }
    if (path.endsWith('/mfa/verify')) { state.verified = true; state.csrf = 'rotated-example-csrf'; return route.fulfill({ json: context() }) }
    if (path.endsWith('/mfa/recover') || path.endsWith('/mfa/emergency')) { state.enrolled = false; state.verified = false; state.emergency = false; state.csrf = 'recovered-example-csrf'; return route.fulfill({ json: context() }) }
    if (path.endsWith('/statistics')) return route.fulfill({ json: { timezone: 'UTC', generated_at: '2026-10-22T00:00:00Z', identity: { total_users: 12, enabled_users: 10, pending_deletions: 2, recording_since: '2026-10-21T12:00:00Z', registrations: [{ day: '2026-10-20', partial: false }, { day: '2026-10-21', registrations: 0, partial: true }] }, applications: { total: 2, pending: 1, published_enabled: 1, suspended: 0 }, notifications: { queued: 1, in_flight: 0, retry_exhausted: 1 } } })
    if (path.endsWith('/registration-policy')) return route.fulfill({ json: { mode: body.mode ?? 'invite', revision: 3 } })
    if (path.endsWith('/sessions')) return route.fulfill({ json: { current_id: 'example-current', items: [{ id: 'example-current', created_at: '2026-10-22T00:00:00Z', expires_at: '2030-01-01T00:00:00Z' }] } })
    if (path.endsWith('/mfa/recovery-codes')) return route.fulfill({ json: { recovery_codes: ['EXAMPLE-ONLY-ROTATED'] } })
    if (path.endsWith('/versions')) return route.fulfill({ json: { items: [{ ...application.latest, created_at: application.created_at }], next_before: 0, active_version: 0, latest_version: 1 } })
    if (path.includes(`/applications/${application.id}`)) return route.fulfill({ json: application })
    if (path.endsWith('/applications')) return route.fulfill({ json: { items: [application], next_cursor: '' } })
    if (path.endsWith('/users')) return route.fulfill({ json: { items: [{ ...previewUser, has_password: true, enabled: true, created_at: application.created_at, disabled_reason: '' }], next_cursor: '' } })
    if (path.endsWith('/invitations')) return route.fulfill({ json: write ? { id: 'example-invitation-id', code: 'EXAMPLE-ONLY-INVITATION', expires_at: '2030-01-01T00:00:00Z' } : { items: [], next_cursor: '' } })
    if (path.endsWith('/audit-events')) return route.fulfill({ json: { items: [{ id: 'event', action: 'application.review', actor_id: 'example-actor', target_id: application.id, outcome: 'success', reason: '测试审核', request_id: 'example-request', created_at: application.created_at }], next_cursor: '' } })
    if (path.endsWith('/principals')) return route.fulfill({ json: { items: [{ id: 'example-admin', enabled: true, enrolled: true, emergency_required: false, revision: 5, created_at: application.created_at }], next_after: '' } })
    if (path.endsWith('/notifications/failed')) return route.fulfill({ json: { items: [{ id: 'example-notification', attempts: 8, available_at: application.created_at, expires_at: '2030-01-01T00:00:00Z' }], next_after: '' } })
    return route.fulfill({ status: 204 })
  })
  return state
}

test('administrator enrollment withholds protected pages until one-time codes are acknowledged', async ({ page }) => {
  const state = await adminSetup(page, false, false)
  await page.goto(admin + '/users')
  await expect(page.getByText('绑定验证器', { exact: true })).toBeVisible()
  await expect(page.getByRole('table')).toHaveCount(0)
  await page.getByLabel('当前 Moetopia 登录密码', { exact: true }).fill('example-only-password')
  await page.getByRole('button', { name: '确认', exact: true }).click()
  await expect(page.locator('pre')).toHaveText('EXAMPLE-ONLY-SETUP-KEY')
  const qr = page.getByRole('img', { name: '使用验证器扫描二维码', exact: true })
  await expect(qr).toBeVisible()
  await expect(qr).toHaveJSProperty('naturalWidth', 256)
  const pixels = await qr.evaluate(image => {
    const canvas = document.createElement('canvas')
    const img = image as HTMLImageElement
    canvas.width = img.naturalWidth; canvas.height = img.naturalHeight
    const context = canvas.getContext('2d')!
    context.drawImage(img, 0, 0)
    return { data: Array.from(context.getImageData(0, 0, canvas.width, canvas.height).data), width: canvas.width, height: canvas.height }
  })
  expect(jsQR(new Uint8ClampedArray(pixels.data), pixels.width, pixels.height)?.data).toBe('otpauth://example')
  await page.getByLabel('验证器验证码', { exact: true }).fill('123456')
  await page.getByRole('button', { name: '确认', exact: true }).click()
  await expect(page.locator('pre')).toHaveText('EXAMPLE-ONLY-RECOVERY')
  await expect(page.getByRole('table')).toHaveCount(0)
  await page.getByRole('button', { name: '我已安全保存', exact: true }).click()
  await expect(page.getByRole('table')).toBeVisible()
  await expect(page.locator('body')).not.toContainText('EXAMPLE-ONLY-RECOVERY')
  await expect(qr).toHaveCount(0)
  expect(state.writes.map(write => write.csrf)).toEqual(['admin-example-csrf', 'admin-example-csrf'])
  await page.getByRole('button', { name: '停用', exact: true }).click()
  await page.getByLabel('操作原因').fill('测试停用')
  await page.getByRole('button', { name: '确认', exact: true }).click()
  expect(state.writes.at(-1)).toMatchObject({ csrf: 'rotated-example-csrf', body: { enabled: false, revision: 1, reason: '测试停用' } })
})

test('administrator emergency and recovery return to enrollment without verified access', async ({ page }) => {
  const state = await adminSetup(page, false, true, true)
  await page.goto(admin + '/')
  await page.getByLabel('当前 Moetopia 登录密码', { exact: true }).fill('example-only-password')
  await page.getByLabel('运维提供的一次性恢复令牌').fill('EXAMPLE-ONLY-TOKEN')
  await page.getByRole('button', { name: '确认', exact: true }).click()
  await expect(page.getByText('绑定验证器', { exact: true })).toBeVisible()
  expect(state.writes[0]?.path).toBe('/api/v1/admin/mfa/emergency')
  await expect(page.locator('body')).not.toContainText('EXAMPLE-ONLY-TOKEN')
})

test('expired authenticator setup clears both QR and manual key before restarting', async ({ page }) => {
  await page.clock.install({ time: new Date('2026-10-22T00:00:00Z') })
  await adminSetup(page, false, false)
  await page.route('**/api/v1/admin/mfa/enrollment', route => route.fulfill({ json: { secret: 'EXAMPLE-ONLY-SETUP-KEY', otpauth_uri: 'otpauth://example', expires_at: '2026-10-22T00:05:00Z' } }))
  await page.goto(admin + '/')
  await page.getByLabel('当前 Moetopia 登录密码', { exact: true }).fill('example-only-password')
  await page.getByRole('button', { name: '确认', exact: true }).click()
  await expect(page.getByRole('img', { name: '使用验证器扫描二维码' })).toBeVisible()
  await page.clock.fastForward(301000)
  await expect(page.locator('pre')).toHaveCount(0)
  await expect(page.getByRole('img', { name: '使用验证器扫描二维码' })).toHaveCount(0)
  await expect(page.getByLabel('当前 Moetopia 登录密码', { exact: true })).toBeVisible()
  await expect(page.getByRole('alert')).toContainText('设置请求已到期')
})

test('administrator review, histories, invitations, principals, notifications and settings use real contracts', async ({ page }) => {
  const state = await adminSetup(page)
  await page.goto(admin + '/applications')
  await page.getByRole('link', { name: application.latest.config.name }).click()
  await page.getByRole('button', { name: '拒绝', exact: true }).click()
  await page.getByLabel('操作原因').fill('请补充说明')
  await page.getByRole('button', { name: '确认', exact: true }).click()
  expect(state.writes.at(-1)?.body).toEqual({ revision: 3, approve: false, reason: '请补充说明' })
  await expect(page.getByText('v1 · pending')).toBeVisible()
  await page.goto(admin + '/invitations')
  await page.getByRole('button', { name: '生成邀请码', exact: true }).click()
  await page.getByRole('button', { name: '确认', exact: true }).click()
  await expect(page.locator('pre')).toHaveText('EXAMPLE-ONLY-INVITATION')
  await page.keyboard.press('Escape')
  await expect(page.locator('pre')).toHaveCount(0)
  await page.goto(admin + '/principals')
  await page.getByRole('button', { name: '授予管理员权限' }).click()
  await page.getByLabel('用户 UUID').fill('22222222-2222-4222-8222-222222222222')
  await page.getByLabel('操作原因').fill('测试授权')
  await page.getByLabel('当前密码', { exact: true }).fill('example-only-password')
  await page.getByRole('button', { name: '确认', exact: true }).click()
  expect(state.writes.at(-1)?.body).toMatchObject({ revision: 0, enabled: true, reason: '测试授权' })
  await page.goto(admin + '/notifications')
  await page.getByRole('button', { name: '重新入队', exact: true }).click()
  await page.getByLabel('操作原因').fill('确认投递服务恢复')
  await page.getByRole('button', { name: '确认', exact: true }).click()
  expect(state.writes.at(-1)?.body).toEqual({ attempts: 8, reason: '确认投递服务恢复' })
  await page.goto(admin + '/settings')
  await page.getByRole('button', { name: /重新生成恢复码/ }).click()
  await page.getByLabel('当前密码', { exact: true }).fill('example-only-password')
  await page.getByRole('button', { name: '确认', exact: true }).click()
  await expect(page.locator('pre')).toHaveText('EXAMPLE-ONLY-ROTATED')
  await page.keyboard.press('Escape')
  await expect(page.locator('pre')).toHaveCount(0)
  await page.goto(admin + '/audit')
  await expect(page.getByRole('table')).toContainText('测试审核')
  await page.goto(admin + '/')
  await page.getByRole('button', { name: '7 天', exact: true }).click()
  await expect(page.getByRole('button', { name: '7 天', exact: true })).toHaveAttribute('aria-pressed', 'true')
  await page.getByText('每日注册（UTC）', { exact: true }).last().click()
  await expect(page.getByRole('table')).toContainText('未开始记录')
})

test('expired administrator MFA returns to the factor form without OAuth loops', async ({ page }) => {
  const state = await adminSetup(page)
  await page.route('**/api/v1/admin/users*', route => { state.verified = false; return route.fulfill({ status: 403, json: { code: 'admin_mfa_required' } }) })
  await page.goto(admin + '/users')
  await expect(page.getByLabel('验证器验证码')).toBeVisible()
  await expect(page).toHaveURL(admin + '/users')
})

test('administrator state conflicts close stale actions and require a fresh explicit submission', async ({ page }) => {
  await adminSetup(page)
  let attempts = 8
  let writes = 0
  await page.route('**/api/v1/admin/notifications/failed*', route => route.fulfill({ json: { items: [{ id: 'example-notification', attempts, available_at: application.created_at, expires_at: '2030-01-01T00:00:00Z' }], next_after: '' } }))
  await page.route('**/api/v1/admin/notifications/example-notification/retry', route => { writes++; expect(route.request().postDataJSON().attempts).toBe(attempts); attempts++; return route.fulfill({ status: 409, json: { code: 'notification_state_changed' } }) })
  await page.goto(admin + '/notifications')
  await page.getByRole('button', { name: '重新入队', exact: true }).click()
  await page.getByLabel('操作原因').fill('检查任务状态')
  await page.getByRole('button', { name: '确认', exact: true }).click()
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(page.getByRole('alert')).toContainText('状态已变化')
  expect(writes).toBe(1)
  await expect(page.getByRole('table')).toContainText('9')
})

test('developer test membership, history and usage preserve revision and missing-day semantics', async ({ page }) => {
  await mockPortal(page)
  let revision = 3
  let writes = 0
  await page.route('**/api/v1/developer/applications**', route => {
    const path = new URL(route.request().url()).pathname
    if (path.endsWith('/test-members')) { if (route.request().method() === 'POST') { expect(route.request().postDataJSON()).toMatchObject({ revision: 3 }); revision++; writes++ } return route.fulfill({ json: { revision, limit: 20, items: writes ? [{ user_id: '22222222-2222-4222-8222-222222222222', created_at: application.created_at }] : [] } }) }
    if (path.endsWith('/versions')) return route.fulfill({ json: { items: [{ ...application.latest, created_at: application.created_at }], active_version: 0, latest_version: 1, next_before: 0 } })
    if (path.endsWith('/statistics')) return route.fulfill({ json: { timezone: 'UTC', generated_at: application.created_at, recording_since: application.created_at, window_start: '2026-09-23', window_end: '2026-10-23', totals: { authorizations: 1, code_exchanges: 0, refreshes: 0, machine_tokens: 0 }, days: [{ day: '2026-10-20', partial: false }, { day: '2026-10-21', partial: true, counts: { authorizations: 1, code_exchanges: 0, refreshes: 0, machine_tokens: 0 } }] } })
    if (path.endsWith('/applications')) return route.fulfill({ json: { items: [{ ...application, revision }], next_cursor: '' } })
    return route.fulfill({ json: { ...application, revision } })
  })
  await page.goto(account + `/developer/applications/${application.id}`)
  await page.getByLabel('用户 UUID').fill('22222222-2222-4222-8222-222222222222')
  await page.getByRole('button', { name: '添加', exact: true }).click()
  await expect(page.getByRole('table')).toContainText('22222222-2222-4222-8222-222222222222')
  expect(writes).toBe(1)
  await page.getByRole('link', { name: '用量', exact: true }).click()
  await expect(page.getByRole('figure')).toBeVisible()
  await page.locator('summary').click()
  await expect(page.getByRole('table')).toContainText('未开始记录')
})

test('administrator pagination omits empty cursors and retains nonempty cursors', async ({ page }) => {
  await adminSetup(page)
  for (const [section, endpoint, key] of [['audit', 'audit-events', 'cursor'], ['principals', 'principals', 'after']] as const) {
    const queries: string[] = []
    await page.route(`**/api/v1/admin/${endpoint}*`, route => {
      const url = new URL(route.request().url())
      queries.push(url.search)
      return route.fulfill({ json: { items: [], [key === 'after' ? 'next_after' : 'next_cursor']: url.searchParams.has(key) ? '' : 'example-next-page' } })
    })
    await page.goto(admin + '/' + section)
    await expect(page.getByRole('button', { name: '下一页', exact: true })).toBeEnabled()
    await page.getByRole('button', { name: '下一页', exact: true }).click()
    await expect(page.getByRole('button', { name: '上一页', exact: true })).toBeEnabled()
    await page.getByRole('button', { name: '上一页', exact: true }).click()
    await expect(page.getByRole('button', { name: '下一页', exact: true })).toBeEnabled()
    expect(queries).toEqual(['', `?${key}=example-next-page`, ''])
  }
})

test('usage can retry its first application load and automatically loads statistics', async ({ page }) => {
  await mockPortal(page)
  let attempts = 0
  await page.route('**/api/v1/developer/applications**', route => {
    const url = new URL(route.request().url())
    if (url.pathname.endsWith('/statistics')) return route.fulfill({ json: { generated_at: application.created_at, days: [] } })
    expect(url.search).toBe('')
    attempts++
    return attempts === 1 ? route.fulfill({ status: 503, json: {} }) : route.fulfill({ json: { items: [application], next_cursor: '' } })
  })
  await page.goto(account + '/developer/usage')
  await expect(page.getByRole('alert')).toBeVisible()
  const retry = page.getByRole('button', { name: '刷新', exact: true })
  await expect(retry).toBeEnabled()
  await retry.click()
  await expect(page.getByRole('figure')).toBeVisible()
  await expect(page.getByRole('alert')).toHaveCount(0)
  expect(attempts).toBe(2)
})

test('central sign-out requires an explicit decision and re-confirms a changed account', async ({ page }) => {
  const request = 'a'.repeat(43)
  let decisions = 0
  await page.route('**/api/v1/status', route => route.fulfill({ json: { service: 'moetopia' } }))
  await page.route('**/api/v1/account/context', route => route.fulfill({ json: { csrf_token: 'example-identity-csrf', registration_mode: 'invite', user: previewUser } }))
  await page.route(`**/api/v1/account/logout-requests/${request}`, route => {
    if (route.request().method() !== 'GET') { decisions++; expect(route.request().postDataJSON()).toEqual({ decision: 'approve', subject_id: previewUser.id }); return route.fulfill({ status: 409, json: { code: 'logout_account_changed' } }) }
    return route.fulfill({ json: { client_id: 'example-client', client_name: '交接测试应用', post_logout_redirect_uri: 'https://example.com/logout', expires_at: '2030-01-01T00:00:00Z', requires_confirmation: true, user: previewUser } })
  })
  await page.goto(account + `/auth/confirm-logout?logout_request=${request}`)
  await expect(page.getByRole('button', { name: '确认', exact: true })).toBeEnabled()
  expect(decisions).toBe(0)
  await page.getByRole('button', { name: '确认', exact: true }).click()
  await expect(page.getByText('状态已变化，已刷新，请检查后重新操作。')).toBeVisible()
  expect(decisions).toBe(1)
})

test('passwordless accounts use recovery instead of collecting unavailable credentials', async ({ page }) => {
  await mockPortal(page)
  await page.route('**/api/v1/portal/context', route => route.fulfill({ json: { authenticated: true, csrf_token: 'portal-csrf', login_url: '/api/v1/portal/login', user: { ...previewUser, has_password: false } } }))
  await page.goto(account + '/profile')
  await expect(page.getByText(/此账号尚未设置密码/)).toBeVisible()
  await expect(page.getByLabel('当前密码', { exact: true })).toHaveCount(0)
  await expect(page.getByRole('button', { name: '注销账号', exact: true })).toBeDisabled()
})

test('federation registration verifies email and never requests a local password', async ({ page }) => {
  let completionBody: Record<string, unknown> | undefined
  await page.route('**/api/v1/status', route => route.fulfill({ json: { service: 'moetopia' } }))
  await page.route('**/api/v1/account/context', route => route.fulfill({ json: { csrf_token: 'example-account-csrf', registration_mode: 'invite' } }))
  await page.route('**/api/v1/account/verification', route => route.fulfill({ json: { challenge_id: 'example-challenge' } }))
  await page.route('**/api/v1/account/federation/registration**', route => {
    if (route.request().method() === 'POST') { completionBody = route.request().postDataJSON(); return route.fulfill({ json: { redirect_to: '/auth/login' } }) }
    return route.fulfill({ json: { provider: 'github', display_name: '示例开发者', email_hint: 'example@example.com', expires_at: '2030-01-01T00:00:00Z' } })
  })
  await page.route('**/api/v1/account/federation/providers', route => route.fulfill({ json: { items: [] } }))
  await page.goto(account + '/auth/federation?completion=example-completion')
  await expect(page.getByLabel('密码', { exact: true })).toHaveCount(0)
  await page.getByLabel('邀请码', { exact: true }).fill('EXAMPLE-ONLY-INVITE')
  await page.getByRole('button', { name: '发送验证码', exact: true }).click()
  await page.getByLabel('邮箱验证码', { exact: true }).fill('12345678')
  await page.getByRole('button', { name: '验证并创建账号', exact: true }).click()
  await expect(page).toHaveURL(account + '/auth/login')
  expect(completionBody).toMatchObject({ completion: 'example-completion', challenge_id: 'example-challenge', code: '12345678', display_name: '示例开发者', invitation: 'EXAMPLE-ONLY-INVITE' })
  expect(completionBody).not.toHaveProperty('password')
})

test('account deletion reports enabled applications and cancellation remains anonymous', async ({ page }) => {
  await mockPortal(page)
  let writes = 0
  await page.route('**/api/v1/portal/deletion', route => { writes++; expect(route.request().headers()['x-moetopia-csrf']).toBe('portal-csrf'); return route.fulfill({ status: 409, json: { code: 'account_published_applications' } }) })
  await page.route('**/api/v1/account/context', route => route.fulfill({ json: { csrf_token: 'example-account-csrf', registration_mode: 'invite' } }))
  await page.route('**/api/v1/account/deletion-cancellations', route => route.fulfill({ json: { challenge_id: 'example-challenge' } }))
  await page.route('**/api/v1/account/deletion-cancellations/confirm', route => { expect(route.request().headers()['x-moetopia-csrf']).toBe('example-account-csrf'); expect(route.request().postDataJSON()).toEqual({ challenge_id: 'example-challenge', code: '12345678', current_password: 'example-only-password' }); return route.fulfill({ status: 204 }) })
  await page.goto(account + '/profile')
  await page.getByRole('button', { name: '注销账号', exact: true }).click()
  const dialog = page.getByRole('dialog')
  await dialog.getByLabel('当前密码', { exact: true }).fill('example-only-password')
  await dialog.getByRole('button', { name: '注销账号', exact: true }).click()
  await expect(dialog.getByRole('alert')).toContainText('请先停用所有已发布且启用的应用')
  expect(writes).toBe(1)
  await page.goto(account + '/auth/cancel-deletion')
  await page.getByLabel('邮箱', { exact: true }).fill('example@example.com')
  await page.getByRole('button', { name: '发送验证码', exact: true }).click()
  await page.getByLabel('邮箱验证码', { exact: true }).fill('12345678')
  await page.getByLabel('当前密码', { exact: true }).fill('example-only-password')
  await page.getByRole('button', { name: '取消注销', exact: true }).click()
  await expect(page.getByText(/注销已取消，请重新登录/)).toBeVisible()
  await expect(page).toHaveURL(account + '/auth/cancel-deletion')
})

for (const locale of ['zh-CN', 'zh-TW', 'en', 'ja']) {
  for (const width of [1385, 375]) {
    test(`administrator pages remain usable at ${width}px in ${locale}`, async ({ page }) => {
      const errors: string[] = []
      page.on('pageerror', error => errors.push(error.message))
      await adminSetup(page)
      await page.context().addCookies([{ name: 'moetopia_locale', value: locale, url: admin }, { name: 'moetopia_theme', value: 'dark', url: admin }])
      await page.setViewportSize({ width, height: 900 })
      for (const path of ['/', '/users', `/applications/${application.id}`]) {
        await page.goto(admin + path)
        await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
        if (path === '/') await expect(page.getByRole('figure')).toBeVisible()
        else if (path === '/users') await expect(page.getByRole('table')).toContainText(previewUser.email)
        else await expect(page.getByRole('heading', { name: application.latest.config.name, exact: true })).toBeVisible()
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
        if (path.startsWith('/applications/')) {
          const summary = await page.locator('.moe-resource-summary').boundingBox()
          for (const button of await page.locator('.moe-resource-summary button').all()) {
            const bounds = await button.boundingBox()
            expect(bounds && summary && bounds.x >= summary.x && bounds.x + bounds.width <= summary.x + summary.width).toBe(true)
          }
        }
        await page.screenshot({ path: `.data/previews/handoff/admin-${locale}-${width}-${path === '/' ? 'overview' : path === '/users' ? 'users' : 'application'}.png`, fullPage: true })
      }
      expect(errors).toEqual([])
    })
  }
}
