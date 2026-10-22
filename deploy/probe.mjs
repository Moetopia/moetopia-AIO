import http from 'node:http'
import https from 'node:https'
import { performance } from 'node:perf_hooks'

const args = process.argv.slice(2)
const base = new URL(args[0] ?? '')
const requests = Number(args[1] ?? 100)
const concurrency = Number(args[2] ?? 4)
const insecure = args.includes('--insecure-test-tls')
const connect = args.find(value => value.startsWith('--connect-host='))?.split('=')[1] ?? base.hostname
if (!['http:', 'https:'].includes(base.protocol) || base.username || base.password || base.pathname !== '/' || base.search || base.hash || !Number.isInteger(requests) || requests < 1 || requests > 1000 || !Number.isInteger(concurrency) || concurrency < 1 || concurrency > 16) throw new Error('Use an origin, 1-1000 requests and 1-16 workers')
if (insecure && !base.hostname.endsWith('.test') && !['localhost', '127.0.0.1', '[::1]'].includes(base.hostname)) throw new Error('Unverified TLS is only permitted for explicit test origins')
const paths = ['/api/v1/status', '/api/v1/account/context', '/.well-known/openid-configuration']
const samples = []
const statuses = {}
let index = 0
const started = performance.now()
await Promise.all(Array.from({ length: concurrency }, async () => {
  while (index < requests) {
    const current = index++
    const begin = performance.now()
    const status = await new Promise(resolve => {
      const transport = base.protocol === 'https:' ? https : http
      const request = transport.get({ hostname: connect, port: base.port || (base.protocol === 'https:' ? 443 : 80), servername: base.hostname, path: paths[current % paths.length], headers: { host: base.host }, rejectUnauthorized: !insecure, timeout: 10000 }, response => {
        response.resume()
        response.on('end', () => resolve(response.statusCode))
      })
      request.on('timeout', () => request.destroy())
      request.on('error', () => resolve('network_error'))
    })
    samples.push(performance.now() - begin)
    statuses[status] = (statuses[status] ?? 0) + 1
  }
}))
samples.sort((a, b) => a - b)
const percentile = value => Number(samples[Math.min(samples.length - 1, Math.ceil(samples.length * value) - 1)].toFixed(2))
console.log(JSON.stringify({ scenario: 'anonymous_read_smoke', requests, concurrency, duration_ms: Number((performance.now() - started).toFixed(2)), p50_ms: percentile(0.5), p95_ms: percentile(0.95), p99_ms: percentile(0.99), statuses, verified_tls: !insecure }, null, 2))
if (Object.keys(statuses).some(status => status !== '200')) process.exitCode = 1
