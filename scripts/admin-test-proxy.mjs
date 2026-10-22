import { createServer } from 'node:http'
import { oidcAPIOrigin } from './oidc-test-config.mjs'
import { readFile } from 'node:fs/promises'
import { resolve, extname, sep } from 'node:path'

const publicDirectory = resolve('apps/admin/.output/public')
const contentTypes = { '.html': 'text/html; charset=utf-8', '.js': 'text/javascript', '.css': 'text/css', '.json': 'application/json', '.svg': 'image/svg+xml', '.png': 'image/png', '.woff2': 'font/woff2' }

if (process.env.MOETOPIA_ENV !== 'test' || process.env.MOETOPIA_TEST_OIDC_SUITE !== 'admin' || process.env.MOETOPIA_ADMIN_ORIGIN !== 'http://admin.localhost:54311') throw new Error('The isolated administrator browser environment is required')
const server = createServer(async (req, res) => {
  res.setHeader('cache-control', 'no-store')
  res.setHeader('referrer-policy', 'no-referrer')
  try {
    const url = new URL(req.url, 'http://admin.localhost:54311')
    if (url.pathname === '/healthz') { res.end('ready'); return }
    if (!url.pathname.startsWith('/api/v1/admin/') && url.pathname !== '/api/v1/status') {
      const requested = resolve(publicDirectory, `.${decodeURIComponent(url.pathname)}`)
      if (requested !== publicDirectory && !requested.startsWith(publicDirectory + sep)) { res.writeHead(403); res.end(); return }
      const file = extname(requested) ? requested : resolve(publicDirectory, 'index.html')
      const content = await readFile(file)
      res.setHeader('content-type', contentTypes[extname(file)] ?? 'application/octet-stream')
      res.setHeader('x-content-type-options', 'nosniff')
      res.end(content)
      return
    }
    let body = Buffer.alloc(0)
    for await (const chunk of req) {
      body = Buffer.concat([body, chunk])
      if (body.length > 65536) { res.writeHead(413); res.end(); return }
    }
    const headers = {}
    for (const name of ['cookie', 'origin', 'content-type', 'x-moetopia-csrf']) if (req.headers[name]) headers[name] = req.headers[name]
    const upstream = await fetch(oidcAPIOrigin() + url.pathname + url.search, { method: req.method, headers, body: ['GET', 'HEAD'].includes(req.method) ? undefined : body, redirect: 'manual', signal: AbortSignal.timeout(15000) })
    for (const name of ['content-type', 'location', 'retry-after']) if (upstream.headers.has(name)) res.setHeader(name, upstream.headers.get(name))
    const cookies = upstream.headers.getSetCookie()
    if (cookies.length) res.setHeader('set-cookie', cookies)
    res.writeHead(upstream.status)
    res.end(Buffer.from(await upstream.arrayBuffer()))
  } catch {
    if (!res.headersSent) res.writeHead(503)
    res.end('Administrator probe unavailable')
  }
})
server.listen(54311, '127.0.0.1')
for (const signal of ['SIGINT', 'SIGTERM']) process.once(signal, () => server.close(() => process.exit(0)))
