import { expect, test, type Page } from '@playwright/test'
import { selectLanguage } from './helpers/language'

const origin = 'http://127.0.0.1:53000'
const anonymous = { csrf_token: 'example-csrf', registration_mode: 'invite' }
const prepare = async (page: Page, signedIn = false) => {
  await page.route('**/api/v1/status', route => route.fulfill({ json: { service: 'moetopia' } }))
  await page.route('**/api/v1/account/context', route => route.fulfill({ json: { ...anonymous, ...(signedIn ? { user: { id: 'example', display_name: '预览用户', email: 'preview@example.com' } } : {}) } }))
  await page.route('**/api/v1/account/sessions', route => route.fulfill({ json: { current_id: 'example', items: [] } }))
}

test('language menu offers four native names, preserves input and persists Japanese on reload', async ({ page }) => {
  await prepare(page)
  await page.goto(`${origin}/auth/register`)
  await page.locator('#account-email').fill('preview@example.com')
  await page.locator('#account-password').fill('example passphrase only')
  await page.getByRole('combobox', { name: '语言', exact: true }).click()
  for (const name of ['简体中文', '繁體中文', 'English', '日本語']) await expect(page.getByRole('option', { name, exact: true })).toBeVisible()
  await page.getByRole('option', { name: '日本語', exact: true }).click()
  await expect(page.locator('html')).toHaveAttribute('lang', 'ja')
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('アカウントを作成')
  await expect(page.locator('#account-email')).toHaveValue('preview@example.com')
  await expect(page.locator('#account-password')).toHaveValue('example passphrase only')
  await page.reload()
  await expect(page.locator('html')).toHaveAttribute('lang', 'ja')
  await expect(page.getByRole('combobox', { name: '言語', exact: true })).toContainText('日本語')
})

for (const [locale, name, emailLanguage] of [['zh-TW', '繁體中文', 'zh-CN'], ['ja', '日本語', 'en']] as const) {
  test(`${locale} uses a complete frontend translation and a supported verification email locale`, async ({ page }) => {
    await prepare(page)
    await page.route('**/api/v1/account/verification', route => {
      expect(route.request().postDataJSON()).toEqual({ email: 'preview@example.com', locale: emailLanguage })
      return route.fulfill({ json: { challenge_id: 'example-challenge' } })
    })
    await page.goto(`${origin}/auth/register`)
    await selectLanguage(page, name)
    await page.locator('#account-email').fill('preview@example.com')
    await page.getByRole('button', { name: locale === 'ja' ? 'コードを送信' : '傳送驗證碼', exact: true }).click()
    await expect(page.locator('#account-code')).toBeEnabled()
    await expect(page.locator('html')).toHaveAttribute('lang', locale)
    expect(await page.locator('main').innerText()).not.toContain('undefined')
  })
}

test('unsupported locale cookie falls back consistently on SSR and client', async ({ page }) => {
  await prepare(page)
  await page.context().addCookies([{ name: 'moetopia_locale', value: 'unsupported', url: origin }])
  const response = await page.goto(`${origin}/auth/login`)
  expect(await response?.text()).toContain('lang="zh-CN"')
  await expect(page.getByRole('combobox', { name: '语言', exact: true })).toContainText('简中')
})

test('slow route loads show progress and page transitions without resetting the account context', async ({ page }) => {
  await prepare(page, true)
  await page.goto(`${origin}/developer/docs`)
  await expect(page.getByRole('combobox', { name: '语言', exact: true })).toBeEnabled()
  await page.getByRole('button', { name: '你的账号', exact: true }).click()
  await page.evaluate(() => {
    document.documentElement.dataset.transitions = '0'
    document.addEventListener('transitionrun', event => {
      if (event.target instanceof Element && [...event.target.classList].some(name => /moe-(page|layout)-(enter|leave)-active/.test(name))) {
        document.documentElement.dataset.transitions = String(Number(document.documentElement.dataset.transitions) + 1)
      }
    })
  })
  let release!: () => void
  const gate = new Promise<void>(resolve => { release = resolve })
  await page.route('**/_nuxt/**', async route => {
    if (route.request().resourceType() === 'script') await gate
    await route.continue()
  })
  try {
    await page.getByRole('menuitem', { name: '个人资料', exact: true }).click()
    await expect(page.getByRole('progressbar', { name: '正在加载页面', exact: true })).toBeVisible()
  } finally { release() }
  await expect(page).toHaveURL(`${origin}/profile`)
  await expect(page.getByRole('heading', { name: '个人资料', exact: true })).toBeVisible()
  await expect(page.getByRole('progressbar')).toHaveCount(0)
  expect(await page.evaluate(() => Number(document.documentElement.dataset.transitions))).toBeGreaterThan(0)
})

test('reduced motion removes route movement while keeping the page usable', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' })
  await prepare(page)
  await page.goto(`${origin}/auth/login`)
  const style = await page.evaluate(() => {
    const probe = document.createElement('div')
    probe.className = 'moe-page-enter-active moe-page-enter-from'
    document.body.append(probe)
    const result = { transform: getComputedStyle(probe).transform, duration: getComputedStyle(probe).transitionDuration }
    probe.remove()
    return result
  })
  expect(style.transform).toBe('none')
  expect(style.duration.split(',').every(value => parseFloat(value) <= .001)).toBe(true)
  await page.getByRole('link', { name: '立即注册', exact: true }).click()
  await expect(page.getByRole('heading', { level: 1 })).toHaveText('创建账号')
})

test('administration uses a full-height rail and its collapse controls translate', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 })
  await page.route('**/api/**', route => route.fulfill({ status: 503, json: {} }))
  await page.goto('http://127.0.0.1:53001')
  const rail = page.locator('.moe-page-rail')
  await expect(rail).toBeVisible()
  expect((await rail.boundingBox())?.y).toBe(0)
  expect((await rail.boundingBox())?.height).toBe(900)
  await selectLanguage(page, 'English')
  await page.getByRole('button', { name: 'Collapse sidebar', exact: true }).click()
  await expect(page.locator('.moe-sidebar')).toHaveAttribute('data-collapsed', 'true')
  await expect(page.getByRole('button', { name: 'Expand sidebar', exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Expand sidebar', exact: true }).click()
  const first = await page.locator('.moe-page-rail .moe-nav-item').nth(0).boundingBox()
  const second = await page.locator('.moe-page-rail .moe-nav-item').nth(1).boundingBox()
  expect(first && second && second.y - first.y < 64).toBeTruthy()
  await page.setViewportSize({ width: 820, height: 900 })
  await expect(page.getByRole('button', { name: 'Open navigation', exact: true })).toBeHidden()
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
})
