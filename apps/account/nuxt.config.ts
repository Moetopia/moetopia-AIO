const privatePageHeaders = {
  'cache-control': 'no-store',
  'referrer-policy': 'no-referrer',
  'content-security-policy': "frame-ancestors 'none'",
  'x-frame-options': 'DENY',
}

export default defineNuxtConfig({
  extends: ['../../packages/portal'],
  ssr: true,
  routeRules: {
    '/': { headers: privatePageHeaders },
    '/sessions': { headers: privatePageHeaders },
    '/security': { headers: privatePageHeaders },
    '/profile': { headers: privatePageHeaders },
    '/auth/**': { headers: privatePageHeaders },
    '/developer/**': { headers: privatePageHeaders },
    '/developer': { headers: privatePageHeaders },
  },
})
