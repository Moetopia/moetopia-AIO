import { spawn } from 'node:child_process'
import { resolve, sep } from 'node:path'

const directory = process.env.MOETOPIA_TEST_OIDC_DIRECTORY
if (!directory || !resolve(directory).startsWith(resolve('.data') + sep)) throw new Error('An owned OIDC test directory is required')
const part = process.argv[2]
const command = part === 'api' ? process.env.MOETOPIA_TEST_API_BINARY : process.execPath
const args = part === 'api' ? [] : [resolve(directory, 'output/server/index.mjs')]
if (!['api', 'account'].includes(part) || !command) throw new Error('Expected an OIDC test server')
if (part === 'api' && !resolve(command).startsWith(resolve(directory) + sep)) throw new Error('An owned OIDC test API binary is required')
const child = spawn(command, args, { stdio: 'inherit', windowsHide: true, env: { ...process.env, HOST: '127.0.0.1', PORT: '54305' } })
child.once('error', () => { process.exitCode = 1 })
child.once('exit', code => { process.exitCode = code ?? 1 })
process.on('SIGTERM', () => child.kill('SIGTERM'))
process.on('SIGINT', () => child.kill('SIGINT'))
