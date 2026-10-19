import { randomBytes, randomUUID } from 'node:crypto'
import { spawnSync } from 'node:child_process'
import { mkdirSync, rmSync } from 'node:fs'
import { resolve, sep } from 'node:path'
import { checkPort } from './dev-config.mjs'

if (process.env.MOETOPIA_ENV !== 'development') throw new Error('OIDC tests require the explicit development dependency environment')
let dsn
try { dsn = new URL(process.env.MOETOPIA_DATABASE_URL ?? '') } catch { throw new Error('Invalid development database configuration') }
if (dsn.pathname !== '/moetopia_dev' || dsn.port !== '55432') throw new Error('Refusing to provision outside the dedicated development database')
for (const port of [54305, 54310, 58490]) await checkPort('OIDC integration', port)
const name = `moetopia_test_${randomBytes(8).toString('hex')}`
const directory = resolve('.data', `oidc-${process.pid}-${randomBytes(4).toString('hex')}`)
const binary = resolve(directory, `api${process.platform === 'win32' ? '.exe' : ''}`)
mkdirSync(directory, { recursive: true })
const provision = { ...process.env, MOETOPIA_TEST_DATABASE_NAME: name }
const execute = (command, args, env, capture = false) => {
  const result = spawnSync(command, args, { env, encoding: 'utf8', stdio: capture ? ['ignore', 'pipe', 'inherit'] : 'inherit', windowsHide: true })
  if (result.error || result.status !== 0) throw new Error('OIDC test command failed; no configuration or credential values were displayed')
  return result.stdout?.trim()
}
let created = false
try {
  execute('go', ['-C', 'apps/api', 'run', './cmd/testdb', 'create'], provision)
  created = true
  dsn.pathname = `/${name}`
  const issuer = 'http://127.0.0.1:54305'
  const env = { ...provision, MOETOPIA_ENV: 'test', MOETOPIA_DATABASE_URL: dsn.href, MOETOPIA_LISTEN: '127.0.0.1:58490', MOETOPIA_ISSUER: issuer, MOETOPIA_ACCOUNT_ORIGIN: issuer, MOETOPIA_ENCRYPTION_KEY: randomBytes(32).toString('base64'), MOETOPIA_TEST_API_BINARY: binary, MOETOPIA_TEST_OIDC_DIRECTORY: directory, MOETOPIA_TEST_OIDC_CALLBACK: 'http://localhost:54310/callback', MOETOPIA_TEST_OIDC_EMAIL: `oidc-${randomUUID()}@moetopia.test`, MOETOPIA_TEST_OIDC_PASSWORD: randomBytes(24).toString('base64url') }
  execute('go', ['-C', 'apps/api', 'run', './cmd/migrate'], env)
  execute('go', ['-C', 'apps/api', 'run', './cmd/keys', 'init'], env)
  execute('go', ['-C', 'apps/api', 'run', './cmd/firstparty'], env)
  const fixture = JSON.parse(execute('go', ['-C', 'apps/api', 'run', './cmd/testoidc'], env, true))
  env.MOETOPIA_TEST_OIDC_CLIENT = fixture.client_id
  env.MOETOPIA_TEST_INVITATION = fixture.invitation
  execute('go', ['-C', 'apps/api', 'build', '-o', binary, './cmd/api'], env)
  execute(process.execPath, ['scripts/build-oidc-account.mjs'], env)
  const pnpm = process.env.npm_execpath
  if (!pnpm) throw new Error('Run with pnpm test:oidc')
  const args = ['exec', 'playwright', 'test', '--config', 'playwright.oidc.config.ts']
  if (/\.[cm]?js$/.test(pnpm)) execute(process.execPath, [pnpm, ...args], env)
  else execute(pnpm, args, env)
} finally {
  try {
    if (created) execute('go', ['-C', 'apps/api', 'run', './cmd/testdb', 'drop'], provision)
  } finally {
    if (!directory.startsWith(resolve('.data') + sep)) throw new Error('Refusing to remove an unowned test directory')
    rmSync(directory, { recursive: true, force: true })
  }
}
