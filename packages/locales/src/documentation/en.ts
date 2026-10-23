import type { DocumentationMessages } from './types'
export default {
  "labels": {
    "test": "Try this API",
    "run": "Send request",
    "openTest": "Open test window",
    "clearTest": "Clear test data",
    "testHint": "Requests go to this site. Write operations change real data; use test accounts and applications. Inputs and responses remain only on this page and are cleared when you leave.",
    "serverOnly": "This endpoint requires a backend client secret. Test from your own backend or terminal; do not enter client_secret into this page.",
    "adminOnly": "This endpoint needs the independent administrator session. Use its management-console action; the user site does not borrow administrator Cookies.",
    "requestBody": "JSON request body",
    "bearer": "Access token",
    "result": "Actual response",
    "elapsed": "Elapsed",
    "cancelTest": "Stop waiting",
    "testFailure": "The request did not complete. Check service availability, parameters or timeout. A write may still have completed; it will not be retried automatically.",
    "invalidTest": "Fill the required parameters and check the JSON format.",
    "redirectTest": "Authorization and sign-out use browser navigation. Complete the flow in the new window.",
    "pkce": "Generate state / nonce / PKCE",
    "verifier": "This code_verifier",
    "testOperation": "Test endpoint",
    "testSecret": "Only for this test. Cleared when you change endpoints or leave the page.",
    "terminal": "Terminal template (replace placeholders before running)",
    "title": "Developer documentation",
    "description": "From your first sign-in request to a production integration.",
    "guides": "Integration guides",
    "reference": "API reference",
    "outline": "On this page",
    "search": "Search endpoints",
    "searchPlaceholder": "Path, operation title or operationId",
    "group": "API group",
    "method": "HTTP method",
    "all": "All",
    "empty": "No matching endpoints. Adjust your filters.",
    "download": "Download OpenAPI",
    "source": "API structures come from the generated OpenAPI for this build. Discover capabilities from your target issuer.",
    "languageNote": "Operation titles and field descriptions retain the backend contract language. Guides and navigation are available in four languages.",
    "parameters": "Parameters",
    "body": "Request body",
    "responses": "Responses",
    "authentication": "Authentication and access",
    "required": "Required",
    "optional": "Optional",
    "none": "No fields",
    "raw": "Raw operation definition",
    "schema": "Schema fields",
    "requestExample": "Request example",
    "exampleNote": "Examples use placeholders to illustrate request formats. Use test accounts and Public clients for browser tests. Do not enter backend client secrets here.",
    "protocol": "Third-party OAuth / OIDC",
    "internal": "Site business API",
    "internalNote": "For site workflows using the matching HttpOnly Cookie, Origin and CSRF token. Administrative operations additionally require a role and valid MFA. OAuth Bearer does not replace these requirements.",
    "protocolNote": "Use your registered client type and supported grant. Basic credentials belong on your server; Bearer requires an access token. Check descriptions and the raw oneOf schema for conditional requirements.",
    "back": "Back to API reference",
    "related": "Related resources",
    "previous": "Previous",
    "next": "Next",
    "retry": "Reload",
    "loadError": "Documentation could not be loaded. Please retry.",
    "fields": "Fields",
    "status": "Status",
    "media": "Content type",
    "headers": "Response headers",
    "noBody": "No response body",
    "deprecated": "Deprecated",
    "contract": "View API contract",
    "count": "operations",
    "sdk": "Use a maintained standard OIDC client library. HTTP examples explain the protocol and do not replace callback and token validation performed by a library."
  },
  "guides": {
    "overview": {
      "title": "Quick start",
      "description": "Complete one authorization-code login before enabling additional capabilities.",
      "sections": [
        {
          "id": "prepare",
          "title": "Create a test application",
          "paragraphs": [
            "Create an application in My applications and register its name and callback. The application UUID identifies a console record; client_id identifies your OAuth client. Use Public for clients that cannot keep secrets, and Confidential for a backend that can.",
            "Only the owner and explicitly enrolled test members can use a test application. Creating a draft or submitting a review does not publish it."
          ],
          "code": "discovery",
          "operations": [
            "oidc-discovery"
          ]
        },
        {
          "id": "connect",
          "title": "Complete the sign-in flow",
          "paragraphs": [
            "Read Discovery from your target issuer and use its authorization_endpoint, token_endpoint and jwks_uri. Do not infer your production issuer from this documentation site.",
            "Generate state, nonce and a PKCE verifier/S256 challenge per login, bind them to your application session, navigate to authorization, validate the callback and ID Token, then establish your own application session."
          ],
          "code": "authorize",
          "operations": [
            "oauth-authorize",
            "oauth-token",
            "oauth-userinfo"
          ]
        },
        {
          "id": "publish",
          "title": "Test before publishing",
          "paragraphs": [
            "Test denied consent, invalid callbacks, replay and expiry. Submit a review when the application is ready for users beyond the test group.",
            "Sensitive configuration changes create a new review version. The previous approved configuration continues serving until the new one is approved. Read the following guides before going live."
          ]
        }
      ]
    },
    "authentication": {
      "title": "Authorization-code sign-in",
      "description": "Use Authorization Code, PKCE S256 and standard OIDC validation.",
      "sections": [
        {
          "id": "transaction",
          "title": "Create a single-use login transaction",
          "paragraphs": [
            "Use cryptographically random state, nonce and code_verifier for every request. State and OIDC nonce require at least eight characters; use high-entropy random values and bind them to the browser transaction. Derive the challenge with SHA-256 and unpadded base64url.",
            "Only code, query response mode and S256 are supported. Requests with openid require nonce. Request one registered audience; it may be omitted when only one is registered. Match the registered redirect URI."
          ],
          "code": "authorize",
          "operations": [
            "oauth-authorize"
          ]
        },
        {
          "id": "callback",
          "title": "Validate the callback and tokens",
          "paragraphs": [
            "Handle error and validate state and transaction expiry before exchanging the code using form encoding. Use the original verifier and identical redirect_uri. Codes are single-use; do not automatically retry consumed codes.",
            "Use an OIDC library to validate signatures, issuer, audience/client_id, expiry, nonce and applicable azp, auth_time and at_hash. Decoding a JWT is insufficient. Consume the transaction and identify users by (iss, sub)."
          ],
          "code": "token",
          "operations": [
            "oauth-token",
            "oauth-jwks"
          ]
        },
        {
          "id": "claims",
          "title": "Read approved identity claims",
          "paragraphs": [
            "UserInfo requires an access token, not an ID Token. Email and profile claims require their scopes and may change; they are not permanent account keys.",
            "prompt=login forces authentication; consent requests consent; none currently returns interaction_required. max_age=0 requires authentication after the current request began."
          ],
          "code": "userinfo",
          "operations": [
            "oauth-userinfo"
          ]
        }
      ]
    },
    "credentials": {
      "title": "Applications and credentials",
      "description": "Choose a client type and understand testing, review and effective versions.",
      "sections": [
        {
          "id": "types",
          "title": "Public and Confidential",
          "paragraphs": [
            "Choose Public when a client cannot protect secrets. Secrets shipped in JavaScript or native packages do not make a client confidential. Public uses client_id and PKCE; Confidential keeps secrets in its backend.",
            "Use a standard OAuth library for Basic credential encoding and never mix Basic with form credentials. Introspection accepts only Confidential HTTP Basic. Client credentials require an eligible Confidential client."
          ]
        },
        {
          "id": "versions",
          "title": "Testing and review",
          "paragraphs": [
            "Owners can add explicit test members; removal revokes the relevant online authorization eligibility. Pending review versions cannot be edited; rejected versions can be revised and resubmitted.",
            "Owner enablement, platform suspension and the approved version determine availability. Re-enabling an application does not override a suspension or restore revoked grants."
          ],
          "operations": [
            "get-api-v1-developer-applications",
            "get-api-v1-developer-applications-by-id-versions"
          ]
        },
        {
          "id": "secrets",
          "title": "Secret lifecycle",
          "paragraphs": [
            "A Confidential secret is shown once after creation or rotation. Store it in your backend secret manager, never in frontend code, URLs, logs or screenshots.",
            "Rotation immediately invalidates the old secret. Prepare a coordinated switch and use the current revision and identity proof. Do not retry rotation blindly when its result is unknown."
          ],
          "operations": [
            "post-api-v1-developer-applications-by-id-secret"
          ]
        }
      ]
    },
    "tokens": {
      "title": "Tokens, refresh and revocation",
      "description": "Handle access tokens, ID Tokens and refresh tokens separately.",
      "sections": [
        {
          "id": "types",
          "title": "Token responsibilities",
          "paragraphs": [
            "An ID Token verifies a login; an access token accesses approved resources; a refresh token renews access. They are not interchangeable, and machine tokens do not represent a user login.",
            "Keep credentials on your backend and use your own secure browser session. Offline JWT validation checks issuer, audience, algorithm, expiry and required scopes, but has a revocation propagation window."
          ]
        },
        {
          "id": "refresh",
          "title": "Strict refresh rotation",
          "paragraphs": [
            "Refresh tokens require an enabled grant and explicit offline_access consent in the authorization-code flow. Serialize refresh operations and atomically replace the refresh token after success. Reusing an old value revokes the whole family, with no concurrency grace period.",
            "Currently omit scope/audience to retain the grant or pass the exact original values. Do not change audience or depend on scope narrowing. An ambiguous network result must not trigger a blind retry with the old refresh token."
          ],
          "code": "refresh",
          "operations": [
            "oauth-token"
          ]
        },
        {
          "id": "revoke",
          "title": "Revocation and online checks",
          "paragraphs": [
            "Revocation returns an empty 200 even for unknown, expired or another client’s tokens. This does not reveal whether a token exists or end all site sessions.",
            "Introspection is restricted to Confidential backends checking their own access/refresh tokens. Inactive responses contain only active=false. Do not cache active=true while claiming immediate status checks."
          ],
          "code": "introspect",
          "operations": [
            "oauth-revoke",
            "oauth-introspect"
          ]
        }
      ]
    },
    "scopes": {
      "title": "Scopes and identity",
      "description": "Request minimal permissions and separate identity from resource access.",
      "sections": [
        {
          "id": "scopes",
          "title": "Requested and granted scopes",
          "paragraphs": [
            "openid enables OIDC; profile permits profile claims; email permits email claims; offline_access enables approved refresh access. Request only registered and approved scopes and read the actual granted scope from the token response.",
            "Do not copy permissions from unrelated ecosystem modules. Confirm the resource server, audience and its scope definitions before using business APIs. UserInfo is not a general identity database."
          ]
        },
        {
          "id": "identity",
          "title": "Map users safely",
          "paragraphs": [
            "Use (iss, sub) as the external identity key. Do not merge accounts by mutable email. UserInfo sub must match the ID Token subject, and omitted claims must remain unknown.",
            "Test membership does not grant administration rights. Account, portal and admin business APIs use their own Cookie sessions; arbitrary third-party Bearer tokens do not authorize them."
          ],
          "code": "userinfo",
          "operations": [
            "oauth-userinfo"
          ]
        }
      ]
    },
    "logout": {
      "title": "Sign-out",
      "description": "Distinguish application sign-out, central sign-out and grant revocation.",
      "sections": [
        {
          "id": "local",
          "title": "End your application session",
          "paragraphs": [
            "Your server must invalidate its own session and Cookie. Moetopia user-site sign-out affects its RP session; clearing localStorage is not server-side revocation.",
            "Signing out does not automatically revoke OAuth grants. Revoke tokens separately when long-lived access must end."
          ]
        },
        {
          "id": "central",
          "title": "Request central sign-out",
          "paragraphs": [
            "Read end_session_endpoint from Discovery. Provide a valid ID Token from this issuer as id_token_hint, or a client_id. Match an approved post_logout_redirect_uri and verify the returned state.",
            "The user must explicitly confirm central sign-out. Do not confirm automatically or promise immediate sign-out across all RPs and devices."
          ],
          "code": "logout",
          "operations": [
            "oauth-logout-get",
            "oauth-logout-post"
          ]
        }
      ]
    },
    "errors": {
      "title": "Errors and troubleshooting",
      "description": "Identify protocol errors, business errors and ambiguous network outcomes.",
      "sections": [
        {
          "id": "protocol",
          "title": "OAuth and OIDC errors",
          "paragraphs": [
            "Protocol errors use error/error_description and may redirect to a validated callback. Without a trusted callback they can be returned directly. Always validate state; HTTP 200 alone is not success.",
            "For invalid_client check client type and authentication. For invalid_grant check consumption, expiry, verifier and redirect_uri. access_denied means consent was denied; interaction_required needs interactive sign-in."
          ]
        },
        {
          "id": "business",
          "title": "Site business APIs",
          "paragraphs": [
            "Business APIs use Problem Details with optional code and field errors. For 401 reauthenticate; for 403 check permissions/MFA/Origin/CSRF; for 409 reload the revision and ask for a new decision; for 422 fix input; respect Retry-After for 429 and preserve input on 503.",
            "Omit empty optional query parameters. Bare ?cursor or ?after are not first-page placeholders. Use the response cursor for pagination; an empty next_cursor/next_after means the end."
          ]
        },
        {
          "id": "diagnose",
          "title": "Collect useful evidence",
          "paragraphs": [
            "Record the method, endpoint, status and nonsensitive request_id, client type and reproduction steps. Never record Cookies, Authorization, passwords, codes, verifiers, secrets or full request bodies.",
            "Verify Discovery, issuer, time synchronization, callbacks, registered scopes and application test/review/suspension state. Do not disable PKCE or signature checks to work around an error."
          ]
        }
      ]
    },
    "production": {
      "title": "Production checklist",
      "description": "Validate the real deployment and application backend.",
      "sections": [
        {
          "id": "configuration",
          "title": "Configuration and secrets",
          "paragraphs": [
            "Choose the production issuer and HTTPS callbacks. Use the correct Discovery and client configuration rather than copying development loopback settings, reference-project domains or credentials.",
            "Confirm approval, enablement and suspension state. Protect credentials, synchronize clocks, define dependency timeouts and controlled retries, and exclude credentials from logs."
          ]
        },
        {
          "id": "acceptance",
          "title": "Acceptance scenarios",
          "paragraphs": [
            "Test successful and denied sign-in, invalid state/nonce/verifier, callback replay, code replay, expiry, concurrent refresh and reuse, key rotation, application suspension and disabled accounts. Unit tests are not production interoperability evidence.",
            "Verify local sign-out, central confirmation, revocation and downstream offline JWT windows. Assign owners and actionable procedures for monitoring, backups, recovery and key changes."
          ]
        },
        {
          "id": "limits",
          "title": "Current boundaries",
          "paragraphs": [
            "Dynamic client registration, implicit flow, password grant and general multitenancy are not provided. Library capabilities do not imply deployed service support.",
            "Use target Discovery, approved configuration and this API reference. Capacity, production token lifetimes and external identity providers need separate acceptance."
          ]
        }
      ]
    }
  }
} satisfies DocumentationMessages
