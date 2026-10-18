import tailwindcss from '@tailwindcss/vite'

export default defineNuxtConfig({
  compatibilityDate: '2026-10-18',
  devtools: { enabled: false },
  css: ['@moetopia/ui/styles.css'],
  typescript: { strict: true },
  build: { transpile: ['@moetopia/ui', '@moetopia/locales'] },
  runtimeConfig: {
    public: {
      accountUrl: process.env.NODE_ENV === 'development' ? 'http://account.localhost:3000' : '',
      adminUrl: process.env.NODE_ENV === 'development' ? 'http://admin.localhost:3001' : '',
      developerUrl: process.env.NODE_ENV === 'development' ? 'http://developer.localhost:3002' : '',
    },
  },
  vite: { plugins: [tailwindcss()] },
  routeRules: {
    '/.well-known/openid-configuration': { proxy: 'http://127.0.0.1:8080/.well-known/openid-configuration', headers: { 'cache-control': 'no-store' } },
    '/api/**': { proxy: 'http://127.0.0.1:8080/api/**', headers: { 'cache-control': 'no-store' } },
    '/oauth/**': { proxy: 'http://127.0.0.1:8080/oauth/**', headers: { 'cache-control': 'no-store' } },
  },
})
