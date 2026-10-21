# OAuth/OIDC 后端接入与验证

更新时间：2026-10-21。本文件描述已实现后端；前端授权页已接入，并完成 [独立 RP 的真实浏览器验证](oidc-interop.md)，包括刷新轮换与 RFC 7009 撤销。本批新增受限公开内省，RP 标准退出仍未实现。参数与事务决定见 [ADR-0015](decisions/0015-authorization-code-oidc.md)、[ADR-0018](decisions/0018-refresh-token-rotation.md) 和 [ADR-0023](decisions/0023-introspection-and-application-history.md)，跨对话交接见 [后端状态](coordination/backend.md)。

## 运行前置条件

沿用既有 Windows Go/Node/pnpm、`.env` 和 Ubuntu 专属依赖；不重复生成密钥或初始化数据库。迁移 3 新增 authorization.interactions；迁移 4 新增 authorization.grants 并关联旧协议记录，不改写迁移 1/2/3。API readyz 要求上述表存在。专属 Ubuntu 开发库已实际应用迁移 4；其他工作区按自己的目标库确认。

在仓库根目录 PowerShell：

```powershell
pnpm api:migrate
pnpm dev
```

若已有本项目 `pnpm dev`，不要启动第二份或结束其他对话的进程；API 若因缺迁移退出，迁移成功后在 TUI 选 api 按 r。`pnpm api:migrate` 使用 `.env` 的实际目标，执行前应确认是本项目开发库；不能将上述操作套到其他项目。日常账号创建见 [开发预览](live-development.md)。

## 已实现协议路径

| 方法与路径 | 当前能力 |
| --- | --- |
| GET `/.well-known/openid-configuration` | 固定 issuer 的真实元数据，仅声明已实现流程 |
| GET `/oauth/jwks` | RS256 公钥 |
| GET `/oauth/authorize` | Authorization Code + PKCE S256，创建授权交互并跳回 account |
| POST `/oauth/token` | form 编码；authorization_code、refresh_token 和 client_credentials |
| POST `/oauth/revoke` | form 编码；客户端认证及整族撤销，成功为空 200 |
| POST `/oauth/introspect` | form 编码；仅 confidential Basic 认证，查询自身 access/refresh 状态 |
| GET `/oauth/userinfo` | Authorization Bearer；要求 openid；返回 sub 和获准资料 |

issuer 来自配置，不从 Host 或代理头推导。Nuxt 需将 Discovery 转发到固定 Go 上游；现有 `/oauth/**` 转发也需保留。标准 OAuth/OIDC 应由 RP 使用标准客户端实现 state、PKCE、nonce 和 ID Token 验证，不把浏览器账号 Cookie 送给第三方。

本阶段没有 RP 标准退出、动态注册或已记忆同意。openid 必须带 nonce；授权码请求必须有精确 redirect_uri、PKCE S256 和单一允许 audience（仅有一个注册 audience 时可省略）。项目当前只接受 code/query，支持单值 prompt=login/consent，prompt=none 返回 interaction_required；max_age=0 要求本次授权之后重新登录。

## 刷新与撤销

客户端注册需允许 refresh_token grant 和 offline_access scope，授权时请求 openid/offline_access 并明确取得同意。只有该路径签发刷新令牌；默认公开客户端和普通登录不会因此自动获得长期权限。刷新请求为 `POST /oauth/token`，form 包含 grant_type=refresh_token、refresh_token 及对应客户端认证；公开客户端提交 client_id。成功使用标准 token 响应返回轮换后的 refresh_token。scope/audience 省略时沿用原授权，也可明确传入相同值；当前拒绝 scope 缩减和 audience 更换。

刷新必须串行，收到成功响应后原子替换凭据。旧令牌重用会撤销整个族，包括并发刷新中先成功的请求；没有重试宽限。普通刷新不主动撤销旧 access token；重用/协议撤销时，UserInfo 在线检查立即失效，下游离线 JWT 仍有剩余有效期窗口。开发时效与绝对期限仅在 [ADR-0018](decisions/0018-refresh-token-rotation.md) 定义，不当作生产批准值。

撤销请求为 `POST /oauth/revoke`，form 包含 token、可选 token_type_hint，以及客户端认证；可使用 HTTP Basic、client_secret_post 或公开客户端 client_id。有效、重复、未知或其他客户端所属 token 返回空 200；有效的本客户端 access/refresh（包括已消费 refresh）撤销整个族。认证失败是协议 invalid_client；存储故障为 temporarily_unavailable。成功撤销不结束自有用户站 Cookie 会话，也不代表已交付用户授权管理页。

## 在线令牌内省

按 [RFC 7662](https://www.rfc-editor.org/rfc/rfc7662.html) 使用 POST `/oauth/introspect`，Content-Type 为 application/x-www-form-urlencoded，body 含唯一非空 token，可选 token_type_hint。只接受 confidential client 的 HTTP Basic；ID/secret 按 OAuth Basic 规则编码，使用标准 OAuth 客户端库发送。不得把 secret 放入浏览器、查询参数或页面示例；public client、普通 Bearer、client_secret_post 和混用认证不受支持。Discovery 提供 introspection_endpoint 和仅含 client_secret_basic 的认证方式。

客户端只能查询自己的 access/refresh。有效响应含 active=true、client_id、iss、sub、scope、exp，可选 aud；access 另含 token_type=bearer。refresh 的 exp 是刷新令牌自身期限，不是 access 的到期时间。响应不含邮箱、昵称、security_version 或 token 原文。未知/伪造/过期/撤销/已消费/他人 token、ID Token 以及账号安全版本已变时统一只返回 `{"active":false}`；查询不会消费或撤销 refresh。

认证失败为 401 并含 Basic challenge；结构/编码错误为 400；存储、解密、身份依赖或限流服务失败为 503；协议 IP 限额用尽为 429。在线检查要求账号/应用/授权仍可用；机器凭据没有用户账号查询。所有内省响应 no-store，不在服务端缓存活跃结果，资源服务器应核对 sub/aud/scope/exp 并按所需实时性选择调用频率。已有离线 JWT 不会因这个新端点自动获得即时撤销能力。

此能力复用当前版本 9 数据模型，无新迁移、初始化或依赖。没有新增通用资源服务器身份或跨应用令牌查询权限；生产高频调用与独立限额仍待验收。

## 给前端的业务接口

入口跳到固定账号授权页 `/auth/authorize?authorization_request=<43字符随机句柄>`；旧首页事务入口由前端保留兼容。前端要跨身份登录保留这个句柄，但不能自动批准。详情和决定均依赖独立 HttpOnly 授权 Cookie；不得自己读取或生成 Cookie。用户站自身作为 RP 的登录与独立会话接线见 [自有 RP](run-first-party-oidc.md)，不能让授权页进入 RP 登录循环。

`GET /api/v1/account/authorizations/{id}`：

- 返回 client_id、client_name、redirect_uri、scopes、audiences、expires_at、login_required；已登录且可用时附 user（id/email/display_name）。
- 无登录时仍可查看绑定本浏览器的请求；user 省略、login_required=true。
- login_required=true 时先完成真实密码登录；prompt=login/max_age 可能要求已有会话重新认证。登录后重新读取详情，使用新的 CSRF。

`POST /api/v1/account/authorizations/{id}`：

```json
{"decision":"approve","subject_id":"<详情返回的 user.id>"}
```

decision 也可为 deny。使用现有 account Cookie、固定 Origin 和 `X-Moetopia-CSRF`，还需原浏览器授权 Cookie。成功返回 `{"redirect_to":"<标准回调>"}`。前端通过完整浏览器导航前往该地址；不要 fetch 回调、取出 code 展示，或自行拼接 state/回调。

| 状态 | 处理 |
| --- | --- |
| 401 | 登录失效，或 code=authorization_login_required；重新认证后重读详情 |
| 403 | Cookie/Origin/CSRF 校验失败；不自动批准或跳转 |
| 404 + authorization_request_unavailable | 不存在、已消费、过期、浏览器不匹配或客户端配置不再有效；让用户从 RP 重新发起 |
| 409 + authorization_account_changed | 当前账号与页面 subject_id 不同；重读详情并重新让用户确认 |
| 422 | 请求结构校验失败 |
| 429 | 频率限制；稍后重试 |
| 503 | 服务不可用；保留交互，不伪造授权成功 |

业务错误使用 Problem Details；以上新增 code 位于响应顶层。现有账号认证/CSRF 错误仍按原接口状态处理。准确类型由 [OpenAPI](../packages/api-client/openapi.json) 和 [生成类型](../packages/api-client/src/schema.ts) 提供，不手工修改生成文件。授权确认页应设置 no-store、Referrer-Policy: no-referrer 和禁止被嵌入的 CSP/frame-ancestors；这是前端服务响应的接入要求。

## 开发公开客户端

独立 RP 准备好自己的 loopback 回调后，才创建客户端。下列 53110 是显式示例端口，命令不启动回调服务，也不表示该端口已占用：

```powershell
node --env-file=.env scripts/dev-oauth-client.mjs -redirect-uri http://localhost:53110/callback
```

CLI 只允许 development、moetopia_dev 数据库、55432 端口；创建一个启用的公开客户端，无 secret，仅 authorization_code、openid/profile/email/demo.read、单一 `<issuer>/api` audience。输出公开 client_id、redirect_uri、issuer、audience。每次执行创建新客户端，不要反复运行冒充幂等更新；此入口不代表正式开发者审核机制。

需要测试刷新时显式加入 `-offline-access`，使该新客户端允许 refresh_token 和 offline_access；RP 也必须在授权时请求该 scope。本命令不修改旧客户端，也不创建预设测试账号：

```powershell
node --env-file=.env scripts/dev-oauth-client.mjs -redirect-uri http://localhost:53110/callback -offline-access
```

## 验证范围

```powershell
pnpm api:test
go -C apps/api vet ./...
pnpm test:integration
pnpm api:check:client
```

集成运行器创建独立随机测试库，完成迁移、协议与账号测试后清理，不拿开发库当测试库。后端测试不代替 Nuxt 代理、授权页操作、第三方 RP 浏览器互操作或生产负载验收。

真实代理与独立客户端使用 `pnpm test:oidc`，当前已通过；专属端口、临时构建、邮件及数据清理要求见 [独立客户端验证](oidc-interop.md)。协议代理必须保留上游重定向，由浏览器处理 303 与 Set-Cookie，不能在 Nuxt 服务端自动跟随。
