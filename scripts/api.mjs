import { spawn } from 'node:child_process'

const commands = {
  dev: ['run', './cmd/api'],
  migrate: ['run', './cmd/migrate'],
  'keys-init': ['run', './cmd/keys', 'init'],
  'dev-invite': ['run', './cmd/devinvite'],
  'first-party-init': ['run', './cmd/firstparty'],
  'grants-backfill': ['run', './cmd/backfillgrants'],
}
const command = process.argv[2]
if (!Object.hasOwn(commands, command)) throw new Error('Unknown API task')
const child = spawn('go', ['-C', 'apps/api', ...commands[command]], { stdio: 'inherit', windowsHide: true })
child.on('error', () => { process.exitCode = 1 })
child.on('exit', (code) => { process.exitCode = code ?? 1 })
