import { spawn } from 'node:child_process'
import { resolve } from 'node:path'

const binary = process.env.MOETOPIA_TEST_API_BINARY
const directory = resolve('.data/bin')
if (!binary || !resolve(binary).startsWith(directory + (process.platform === 'win32' ? '\\' : '/'))) throw new Error('Expected an owned test API binary')
const child = spawn(binary, [], { stdio: 'inherit', windowsHide: true })
child.once('error', () => { process.exitCode = 1 })
child.once('exit', code => { process.exitCode = code ?? 1 })
process.on('SIGTERM', () => child.kill('SIGTERM'))
process.on('SIGINT', () => child.kill('SIGINT'))
