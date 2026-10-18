import assert from 'node:assert/strict'
import { createServer } from 'node:net'
import { test } from 'node:test'
import { checkPort, selectedServices, serviceCommand, servicePort, tuiConfig } from './dev-config.mjs'

test('service selection rejects typos and does not start unrelated apps', () => {
  assert.deepEqual(selectedServices(['--plain', '--', 'api', 'account', 'api']), ['api', 'account'])
  assert.deepEqual(selectedServices([]), ['api', 'account', 'admin'])
  assert.throws(() => selectedServices(['developer']), /Developer pages are served by account/)
  assert.throws(() => selectedServices(['acount']), /Unknown service/)
  assert.equal(servicePort('api', { MOETOPIA_LISTEN: '127.0.0.1:58480' }), 58480)
  assert.throws(() => servicePort('api', { MOETOPIA_LISTEN: '127.0.0.1:invalid' }), /valid explicit port/)
})

test('an occupied port is rejected without disturbing its owner', async (t) => {
  const owner = createServer(socket => socket.end())
  await new Promise(resolve => owner.listen(0, '127.0.0.1', resolve))
  t.after(() => new Promise(resolve => owner.close(resolve)))
  const port = owner.address().port
  await assert.rejects(checkPort('api', port), /api: port .* is occupied/)
  assert.equal(owner.listening, true)
})

test('commands preserve executable and workspace paths as arguments', () => {
  const node = 'C:/Program Files/nodejs/node.exe'
  const pnpm = 'D:/Tools & Packages/pnpm/pnpm.exe'
  assert.deepEqual(serviceCommand('account', node, pnpm), [pnpm, '--filter', '@moetopia/account', 'dev'])
  assert.deepEqual(serviceCommand('admin', node, 'C:/Program Files/pnpm.cjs'), [node, 'C:/Program Files/pnpm.cjs', '--filter', '@moetopia/admin', 'dev'])
  const config = tuiConfig('D:/My Projects/Moetopia', node, ['api', 'account'])
  assert.deepEqual(config.procs.api.cmd, [node, 'scripts/dev-service.mjs', 'api'])
  assert.equal(config.procs.account.cwd, 'D:/My Projects/Moetopia')
  assert.equal(config.procs.api.autorestart, false)
  assert.equal(config.proc_log, false)
})
