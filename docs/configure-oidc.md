# 配置 OAuth 2.0 / OpenID Connect

Moetopia 可以作为其他站点的 OpenID Provider。以下配置面向使用 Moetopia 登录的应用。若要让 Moetopia 接受 GitHub 或 Google 登录，见 [上游身份供应商配置](run-federation.md)。

## 配置 Moetopia

- 本地开发使用 `MOETOPIA_ISSUER=http://account.localhost:3000`，并让 `MOETOPIA_ACCOUNT_ORIGIN` 使用同一个地址。浏览器和应用服务端都要能访问 issuer。Go 内部监听地址和数据库地址不填入客户端。
- 生产 Compose 设置 `.env.production` 中的 `MOETOPIA_ACCOUNT_HOST`，值仅填域名，例如 `auth.example.com`。Compose 据此设置 `MOETOPIA_ISSUER=https://auth.example.com`。完成 [首次安装](run-production.md#首次安装) 的迁移、密钥初始化和入口配置后再登记客户端。issuer 是持久身份标识，上线前选定稳定域名。
- 用客户端实际访问的地址检查发现文档。返回的 `issuer` 必须与客户端配置逐字一致，包含协议、域名及实际端口。OIDC 客户端填 issuer 根地址，只有明确要求 Metadata/Discovery URL 的字段才填完整发现地址。RFC 8414 客户端可以使用第二条。 [OIDC Discovery](https://openid.net/specs/openid-connect-discovery-1_0.html) · [RFC 8414](https://www.rfc-editor.org/rfc/rfc8414.html)

```powershell
$moetopiaIssuer = 'http://account.localhost:3000'
Invoke-RestMethod "$moetopiaIssuer/.well-known/openid-configuration"
Invoke-RestMethod "$moetopiaIssuer/.well-known/oauth-authorization-server"
```

## 登记一个应用

1. 登录 Moetopia 用户站的 **开发者中心 → 应用**，创建测试应用。首次测试使用应用所有者或[已添加的测试成员](run-test-members.md)。
2. 从应用自己的 OIDC 设置复制它给出的完整登录回调，例如 `https://app.example.com/auth/callback`，粘贴到 Moetopia 的登录回调列表。协议、主机、端口、路径及末尾斜杠需要精确一致。开发环境的 HTTP 仅允许环回主机，生产使用 HTTPS。
3. 有后端、能保管密钥的 Web 应用选 **Confidential**。保存创建时显示一次的 Client Secret，放入应用服务端的秘密配置。不能保管密钥的客户端选 **Public**，以无密钥的方式换码。
4. 勾选应用实际需要的 scope。普通登录从 `openid profile email` 开始。仅在应用需要持续代表用户访问时启用并请求 `offline_access`，由用户明确同意。
5. 测通浏览器跳转、登录、同意、回调和本地会话后，提交应用审核。首次通过后普通用户才可使用。已发布应用的新回调或权限等待审核时，继续使用旧批准配置。 [应用管理与审核](run-developer-applications.md)

## 填写应用的通用 OIDC 设置

| 客户端字段 | 填写值 |
| --- | --- |
| Provider 类型 | Generic OpenID Connect / Custom OIDC |
| Issuer | Moetopia 用户站公开 origin，例如 `https://auth.example.com` |
| Client ID | 开发者中心显示的 OAuth Client ID |
| Client Secret / Token endpoint authentication | Confidential 使用服务端 secret，优先选 `client_secret_basic`。Public 留空并选择 `none` |
| Grant / Response type / Response mode | `authorization_code` / `code` / `query` |
| PKCE / State / Nonce | `S256`，每次登录由客户端库生成并验证对应证明。OIDC 请求必须发送 nonce |
| Scopes | `openid profile email`，按需增加 `offline_access` |
| ID Token 校验 | 用 Discovery 的 JWKS 验证 RS256 签名，再核验 `iss`、`aud`、时效和 nonce |
| 用户主键 | 使用 `(iss, sub)`；邮箱和显示名可变。读取 UserInfo 时也要核对其 `sub` |
| API audience | 开发者应用默认 `<issuer>/api`。只有一个已登记 audience 时请求可省略该参数。ID Token 的 audience 是 Client ID |
| 中央退出 | 使用 Discovery 的 `end_session_endpoint`，需要回跳时单独登记退出回调。RP 仍需负责结束自己的应用会话 |

客户端若只能手填端点，从 Discovery 复制 `authorization_endpoint`、`token_endpoint`、`userinfo_endpoint`、`jwks_uri` 和所需的退出/撤销/内省地址。申请刷新后，每次刷新成功都要原子保存新的 refresh token，并串行刷新。访问下游 API 使用 access token，建立应用登录会话使用经校验的 ID Token。 [OIDC Core](https://openid.net/specs/openid-connect-core-1_0.html) · [OAuth 安全最佳实践](https://www.rfc-editor.org/rfc/rfc9700.html)

排查时先核对 issuer 和代理转发，再核对应用的生效版本、测试成员资格、完整回调、scope 及认证方式。缺少 PKCE 或 nonce 的旧客户端需要升级。可在用户站 `/developer/docs` 查接口详情，或运行隔离的 [独立客户端测试](oidc-interop.md)。

## 协议与流程覆盖

| 能力 | 当前状态与适配方向 |
| --- | --- |
| OAuth 2.0 授权码 + PKCE、OIDC 1.0 | 已支持 Discovery、RS256/JWKS、ID Token、UserInfo。Web 应用可通过开发者中心接入。 |
| OAuth 服务发现 | OIDC 地址与 RFC 8414 默认地址共用真实元数据。 |
| 刷新、撤销、内省 | 支持刷新轮换、RFC 7009 撤销、RFC 7662 内省。内省仅限 Confidential Basic 认证且只查本客户端的令牌。详见 [协议接入](run-authorization.md)。 |
| OIDC RP 发起退出 | 支持中央确认及受审回跳。中央退出的影响范围是当前浏览器的中央会话。Front-Channel 与 Back-Channel 退出未接入。详见 [退出契约](run-logout.md)。 |
| OAuth 2.0 Client Credentials | 服务端已有机器令牌实现和集成测试；开发者中心暂未开放这类客户端。需要服务间调用时，优先设计独立机器身份、资源权限和凭据管理。 [RFC 6749 §4.4](https://www.rfc-editor.org/rfc/rfc6749.html#section-4.4) |
| CLI、原生应用和无键盘设备 | 暂未完整适配原生应用的动态环回端口等规则，也未开放 Device Authorization。明确有 CLI、电视等接入时再按使用场景补齐。 [RFC 8252](https://www.rfc-editor.org/rfc/rfc8252.html) · [RFC 8628](https://www.rfc-editor.org/rfc/rfc8628.html) |
| Resource Indicators、Token Exchange | 暂未接入。当前使用项目的 `audience` 参数；多业务 API 出现独立资源标识或委托调用需求时再评估。 [RFC 8707](https://www.rfc-editor.org/rfc/rfc8707.html) · [RFC 8693](https://www.rfc-editor.org/rfc/rfc8693.html) |
| SAML 2.0 / SCIM 2.0 | 首期未提供。SAML 用于企业联邦单点登录，SCIM 用于账号和群组供应。由具体企业接入需求驱动。 [OASIS SAML](https://docs.oasis-open.org/security/saml/v2.0/saml-core-2.0-os.pdf) · [RFC 7644](https://www.rfc-editor.org/rfc/rfc7644.html) |
| 隐式授权、密码授权 | 保持关闭，旧客户端迁移到授权码 + PKCE。 [OAuth 安全最佳实践](https://www.rfc-editor.org/rfc/rfc9700.html) |

开发者自助创建与发布沿用现有审核流程，当前不提供 OAuth 动态客户端注册。兼容性以目标部署的发现文档及实际登录测试为准。
