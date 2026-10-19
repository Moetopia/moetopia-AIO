import { createServer } from 'node:http'
import * as oidc from 'openid-client'

const issuer = process.env.MOETOPIA_ISSUER
const clientID = process.env.MOETOPIA_TEST_OIDC_CLIENT
const callback = process.env.MOETOPIA_TEST_OIDC_CALLBACK
if (process.env.MOETOPIA_ENV !== 'test' || issuer !== 'http://127.0.0.1:54305' || callback !== 'http://localhost:54310/callback' || !clientID) throw new Error('The isolated OIDC test environment is required')
const origin = 'http://localhost:54310'
const sessions = new Map()
let jwksFetches = 0

const json = (res, value, status = 200) => {
  res.writeHead(status, { 'content-type': 'application/json', 'cache-control': 'no-store', 'referrer-policy': 'no-referrer' })
  res.end(JSON.stringify(value))
}
const redirect = (res, location) => { res.writeHead(303, { location, 'cache-control': 'no-store', 'referrer-policy': 'no-referrer' }); res.end() }
const readSession = req => {
  const id = req.headers.cookie?.split(';').map(value => value.trim()).find(value => value.startsWith('oidc_test_rp='))?.slice('oidc_test_rp='.length)
  const session = sessions.get(id)
  return session && session.expires > Date.now() ? session : undefined
}
const checks = session => ({ pkceCodeVerifier: session.verifier, expectedState: session.mode === 'bad-state' ? oidc.randomState() : session.state, expectedNonce: session.mode === 'bad-nonce' ? oidc.randomNonce() : session.nonce, idTokenExpected: true })

const server = createServer(async (req, res) => {
  try {
    const url = new URL(req.url, origin)
    if (url.pathname === '/healthz') return json(res, { ready: true })
    if (url.pathname === '/login') {
      const mode = url.searchParams.get('mode') ?? 'normal'
      if (!['normal', 'login', 'deny', 'bad-state', 'bad-nonce', 'bad-signature', 'offline'].includes(mode)) return json(res, { error: 'invalid_mode' }, 400)
      for (const [id, session] of sessions) if (session.expires <= Date.now()) sessions.delete(id)
      const config = await oidc.discovery(new URL(issuer), clientID, { id_token_signed_response_alg: 'RS256' }, oidc.None(), {
        execute: [oidc.allowInsecureRequests, oidc.enableNonRepudiationChecks],
        [oidc.customFetch]: async (input, options) => {
          const path = new URL(input).pathname
          if (path === '/oauth/jwks') jwksFetches++
          const response = await fetch(input, options)
          if (path !== '/oauth/token' || mode !== 'bad-signature' || !response.ok) return response
          const body = await response.json()
          const parts = body.id_token.split('.')
          parts[2] = (parts[2][0] === 'A' ? 'B' : 'A') + parts[2].slice(1)
          return new Response(JSON.stringify({ ...body, id_token: parts.join('.') }), { status: response.status, headers: response.headers })
        },
      })
      const verifier = oidc.randomPKCECodeVerifier()
      const state = oidc.randomState()
      const nonce = oidc.randomNonce()
      const id = oidc.randomState()
      sessions.set(id, { config, verifier, state, nonce, mode, expires: Date.now() + 600000 })
      const parameters = { redirect_uri: callback, response_type: 'code', scope: 'openid profile email demo.read', state, nonce, code_challenge_method: 'S256', code_challenge: await oidc.calculatePKCECodeChallenge(verifier) }
      if (mode === 'login') parameters.prompt = 'login'
      if (mode === 'offline') parameters.scope += ' offline_access'
      res.setHeader('set-cookie', `oidc_test_rp=${id}; HttpOnly; SameSite=Lax; Path=/; Max-Age=600`)
      return redirect(res, oidc.buildAuthorizationUrl(config, parameters).href)
    }
    const session = readSession(req)
    if (!session) return json(res, { error: 'rp_session_required' }, 401)
    if (req.headers.cookie?.includes('moetopia_account=')) return json(res, { error: 'account_cookie_crossed_host_boundary' }, 400)
    if (url.pathname === '/callback') {
      if (session.callback) return json(res, { error: 'rp_callback_consumed' }, 400)
      session.callback = url
      try {
        const tokens = await oidc.authorizationCodeGrant(session.config, url, checks(session))
        const claims = tokens.claims()
        const user = await oidc.fetchUserInfo(session.config, tokens.access_token, claims.sub)
        if (claims.email !== process.env.MOETOPIA_TEST_OIDC_EMAIL || claims.name !== 'OIDC reader' || user.sub !== claims.sub || user.email !== claims.email || !jwksFetches) throw new Error('identity_checks_failed')
        session.tokens = tokens
        session.result = { status: 'approved', signature_checked: true, userinfo_checked: true, identity_checked: true }
      } catch (error) {
        session.result = { status: error instanceof oidc.AuthorizationResponseError && error.error === 'access_denied' ? 'denied' : 'rejected', error_code: /^[A-Z_]+$/.test(error.code ?? '') ? error.code : 'RP_VALIDATION_FAILED' }
      }
      return redirect(res, origin + '/result')
    }
    if (url.pathname === '/result') return json(res, session.result ?? { status: 'pending' })
    if (url.pathname === '/refresh' && session.tokens?.refresh_token) {
      const original = session.tokens
      const tokens = await oidc.refreshTokenGrant(session.config, original.refresh_token)
      const claims = tokens.claims()
      const user = await oidc.fetchUserInfo(session.config, tokens.access_token, original.claims().sub)
      session.spentRefresh = original.refresh_token
      session.tokens = tokens
      return json(res, { rotated: tokens.refresh_token !== original.refresh_token, identity_preserved: claims.sub === original.claims().sub && claims.auth_time === original.claims().auth_time && user.sub === claims.sub })
    }
    if (url.pathname === '/replay-refresh' && session.spentRefresh && session.tokens) {
      let rejected = false
      try { await oidc.refreshTokenGrant(session.config, session.spentRefresh) } catch (error) { rejected = error instanceof oidc.ResponseBodyError && error.error === 'invalid_grant' }
      const response = await fetch(issuer + '/oauth/userinfo', { headers: { authorization: `Bearer ${session.tokens.access_token}` } })
      return json(res, { replay_rejected: rejected, family_revoked: response.status === 401 })
    }
    if (url.pathname === '/revoke' && session.tokens?.refresh_token) {
      await oidc.tokenRevocation(session.config, session.tokens.refresh_token)
      await oidc.tokenRevocation(session.config, session.tokens.refresh_token)
      let rejected = false
      try { await oidc.refreshTokenGrant(session.config, session.tokens.refresh_token) } catch (error) { rejected = error instanceof oidc.ResponseBodyError && error.error === 'invalid_grant' }
      return json(res, { refresh_rejected: rejected })
    }
    if (url.pathname === '/replay' && session.tokens && session.callback) {
      let replayRejected = false
      try { await oidc.authorizationCodeGrant(session.config, session.callback, checks(session)) } catch (error) { replayRejected = error instanceof oidc.ResponseBodyError && error.error === 'invalid_grant' }
      const response = await fetch(issuer + '/oauth/userinfo', { headers: { authorization: `Bearer ${session.tokens.access_token}` } })
      return json(res, { replay_rejected: replayRejected, prior_token_revoked: response.status === 401 })
    }
    json(res, { error: 'not_found' }, 404)
  } catch {
    json(res, { error: 'rp_request_failed' }, 500)
  }
})
server.listen(54310, '127.0.0.1')
const stop = () => server.close(() => process.exit(0))
process.on('SIGINT', stop)
process.on('SIGTERM', stop)
