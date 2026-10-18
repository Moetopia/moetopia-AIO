import { spawnSync } from 'node:child_process'

const result = spawnSync('go', ['-C', 'apps/api', 'run', './cmd/devoauthclient', ...process.argv.slice(2)], {
  stdio: 'inherit',
  windowsHide: true,
  env: process.env,
})
if (result.error) throw new Error('Could not start the development OAuth client command')
process.exitCode = result.status ?? 1
