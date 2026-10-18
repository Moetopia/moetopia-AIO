import { randomBytes } from 'node:crypto'
import { spawnSync } from 'node:child_process'
import { mkdirSync, rmSync } from 'node:fs'
import { resolve } from 'node:path'

if (process.env.MOETOPIA_ENV !== 'development') throw new Error('Account browser tests require the development environment')
let dsn
try { dsn = new URL(process.env.MOETOPIA_DATABASE_URL ?? '') } catch { throw new Error('Invalid development database configuration') }
if (dsn.pathname !== '/moetopia_dev' || dsn.port !== '55432') throw new Error('Refusing to provision outside the dedicated development database')
const name = `moetopia_test_${randomBytes(8).toString('hex')}`
const binary = resolve('.data/bin', `account-test-${process.pid}${process.platform === 'win32' ? '.exe' : ''}`)
mkdirSync(resolve('.data/bin'), { recursive: true })
const environment = { ...process.env, MOETOPIA_TEST_DATABASE_NAME: name }
const execute = (command, args, env = environment, capture = false) => {
  const result = spawnSync(command, args, { env, stdio: capture ? ['ignore', 'pipe', 'inherit'] : 'inherit', encoding: 'utf8', windowsHide: true })
  if (result.error || result.status !== 0) throw new Error('Account browser test command failed; configuration values were not displayed')
  return result.stdout?.trim()
}
execute('go', ['-C', 'apps/api', 'run', './cmd/testdb', 'create'])
try {
  dsn.pathname = `/${name}`
  const origin = 'http://127.0.0.1:4305'
  const env = { ...environment, MOETOPIA_DATABASE_URL: dsn.href, MOETOPIA_LISTEN: '127.0.0.1:58080', MOETOPIA_ISSUER: origin, MOETOPIA_ACCOUNT_ORIGIN: origin, MOETOPIA_ENCRYPTION_KEY: randomBytes(32).toString('base64'), MOETOPIA_TEST_API_BINARY: binary }
  execute('go', ['-C', 'apps/api', 'run', './cmd/migrate'], env)
  execute('go', ['-C', 'apps/api', 'run', './cmd/keys', 'init'], env)
  env.MOETOPIA_TEST_INVITATION = execute('go', ['-C', 'apps/api', 'run', './cmd/devinvite'], env, true)
  execute('go', ['-C', 'apps/api', 'build', '-o', binary, './cmd/api'], env)
  const args = ['exec', 'playwright', 'test', '--config', 'playwright.account.config.ts']
  const pnpm = process.env.npm_execpath
  if (!pnpm) throw new Error('Run this task using pnpm test:account')
  if (/\.[cm]?js$/.test(pnpm)) execute(process.execPath, [pnpm, ...args], env)
  else execute(pnpm, args, env)
} finally {
  execute('go', ['-C', 'apps/api', 'run', './cmd/testdb', 'drop'])
  rmSync(binary, { force: true })
}
