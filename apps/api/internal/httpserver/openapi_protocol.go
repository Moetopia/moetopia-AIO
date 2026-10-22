package httpserver

import (
	"net/http"
	"sort"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

func protocolString(description string, values ...any) *huma.Schema {
	return &huma.Schema{Type: "string", Description: description, Enum: values}
}

func protocolObject(properties map[string]*huma.Schema, required ...string) *huma.Schema {
	return &huma.Schema{Type: "object", Properties: properties, Required: required}
}

func protocolJSON(description string, schema *huma.Schema) *huma.Response {
	return &huma.Response{Description: description, Content: map[string]*huma.MediaType{"application/json": {Schema: schema}}}
}

func protocolForm(schema *huma.Schema) *huma.RequestBody {
	return &huma.RequestBody{Required: true, Content: map[string]*huma.MediaType{"application/x-www-form-urlencoded": {Schema: schema}}}
}

func protocolRedirect() *huma.Response {
	return &huma.Response{Description: "浏览器跳转；保留 Location 和 Set-Cookie，不在服务器代理中自动跟随。", Headers: map[string]*huma.Param{"Location": {Schema: &huma.Schema{Type: "string"}, Required: true}}}
}

func protocolQuery(schema *huma.Schema) []*huma.Param {
	params := []*huma.Param{}
	names := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		property := schema.Properties[name]
		required := false
		for _, key := range schema.Required {
			required = required || key == name
		}
		params = append(params, &huma.Param{Name: name, In: "query", Required: required, Description: property.Description, Schema: property})
	}
	return params
}

func documentProtocolAPI(doc *huma.OpenAPI) {
	doc.Info.Description = "Moetopia 业务 API 与 OAuth/OIDC 协议。协议端点由 Fosite/原生 HTTP handler 处理，此处仅描述契约，不改变运行时处理器。部署地址与支持能力以实际 issuer 的 /.well-known/openid-configuration 为准。浏览器业务接口仍使用各自 Cookie、Origin 与 CSRF；不能用 OAuth Bearer 代替管理员 MFA。"
	doc.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"OAuthClientBasic": {Type: "http", Scheme: "basic", Description: "confidential client 的 client_id/client_secret；仅供应用后端，禁止向前端分发 secret。"},
		"OAuthAccessToken": {Type: "http", Scheme: "bearer", BearerFormat: "JWT", Description: "获准 openid scope 的用户 access token；不是 ID Token。"},
	}
	add := func(method, path, id, summary, description string, responses map[string]*huma.Response) *huma.Operation {
		op := &huma.Operation{Method: method, Path: path, OperationID: id, Summary: summary, Description: description, Tags: []string{"OAuth / OIDC"}, Responses: responses}
		doc.AddOperation(op)
		return op
	}
	errorSchema := protocolObject(map[string]*huma.Schema{
		"error": protocolString("OAuth 错误码"), "error_description": protocolString("可选错误说明，不含凭据"),
		"error_uri": protocolString("可选错误文档地址"),
	}, "error")
	errors := func() map[string]*huma.Response {
		return map[string]*huma.Response{
			"400": protocolJSON("无效协议请求；不使用业务 Problem JSON 包装。", errorSchema),
			"401": protocolJSON("客户端认证或 Bearer 认证失败。", errorSchema),
			"429": {Description: "触发限流，遵循 Retry-After。", Headers: map[string]*huma.Param{"Retry-After": {Schema: &huma.Schema{Type: "string"}}}, Content: map[string]*huma.MediaType{"application/json": {Schema: errorSchema}}},
			"503": protocolJSON("依赖不可用，拒绝放行。", errorSchema),
		}
	}

	documentProtocolMetadata(add)
	request := protocolObject(map[string]*huma.Schema{
		"response_type": protocolString("只支持授权码。", "code"), "client_id": protocolString("已注册的客户端 ID"),
		"redirect_uri": protocolString("与注册值精确匹配的回调"), "scope": protocolString("空格分隔的权限；OIDC 包含 openid"),
		"state": protocolString("至少八个字符，客户端生成并在回调校验的随机关联值"), "nonce": protocolString("请求 openid 时必填，至少八个字符，客户端须校验"),
		"response_mode": protocolString("只支持 query 模式", "query"), "audience": protocolString("已注册的单一 audience；注册值只有一个时可省略"),
		"code_challenge": protocolString("PKCE S256 challenge"), "code_challenge_method": protocolString("PKCE 算法", "S256"),
		"prompt": protocolString("login 强制重新登录，consent 要求同意；none 返回 interaction_required", "login", "consent", "none"), "max_age": {Type: "integer", Minimum: ptrFloat(0)},
	}, "response_type", "client_id", "redirect_uri", "state", "code_challenge", "code_challenge_method")
	responses := errors()
	for _, status := range []string{"400", "503"} {
		responses[status].Content["text/plain"] = &huma.MediaType{Schema: &huma.Schema{Type: "string"}}
	}
	responses["303"], responses["302"] = protocolRedirect(), protocolRedirect()
	add(http.MethodGet, "/oauth/authorize", "oauth-authorize", "发起授权码与 OIDC 登录", "使用浏览器顶层导航，进入身份站登录/同意交互；合法回调上的协议错误可通过重定向返回。PKCE 必须使用 S256。", responses).Parameters = protocolQuery(request)

	tokenForm := protocolObject(map[string]*huma.Schema{
		"grant_type":    protocolString("授权类型", "authorization_code", "refresh_token", "client_credentials"),
		"client_id":     protocolString("public client 或 client_secret_post 使用的 ID；Basic 认证不要混用表单凭据"),
		"client_secret": {Type: "string", WriteOnly: true, Description: "仅 client_secret_post；建议 confidential client 使用 HTTP Basic"},
		"code":          protocolString("authorization_code 必填，单次授权码"), "redirect_uri": protocolString("authorization_code 必填，与授权请求相同"),
		"code_verifier": protocolString("authorization_code 必填，原始 PKCE verifier"),
		"refresh_token": {Type: "string", WriteOnly: true, Description: "refresh_token grant 必填；成功后换用新凭据，旧值重用将撤销整族"},
		"scope":         protocolString("空格分隔；刷新时省略以沿用原授权，或传入与原授权相同的 scope 集合；不支持缩减"), "audience": protocolString("已注册的资源 audience；刷新时省略或与原授权相同，不可更换"),
	}, "grant_type")
	tokenForm.OneOf = []*huma.Schema{
		{Properties: map[string]*huma.Schema{"grant_type": {Const: "authorization_code"}}, Required: []string{"code", "redirect_uri", "code_verifier"}},
		{Properties: map[string]*huma.Schema{"grant_type": {Const: "refresh_token"}}, Required: []string{"refresh_token"}},
		{Properties: map[string]*huma.Schema{"grant_type": {Const: "client_credentials"}}},
	}
	responses = errors()
	responses["200"] = protocolJSON("协议令牌；浏览器业务会话由 Go 管理，不将令牌存入 localStorage。", protocolObject(map[string]*huma.Schema{
		"access_token": {Type: "string", WriteOnly: false}, "token_type": protocolString("令牌类型", "bearer"),
		"expires_in": {Type: "integer"}, "scope": protocolString("实际批准的 scope"),
		"refresh_token": protocolString("仅获准 offline_access 的授权码/刷新流程返回"), "id_token": protocolString("仅 OIDC 用户授权返回"),
	}, "access_token", "token_type", "expires_in"))
	op := add(http.MethodPost, "/oauth/token", "oauth-token", "兑换、刷新或签发令牌", "仅接受 form。confidential client 使用已注册的 Basic 或 client_secret_post；public client 仅允许相应授权码/刷新流程，不允许 client_credentials。不要自动重试已消费的授权码/刷新令牌。", responses)
	op.RequestBody, op.Security = protocolForm(tokenForm), []map[string][]string{{"OAuthClientBasic": {}}, {}}

	responses = errors()
	responses["200"] = protocolJSON("只返回获准 scope 的身份 claims。", protocolObject(map[string]*huma.Schema{
		"sub": protocolString("稳定主体 ID"), "name": protocolString("profile 获准时返回"),
		"email": protocolString("email 获准时返回"), "email_verified": {Type: "boolean"},
	}, "sub"))
	op = add(http.MethodGet, "/oauth/userinfo", "oauth-userinfo", "获取获准的用户身份信息", "使用 Authorization: Bearer access_token，要求 openid；不接受 ID Token 或机器令牌。", responses)
	op.Security = []map[string][]string{{"OAuthAccessToken": {}}}

	for _, endpoint := range []struct{ path, id, summary, description string }{
		{"/oauth/revoke", "oauth-revoke", "撤销当前客户端的令牌", "支持已注册的 Basic、client_secret_post 或 public client 认证；未知或已失效令牌同样返回空 200，不能据此枚举令牌。"},
		{"/oauth/introspect", "oauth-introspect", "检查本客户端令牌的在线状态", "只允许 confidential client 使用 HTTP Basic；只能检查自己的 access/refresh token。无效、撤销或其他客户端的令牌返回 active=false，不消费刷新凭据。"},
	} {
		properties := map[string]*huma.Schema{"token": {Type: "string", WriteOnly: true}, "token_type_hint": protocolString("可选类型提示", "access_token", "refresh_token")}
		responses = errors()
		if endpoint.path == "/oauth/revoke" {
			properties["client_id"], properties["client_secret"] = protocolString("表单认证客户端 ID"), &huma.Schema{Type: "string", WriteOnly: true}
			responses["200"] = &huma.Response{Description: "撤销请求已处理，响应体为空。"}
		} else {
			responses["200"] = protocolJSON("非活跃只包含 active=false；活跃元数据按令牌类型返回，不返回个人资料。", protocolObject(map[string]*huma.Schema{
				"active": {Type: "boolean"}, "client_id": protocolString("令牌客户端"), "iss": protocolString("固定 issuer"),
				"sub": protocolString("用户或机器主体"), "scope": protocolString("批准权限"), "exp": {Type: "integer"},
				"aud": {Type: "array", Items: &huma.Schema{Type: "string"}}, "token_type": protocolString("令牌类型"),
			}, "active"))
		}
		op = add(http.MethodPost, endpoint.path, endpoint.id, endpoint.summary, endpoint.description, responses)
		op.RequestBody, op.Security = protocolForm(protocolObject(properties, "token")), []map[string][]string{{"OAuthClientBasic": {}}}
		if endpoint.path == "/oauth/revoke" {
			op.Security = append(op.Security, map[string][]string{})
		}
	}

	logout := protocolObject(map[string]*huma.Schema{
		"client_id": protocolString("客户端 ID；无 id_token_hint 时必填"), "id_token_hint": protocolString("本 issuer 签发且仍有效的 ID Token，可确定 client_id"),
		"post_logout_redirect_uri": protocolString("必须在已批准退出回调名单中精确匹配"), "state": protocolString("回跳关联值"), "ui_locales": protocolString("界面语言偏好"),
	})
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		responses = errors()
		responses["303"] = protocolRedirect()
		op = add(method, "/oauth/logout", "oauth-logout-"+strings.ToLower(method), "发起 RP 退出与中央确认", "必须提供 client_id 或 id_token_hint。先跳转确认页，由用户决定是否结束当前中央会话；不等于全站所有 RP/设备即时退出。", responses)
		if method == http.MethodGet {
			op.Parameters = protocolQuery(logout)
		} else {
			op.RequestBody = protocolForm(logout)
		}
	}
}

func ptrFloat(value float64) *float64 { return &value }
