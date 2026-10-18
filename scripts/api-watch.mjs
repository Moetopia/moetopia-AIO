import { spawn, spawnSync } from 'node:child_process'
import { watch, mkdirSync, rmSync } from 'node:fs'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '..')
const directory = resolve(root, '.data/bin')
mkdirSync(directory, { recursive: true })
let running
let building
let rebuilding = false
let currentBinary
let sequence = 0
let timer
let queued = false
let stopping = false
const binaries = new Set()
const terminate = (child) => {
  if (!child?.pid || child.exitCode !== null) return
  if (process.platform === 'win32') spawnSync('taskkill', ['/PID', String(child.pid), '/T', '/F'], { stdio: 'ignore', windowsHide: true })
  else child.kill('SIGTERM')
}
const finish = (code = 0) => {
  if (stopping) return
  stopping = true
  clearTimeout(timer)
  watcher.close()
  terminate(building)
  terminate(running)
  for (const binary of binaries) { try { rmSync(binary, { force: true }) } catch {} }
  process.exitCode = code
}
const rebuild = () => {
  if (stopping) return
  if (rebuilding) { queued = true; return }
  rebuilding = true
  const binary = resolve(directory, `api-dev-${process.pid}-${++sequence}${process.platform === 'win32' ? '.exe' : ''}`)
  binaries.add(binary)
  console.log('[api] Building Go changes…')
  building = spawn('go', ['-C', 'apps/api', 'build', '-o', binary, './cmd/api'], { cwd: root, stdio: 'inherit', windowsHide: true })
  building.once('error', () => finish(1))
  building.once('exit', async (code) => {
    building = undefined
    if (stopping) return
    if (code === 0) {
      const previous = running
      running = undefined
      if (previous && previous.exitCode === null) {
        await new Promise(resolveExit => {
          const timeout = setTimeout(() => { previous.kill('SIGKILL') }, 5000)
          previous.once('exit', () => { clearTimeout(timeout); resolveExit() })
          terminate(previous)
        })
      }
      if (stopping) return
      if (currentBinary) { try { rmSync(currentBinary, { force: true }); binaries.delete(currentBinary) } catch {} }
      currentBinary = binary
      running = spawn(binary, [], { cwd: root, stdio: 'inherit', windowsHide: true })
      const owned = running
      owned.once('error', () => finish(1))
      owned.once('exit', (status) => { if (!stopping && running === owned) { console.error('[api] API exited; check the error above.'); finish(status || 1) } })
    } else {
      console.error('[api] Build failed. Fix the errors and save again; the previous API stays running.')
      if (!running) finish(code || 1)
    }
    rebuilding = false
    if (queued) { queued = false; rebuild() }
  })
}
const watcher = watch(resolve(root, 'apps/api'), { recursive: true }, (_event, filename) => {
  if (!filename || !/\.(go|sql)$|^go\.(mod|sum)$/.test(filename)) return
  clearTimeout(timer)
  timer = setTimeout(rebuild, 700)
})
watcher.on('error', () => finish(1))
process.on('SIGINT', () => finish())
process.on('SIGTERM', () => finish())
rebuild()
