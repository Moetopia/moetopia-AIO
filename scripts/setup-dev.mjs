import { randomBytes } from 'node:crypto'
import { readFile, writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { parseArgs } from 'node:util'
import { isIP } from 'node:net'

const root = resolve(import.meta.dirname, '..')
const { values } = parseArgs({ options: { 'services-host': { type: 'string', default: '127.0.0.1' } } })
const host = values['services-host']
if (isIP(host) !== 4 || host === '0.0.0.0' || host.startsWith('127.') && host !== '127.0.0.1') {
  throw new Error('--services-host must be an explicit IPv4 address (127.0.0.1 for local development)')
}
const template = await readFile(resolve(root, '.env.example'), 'utf8')
const password = randomBytes(24).toString('hex')
const valkeyPassword = randomBytes(24).toString('hex')
const encryptionKey = randomBytes(32).toString('base64')
const content = template
  .replaceAll('replace-with-a-local-only-password', password)
  .replaceAll('replace-with-a-valkey-only-password', valkeyPassword)
  .replace('MOETOPIA_ENCRYPTION_KEY=', `MOETOPIA_ENCRYPTION_KEY=${encryptionKey}`)
  .replace('@127.0.0.1:55432', `@${host}:55432`)
  .replace('MOETOPIA_VALKEY_ADDRESS=127.0.0.1', `MOETOPIA_VALKEY_ADDRESS=${host}`)
  .replace('MOETOPIA_SMTP_ADDRESS=127.0.0.1', `MOETOPIA_SMTP_ADDRESS=${host}`)
const composeContent = `MOETOPIA_DEV_BIND_ADDRESS=${host}\nMOETOPIA_DEV_DB_PASSWORD=${password}\nMOETOPIA_DEV_VALKEY_PASSWORD=${valkeyPassword}\n`

try {
  await writeFile(resolve(root, '.env'), content, { flag: 'wx', mode: 0o600 })
  process.stdout.write('Created local .env with generated credentials. Values were not displayed.\n')
} catch (error) {
  if (error.code !== 'EEXIST') throw error
  process.stderr.write('.env already exists; existing credentials were preserved.\n')
  process.exitCode = 1
}
if (!process.exitCode) {
  try {
    await writeFile(resolve(root, '.env.compose'), composeContent, { flag: 'wx', mode: 0o600 })
    process.stdout.write('Created .env.compose for the dependency host; it contains no application encryption key.\n')
  } catch (error) {
    if (error.code !== 'EEXIST') throw error
    process.stderr.write('.env.compose already exists and was preserved; reconcile its dependency credentials with the new .env before starting services.\n')
    process.exitCode = 1
  }
}
