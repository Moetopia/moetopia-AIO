export const documentationExamples: Record<string, string> = {
  discovery: 'curl "https://YOUR_ISSUER/.well-known/openid-configuration"',
  authorize: 'GET /oauth/authorize?response_type=code\n  &client_id=YOUR_CLIENT_ID\n  &redirect_uri=https%3A%2F%2Fapp.example.com%2Fcallback\n  &scope=openid%20profile%20email\n  &state=RANDOM_SINGLE_USE_STATE\n  &nonce=RANDOM_SINGLE_USE_NONCE\n  &code_challenge=BASE64URL_SHA256_OF_VERIFIER\n  &code_challenge_method=S256',
  token: 'curl "https://YOUR_ISSUER/oauth/token" \\\n  --data-urlencode "grant_type=authorization_code" \\\n  --data-urlencode "client_id=YOUR_PUBLIC_CLIENT_ID" \\\n  --data-urlencode "code=ONE_TIME_AUTHORIZATION_CODE" \\\n  --data-urlencode "redirect_uri=https://app.example.com/callback" \\\n  --data-urlencode "code_verifier=ORIGINAL_PKCE_VERIFIER"',
  userinfo: 'curl "https://YOUR_ISSUER/oauth/userinfo" \\\n  --header "Authorization: Bearer YOUR_ACCESS_TOKEN"',
  refresh: 'curl "https://YOUR_ISSUER/oauth/token" \\\n  --data-urlencode "grant_type=refresh_token" \\\n  --data-urlencode "client_id=YOUR_PUBLIC_CLIENT_ID" \\\n  --data-urlencode "refresh_token=CURRENT_REFRESH_TOKEN"',
  introspect: 'curl "https://YOUR_ISSUER/oauth/introspect" \\\n  --user "YOUR_CLIENT_ID:YOUR_CLIENT_SECRET" \\\n  --data-urlencode "token=YOUR_TOKEN" \\\n  --data-urlencode "token_type_hint=access_token"',
  logout: 'GET /oauth/logout?client_id=YOUR_CLIENT_ID\n  &post_logout_redirect_uri=https%3A%2F%2Fapp.example.com%2Fsigned-out\n  &state=RANDOM_SIGN_OUT_STATE',
}
export const documentationSources = [
  { label: 'OIDC Core 1.0', href: 'https://openid.net/specs/openid-connect-core-1_0.html' },
  { label: 'OAuth Security BCP · RFC 9700', href: 'https://www.rfc-editor.org/rfc/rfc9700.html' },
  { label: 'RP-Initiated Logout 1.0', href: 'https://openid.net/specs/openid-connect-rpinitiated-1_0.html' },
]
