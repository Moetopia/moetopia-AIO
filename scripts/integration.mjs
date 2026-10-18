import { randomBytes } from 'node:crypto'
import { spawnSync } from 'node:child_process'

const name = `moetopia_test_${randomBytes(8).toString('hex')}`
let dsn
try {
  dsn = new URL(process.env.MOETOPIA_DATABASE_URL ?? '')
} catch {
  throw new Error('A valid development MOETOPIA_DATABASE_URL is required; credential values were not displayed')
}
if (process.env.MOETOPIA_ENV !== 'development' || dsn.pathname !== '/moetopia_dev' || dsn.port !== '55432') throw new Error('This runner requires the explicit development database on port 55432')
const env = { ...process.env, MOETOPIA_TEST_DATABASE_NAME: name }
const execute = (command, args, extra = {}) => {
  const result = spawnSync(command, args, { stdio: 'inherit', windowsHide: true, ...extra })
  if (result.error || result.status !== 0) throw new Error(`${command} failed; no credential values were displayed`)
}
execute('go', ['-C', 'apps/api', 'run', './cmd/testdb', 'create'], { env })
try {
  dsn.pathname = `/${name}`
  execute('go', ['-C', 'apps/api', 'test', '-tags', 'integration', '-count=1', '-p', '1', './internal/authorization', './internal/httpserver'], { env: { ...env, TEST_DATABASE_DSN: dsn.href, TEST_VALKEY_ADDRESS: process.env.MOETOPIA_VALKEY_ADDRESS, TEST_VALKEY_PASSWORD: process.env.MOETOPIA_VALKEY_PASSWORD } })
} finally {
  if (!/^moetopia_test_[a-f0-9]{16}$/.test(name)) throw new Error('Refusing to clean up an unexpected database')
  execute('go', ['-C', 'apps/api', 'run', './cmd/testdb', 'drop'], { env })
}
