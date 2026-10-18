import { spawn, spawnSync } from 'node:child_process'
import { existsSync, mkdirSync, rmSync, writeFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { resolve } from 'node:path'
import { createInterface } from 'node:readline'
import { checkPort, selectedServices, servicePort, tuiConfig } from './dev-config.mjs'

const root = resolve(import.meta.dirname, '..')
const args = process.argv.slice(2)
if (args.includes('--help')) {
  console.log('Usage: pnpm dev [api account admin developer storybook] [--plain]\nDefault: API + three apps, with separate mprocs log panes in an interactive terminal.\nUse --plain for prefixed logs without a TUI. Storybook starts only when selected.')
  process.exit(0)
}
const names = selectedServices(args)
process.chdir(root)
if (!existsSync('.env')) throw new Error('Run node scripts/setup-dev.mjs before starting development')
process.loadEnvFile('.env')
if (!process.env.npm_execpath) throw new Error('Start the workspace using pnpm dev')
await Promise.all(names.map(name => checkPort(name, servicePort(name))))

const children = []
let stopping = false
let configPath
let failures = 0
const stop = (code = 0) => {
  if (stopping) return
  stopping = true
  for (const child of children) {
    if (!child.pid || child.exitCode !== null || child.signalCode !== null) continue
    if (process.platform === 'win32') spawnSync('taskkill', ['/PID', String(child.pid), '/T', '/F'], { stdio: 'ignore', windowsHide: true })
    else child.kill('SIGTERM')
  }
  process.exitCode = code
}

const interactive = process.stdin.isTTY && process.stdout.isTTY && !args.includes('--plain')
if (interactive) {
  const require = createRequire(import.meta.url)
  const binary = require('mprocs')
  if (!binary) throw new Error('mprocs has no binary for this platform; use pnpm dev:plain')
  const directory = resolve(root, '.data/dev')
  mkdirSync(directory, { recursive: true })
  configPath = resolve(directory, `mprocs-${process.pid}.yaml`)
  writeFileSync(configPath, JSON.stringify(tuiConfig(root, process.execPath, names), null, 2), { flag: 'wx' })
  const child = spawn(binary, ['--config', configPath, '--proc-list-title', 'Moetopia'], { cwd: root, stdio: 'inherit', windowsHide: true })
  children.push(child)
  child.once('error', () => { console.error('Could not start mprocs; try pnpm dev:plain'); stop(1) })
  child.once('exit', (code) => { rmSync(configPath, { force: true }); process.exitCode = code ?? 1 })
} else {
  console.log('Plain log mode. Each line identifies its service. Use Windows Terminal + pnpm dev for separate TUI panes.')
  for (const name of names) {
    const child = spawn(process.execPath, ['scripts/dev-service.mjs', name], { cwd: root, stdio: ['ignore', 'pipe', 'pipe'], windowsHide: true })
    children.push(child)
    for (const [stream, output] of [[child.stdout, process.stdout], [child.stderr, process.stderr]]) {
      createInterface({ input: stream }).on('line', line => output.write(`[${name}] ${line}\n`))
    }
    child.once('error', () => { console.error(`[${name}] Failed to launch`); failures++ })
    child.once('exit', (code) => {
      if (stopping) return
      if (code) failures++
      if (children.every(value => value.exitCode !== null || value.signalCode !== null)) process.exitCode = failures ? 1 : 0
    })
  }
}
process.on('SIGINT', () => stop())
process.on('SIGTERM', () => stop())
process.on('exit', () => { if (configPath) rmSync(configPath, { force: true }) })
