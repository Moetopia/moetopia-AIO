import { createRequire } from 'node:module'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'

const directory = process.env.MOETOPIA_TEST_OIDC_DIRECTORY
const base = resolve('.data')
if (!directory || !resolve(directory).startsWith(base + (process.platform === 'win32' ? '\\' : '/'))) throw new Error('An owned OIDC test output directory is required')
const require = createRequire(resolve('apps/account/package.json'))
const { loadNuxt, buildNuxt } = await import(pathToFileURL(require.resolve('nuxt/kit')).href)
const rules = {
  '/.well-known/openid-configuration': { proxy: 'http://127.0.0.1:58490/.well-known/openid-configuration', headers: { 'cache-control': 'no-store' } },
  '/api/**': { proxy: 'http://127.0.0.1:58490/api/**', headers: { 'cache-control': 'no-store' } },
  '/oauth/**': { proxy: 'http://127.0.0.1:58490/oauth/**', headers: { 'cache-control': 'no-store' } },
}
const nuxt = await loadNuxt({ cwd: resolve('apps/account'), dev: false, overrides: {
  buildDir: resolve(directory, 'nuxt'),
  nitro: { output: { dir: resolve(directory, 'output'), serverDir: resolve(directory, 'output/server'), publicDir: resolve(directory, 'output/public') } },
  hooks: {
    'nitro:config': (config) => {
      for (const [path, rule] of Object.entries(rules)) {
        if (config.routeRules?.[path]?.proxy !== rule.proxy.replace(':58490/', ':8080/')) throw new Error(`Account proxy route changed: ${path}; review the test upstream override`)
        config.routeRules[path] = { ...config.routeRules[path], ...rule }
      }
    },
  },
} })
try {
  await buildNuxt(nuxt)
} finally { await nuxt.close() }
