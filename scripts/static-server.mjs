import { createServer } from 'node:http'
import { readFile } from 'node:fs/promises'
import { resolve, extname, sep } from 'node:path'

const directory = resolve(process.argv[2])
const port = Number(process.argv[3])
if (!Number.isInteger(port) || port < 1024 || port > 65535) throw new Error('Invalid local preview port')
const types = { '.html': 'text/html', '.js': 'text/javascript', '.mjs': 'text/javascript', '.css': 'text/css', '.json': 'application/json', '.svg': 'image/svg+xml', '.png': 'image/png', '.woff2': 'font/woff2' }
createServer(async (request, response) => {
  const pathname = new URL(request.url, 'http://localhost').pathname
  if (pathname.startsWith('/api/') || pathname.startsWith('/oauth/')) { response.writeHead(503); response.end(); return }
  let file
  try { file = resolve(directory, `.${decodeURIComponent(pathname === '/' ? '/index.html' : pathname)}`) } catch { response.writeHead(400); response.end(); return }
  if (!file.startsWith(directory + sep)) { response.writeHead(403); response.end(); return }
  try {
    const content = await readFile(file)
    response.writeHead(200, { 'Content-Type': types[extname(file)] ?? 'application/octet-stream', 'X-Content-Type-Options': 'nosniff' })
    response.end(content)
  } catch {
    if (process.argv.includes('--spa') && !extname(file)) {
      try {
        const content = await readFile(resolve(directory, 'index.html'))
        response.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8', 'X-Content-Type-Options': 'nosniff' })
        response.end(content)
        return
      } catch { response.writeHead(404); response.end(); return }
    }
    response.writeHead(404); response.end()
  }
}).listen(port, '127.0.0.1')
