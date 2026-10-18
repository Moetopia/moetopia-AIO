import { spawn, spawnSync } from 'node:child_process'
import { resolve } from 'node:path'
import { checkPort, serviceCommand, servicePort, services } from './dev-config.mjs'

const root = resolve(import.meta.dirname, '..')
process.chdir(root)
const name = process.argv[2]
if (!Object.hasOwn(services, name)) throw new Error('A known development service name is required')
if (!process.env.npm_execpath) throw new Error('Start services through pnpm dev')
await checkPort(name, servicePort(name))
const [command, ...args] = serviceCommand(name, process.execPath, process.env.npm_execpath)
console.log(`[${name}] Starting ${services[name].description} on port ${servicePort(name)}`)
const child = spawn(command, args, { cwd: root, stdio: 'inherit', windowsHide: true })
let stopping = false
let timeout

const stop = () => {
  if (stopping) return
  stopping = true
  if (!child.pid || child.exitCode !== null || child.signalCode !== null) return
  if (process.platform === 'win32') {
    spawnSync('taskkill', ['/PID', String(child.pid), '/T', '/F'], { stdio: 'ignore', windowsHide: true })
  } else {
    child.kill('SIGTERM')
    timeout = setTimeout(() => child.kill('SIGKILL'), 5000)
  }
}

process.on('SIGINT', stop)
process.on('SIGTERM', stop)
child.once('error', (error) => {
  console.error(`[${name}] Could not start service: ${error.code ?? 'unknown error'}`)
  process.exitCode = 1
})
child.once('exit', (code, signal) => {
  clearTimeout(timeout)
  process.exitCode = stopping ? 0 : code || 1
  console.error(`[${name}] ${stopping ? 'Stopped' : `Exited (${signal ?? code}); inspect this pane, fix the cause and press r to restart`}`)
})
