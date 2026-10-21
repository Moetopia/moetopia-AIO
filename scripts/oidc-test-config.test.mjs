import assert from 'node:assert/strict'
import { test } from 'node:test'
import { oidcAPIOrigin, oidcAPIPort } from './oidc-test-config.mjs'

test('OIDC API override changes only the dedicated loopback test port', () => {
  assert.equal(oidcAPIPort({}), 58490)
  assert.equal(oidcAPIOrigin({ MOETOPIA_TEST_API_PORT: '58500' }), 'http://127.0.0.1:58500')
  for (const value of ['', '8080', '50000', '65536', '54305', '54310', '54311', '58500/path', 'attacker.test', '5e4', ' 58500']) assert.throws(() => oidcAPIPort({ MOETOPIA_TEST_API_PORT: value }), /dedicated test port/)
})
