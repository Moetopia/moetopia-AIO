import tailwindcss from '@tailwindcss/vite'

const apiUpstream = process.env.MOETOPIA_API_UPSTREAM ?? 'http://127.0.0.1:8080'
const apiURL = new URL(apiUpstream)
if (!['http:', 'https:'].includes(apiURL.protocol) || apiURL.username || apiURL.password || apiURL.pathname !== '/' || apiURL.search || apiURL.hash) {
  throw new Error('MOETOPIA_API_UPSTREAM must be a fixed server origin')
}

export default defineNuxtConfig({
  compatibilityDate: '2026-10-18',
  devtools: { enabled: false },
  app: {
    pageTransition: { name: 'moe-page', mode: 'out-in' },
    layoutTransition: { name: 'moe-layout', mode: 'out-in' },
  },
  css: ['@moetopia/ui/styles.css'],
  typescript: { strict: true },
  build: { transpile: ['@moetopia/ui', '@moetopia/locales'] },
  runtimeConfig: {
    public: {
      accountUrl: process.env.NODE_ENV === 'development' ? 'http://account.localhost:3000' : '',
      adminUrl: process.env.NODE_ENV === 'development' ? 'http://admin.localhost:3001' : '',
    },
  },
  vite: { plugins: [tailwindcss()] },
  routeRules: {
    '/.well-known/openid-configuration': { proxy: `${apiURL.origin}/.well-known/openid-configuration`, headers: { 'cache-control': 'no-store' } },
    '/api/**': { proxy: { to: `${apiURL.origin}/api/**`, fetchOptions: { redirect: 'manual' } }, headers: { 'cache-control': 'no-store' } },
    '/oauth/**': { proxy: { to: `${apiURL.origin}/oauth/**`, fetchOptions: { redirect: 'manual' } }, headers: { 'cache-control': 'no-store' } },
  },
})
