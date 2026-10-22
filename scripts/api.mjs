import { spawn } from 'node:child_process'

const commands = {
  dev: ['run', './cmd/api'],
  migrate: ['run', './cmd/migrate'],
  'keys-init': ['run', './cmd/keys', 'init'],
  'keys-rotate': ['run', './cmd/keys', 'rotate'],
  'keys-emergency': ['run', './cmd/keys', 'emergency'],
  'dev-invite': ['run', './cmd/devinvite'],
  'first-party-init': ['run', './cmd/firstparty'],
  'grants-backfill': ['run', './cmd/backfillgrants'],
  'admin-init': ['run', './cmd/administrator', 'init-client'],
  'admin-bootstrap': ['run', './cmd/administrator', 'bootstrap'],
  'admin-emergency': ['run', './cmd/administrator', 'emergency-reset'],
  'admin-inspect': ['run', './cmd/administrator', 'inspect'],
}
const command = process.argv[2]
if (!Object.hasOwn(commands, command)) throw new Error('Unknown API task')
const child = spawn('go', ['-C', 'apps/api', ...commands[command], ...process.argv.slice(3)], { stdio: 'inherit', windowsHide: true })
child.on('error', () => { process.exitCode = 1 })
child.on('exit', (code) => { process.exitCode = code ?? 1 })
