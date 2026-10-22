package httpserver

import (
	"net/http"
	"sort"

	"github.com/danielgtaylor/huma/v2"
)

type protocolDocumenter func(string, string, string, string, string, map[string]*huma.Response) *huma.Operation

func documentProtocolMetadata(add protocolDocumenter) {
	properties := map[string]*huma.Schema{}
	for _, name := range []string{"issuer", "authorization_endpoint", "token_endpoint", "userinfo_endpoint", "jwks_uri", "end_session_endpoint", "revocation_endpoint", "introspection_endpoint"} {
		properties[name] = &huma.Schema{Type: "string", Format: "uri"}
	}
	for _, name := range []string{"response_types_supported", "response_modes_supported", "grant_types_supported", "revocation_endpoint_auth_methods_supported", "introspection_endpoint_auth_methods_supported", "subject_types_supported", "id_token_signing_alg_values_supported", "token_endpoint_auth_methods_supported", "code_challenge_methods_supported", "scopes_supported", "claims_supported"} {
		properties[name] = &huma.Schema{Type: "array", Items: &huma.Schema{Type: "string"}}
	}
	properties["request_parameter_supported"], properties["request_uri_parameter_supported"] = &huma.Schema{Type: "boolean"}, &huma.Schema{Type: "boolean"}
	required := []string{}
	for name := range properties {
		required = append(required, name)
	}
	sort.Strings(required)
	add(http.MethodGet, "/.well-known/openid-configuration", "oidc-discovery", "查询 OIDC 服务发现信息", "固定 issuer 的实际端点、scope、认证方式和算法；客户端应从此发现能力。", map[string]*huma.Response{"200": protocolJSON("OIDC discovery metadata", protocolObject(properties, required...))})
	add(http.MethodGet, "/oauth/jwks", "oauth-jwks", "查询当前可接受的签名公钥", "只返回 active 和重叠窗口内公钥，不含私钥。客户端按 kid 选择并按需刷新；不能永久固定旧公钥。", map[string]*huma.Response{
		"200": protocolJSON("JSON Web Key Set", protocolObject(map[string]*huma.Schema{"keys": {Type: "array", Items: protocolObject(map[string]*huma.Schema{
			"kty": protocolString("密钥类型", "RSA"), "kid": protocolString("签名密钥标识"), "use": protocolString("签名用途", "sig"),
			"alg": protocolString("签名算法", "RS256"), "n": protocolString("RSA modulus，base64url"), "e": protocolString("RSA exponent，base64url"),
		}, "kty", "kid", "n", "e")}}, "keys")),
		"503": protocolJSON("公钥存储不可用", protocolObject(map[string]*huma.Schema{"error": {Type: "string"}}, "error")),
	})
}
