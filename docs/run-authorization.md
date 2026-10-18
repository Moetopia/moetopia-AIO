# OAuth/OIDC 后端接入与验证

更新时间：2026-10-18。本文件描述已实现后端；前端授权页由另一对话接入，不能据此认为浏览器端完整登录已可用。参数与事务决定见 [ADR-0015](decisions/0015-authorization-code-oidc.md)，跨对话交接见 [后端状态](coordination/backend.md)。

## 运行前置条件

沿用既有 Windows Go/Node/pnpm、`.env` 和 Ubuntu 专属依赖；不重复生成密钥或初始化数据库。迁移 3 新增 authorization.interactions，无其他 schema 修改，不改写迁移 1/2。API 的 readyz 现在要求该表存在。

在仓库根目录 PowerShell：

```powershell
pnpm api:migrate
pnpm dev
```

若已有本项目 `pnpm dev`，只执行迁移并等待 API 自动重建；不要启动第二份或结束其他对话的进程。`pnpm api:migrate` 使用 `.env` 的实际目标，执行前应确认是本项目开发库；不能将上述操作套到其他项目。日常账号创建见 [开发预览](live-development.md)。

## 已实现协议路径

| 方法与路径 | 当前能力 |
| --- | --- |
| GET `/.well-known/openid-configuration` | 固定 issuer 的真实元数据，仅声明已实现流程 |
| GET `/oauth/jwks` | RS256 公钥 |
| GET `/oauth/authorize` | Authorization Code + PKCE S256，创建授权交互并跳回 account |
| POST `/oauth/token` | form 编码；authorization_code 和 client_credentials |
| GET `/oauth/userinfo` | Authorization Bearer；要求 openid；返回 sub 和获准资料 |

issuer 来自配置，不从 Host 或代理头推导。Nuxt 需将 Discovery 转发到固定 Go 上游；现有 `/oauth/**` 转发也需保留。标准 OAuth/OIDC 应由 RP 使用标准客户端实现 state、PKCE、nonce 和 ID Token 验证，不把浏览器账号 Cookie 送给第三方。

本阶段没有 refresh_token、公开 revoke/introspect、RP 退出、动态注册或已记忆同意。openid 必须带 nonce；授权码请求必须有精确 redirect_uri、PKCE S256 和单一允许 audience（仅有一个注册 audience 时可省略）。project 当前只接受 code/query，支持单值 prompt=login/consent，prompt=none 返回 interaction_required；max_age=0 要求本次授权之后重新登录。

## 给前端的业务接口

入口跳到固定账号首页 `/?authorization_request=<43字符随机句柄>`。前端要跨登录保留这个句柄，但不能自动批准。详情和决定均依赖独立 HttpOnly 授权 Cookie；不得自己读取或生成 Cookie。

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

## 验证范围

```powershell
pnpm api:test
go -C apps/api vet ./...
pnpm test:integration
pnpm api:check:client
```

集成运行器创建独立随机测试库，完成迁移、协议与账号测试后清理，不拿开发库当测试库。后端测试不代替 Nuxt 代理、授权页操作、第三方 RP 浏览器互操作或生产负载验收。
