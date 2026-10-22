import type { Page } from '@playwright/test'
export const previewUser = { id: 'preview-user', display_name: '预览用户', email: 'preview@example.com', bio: '', revision: 1, has_password: true }
export const previewSession = { id: 'current', created_at: '2026-10-18T02:00:00Z', expires_at: '2030-11-18T02:00:00Z' }
export const mockPortal = async (page: Page, authenticated = true) => {
  const state = { authenticated, reads: 0, writes: [] as string[], items: [previewSession, { ...previewSession, id: 'other' }] }
  const context = () => ({ authenticated: state.authenticated, csrf_token: 'portal-csrf', login_url: '/api/v1/portal/login', ...(state.authenticated ? { user: previewUser, expires_at: '2030-11-18T02:00:00Z' } : {}) })
  await page.route('**/api/v1/status', route => route.fulfill({ json: { service: 'moetopia' } }))
  await page.route('**/api/v1/developer/applications', route => route.fulfill({ json: { items: [], next_cursor: '' } }))
  await page.route('**/api/v1/portal/authorizations', route => route.fulfill({ json: { items: [], next_cursor: '' } }))
  await page.route('**/api/v1/portal/security-events', route => route.fulfill({ json: { items: [] } }))
  await page.route('**/api/v1/account/federation/providers', route => route.fulfill({ json: { items: [] } }))
  await page.route('**/api/v1/portal/identities', route => route.fulfill({ json: { items: [] } }))
  await page.route('**/api/v1/portal/context', route => { state.reads++; return route.fulfill({ json: context() }) })
  await page.route('**/api/v1/portal/sessions', route => route.fulfill({ json: { current_id: 'current', items: state.items } }))
  await page.route('**/api/v1/portal/sessions/*', route => {
    state.writes.push(route.request().headers()['x-moetopia-csrf'] ?? '')
    const id = route.request().url().split('/').at(-1)
    state.items = state.items.filter(item => item.id !== id)
    if (id === 'current') state.authenticated = false
    return route.fulfill({ status: 204 })
  })
  await page.route('**/api/v1/portal/logout', route => { state.writes.push(route.request().headers()['x-moetopia-csrf'] ?? ''); state.authenticated = false; return route.fulfill({ json: context() }) })
  return state
}
