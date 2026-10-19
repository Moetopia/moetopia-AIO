# 自有用户站 OAuth/OIDC 接入

更新时间：2026-10-19。Go 登录、真实协议回调、独立会话与本地退出已实现；前端已接入普通页面入口、portal 上下文、登录守卫、会话及本地退出。真实 Chromium 回归已覆盖正常页面发起登录；独立管理员 RP/MFA 后端另见 [管理员接入](run-administration.md)。架构与开发参数见 [ADR-0019](decisions/0019-first-party-oidc.md)。

## 已完成的链路

用户中心与同源开发者中心共用受控 confidential RP `moetopia-user-site`：启动 → Authorization Code + S256/nonce/state → 身份认证与显式同意 → Go 回调兑换及验签 → 独立 HttpOnly RP Cookie → 固定站内目的页。身份提供者会话和 RP 会话不能互相充当登录证据。

| 用途 | 入口或 API | 说明 |
| --- | --- | --- |
| 用户站登录 | GET `/api/v1/portal/login?return_to=/profile` | 顶层浏览器导航，303 发起真实 OAuth；不能用 fetch 跟随 |
| 回调 | GET `/api/v1/portal/callback` | Go 专用，固定注册；前端不截获 code，不自行换码 |
| 用户站上下文 | GET `/api/v1/portal/context` | authenticated、csrf_token、login_url；已认证时带 user、expires_at |
| 用户站退出 | POST `/api/v1/portal/logout` | RP Cookie、固定 Origin 与 portal CSRF；返回匿名上下文并旋转 Cookie |
| 用户站会话列表 | GET `/api/v1/portal/sessions` | 只列当前用户的 RP 会话，不混入身份会话 |
| 结束用户站会话 | DELETE `/api/v1/portal/sessions/{id}` | RP Cookie、Origin、portal CSRF；检查用户和应用所有权 |
| 身份提供者登录/注册 | 原 `/api/v1/account/*` | 继续服务密码、注册及授权交互；不会直接建立 RP 会话 |
| 授权页 | `/auth/authorize?authorization_request=…` | Go 现在跳转专用 UI；旧首页事务入口仍由前端兼容 |

业务 DTO 来源为 [OpenAPI](../packages/api-client/openapi.json) 与 [生成类型](../packages/api-client/src/schema.ts)，共享包公开 PortalContext、PortalSessions 别名。登录/回调是重定向接口，路径与错误由本文件定义，不套业务 JSON 成功响应。

## 前端接线约定（已接入）

1. 用户中心、developer 导航和页面守卫从 portal/context 判断登录；不再把 account/context.user 当作 RP 已登录。匿名时完整导航到 portal/login，并传允许的 return_to。
2. 身份认证页和授权页继续使用原 account Cookie/CSRF/API，不进入 RP 守卫。它们的密码登录完成后重读授权详情并显式同意，不直接跳入用户中心或再次发起 RP 登录。
3. 资料、用户站会话及站点退出使用 portal 上下文。portal 的写请求使用其 csrf_token；身份登录、注册和授权确认仍用 account 的 csrf_token，不能交叉。
4. Go 回调成功自动跳转目的页；目的页重新获取 portal/context。前端不解析 ID Token，不保存 access/refresh token，不在 Nuxt 再建立另一套身份会话。
5. 用户站会话撤销后若结束当前会话，重读 portal/context；本地退出只结束 RP 会话，不表示中央身份或所有第三方站点已经退出。重新发起登录时可复用身份会话，但仍需显式同意。
6. 保留 `/api/**` 和 `/oauth/**` 的固定上游及 `fetchOptions.redirect = manual`。登录启动与回调依赖浏览器收到原始 303、Location、Set-Cookie；服务端跟随会丢失中间 Cookie。SSR 服务端请求如未来参与登录状态读取，需要显式转发该浏览器 Cookie，不缓存或序列化秘密。

允许的 return_to 只有 `/profile`、`/sessions`、`/security`、`/developer`、`/developer/applications`、`/developer/docs`；省略时 `/profile`。拒绝任意 URL、协议相对地址、query、fragment、认证/回调路径和非白名单路径，防止开放重定向及登录循环。

管理员没有复用这个 RP。admin RP、管理员角色和 TOTP/MFA 已独立实现，页面接线见管理员接入；不能用用户站已登录或隐藏菜单放行后台。

## 错误与恢复

| 状态/code | 含义与处理 |
| --- | --- |
| 400 rp_return_path_invalid | 启动目的页或 query 不合法；修正调用，不接受用户提供任意回跳 |
| 400 rp_callback_invalid | 回调结构不合法、参数重复、code/error 冲突或 iss 不匹配 |
| 400 rp_login_invalid | state/浏览器绑定缺失、已消费或过期；从登录启动入口重新发起 |
| 400 rp_authorization_denied | 用户未授权；不自动重试或批准 |
| 400 rp_exchange_failed / rp_identity_invalid | 兑换或完整 ID Token 校验失败；不能创建 RP 会话，从头登录 |
| 401 rp_identity_unavailable | 身份已不可用；不根据旧令牌继续进入用户站 |
| 503 rp_client_unavailable / rp_unavailable | 客户端配置、存储或服务故障；检查自己的 API 日志与初始化状态 |
| 429 | 启动/回调受到协议 IP 限流；稍后重新发起，不绕过保护 |

上下文/退出/会话业务错误继续使用既有 Problem Details。匿名 portal/context 返回 200、authenticated=false；依赖故障返回 503，不能伪装匿名成功。回调返回脱敏问题响应，绝不回显令牌、内部库错误或任意 error_description；暂未提供前端专用错误页。失败后事务已经消费的情况不能重用同一个回调。

## 运行与已执行迁移

2026-10-19 已核对 `.env` 指向专属 `192.168.101.41:55432/moetopia_dev` 并执行：

```powershell
pnpm api:migrate
pnpm api:first-party:init
```

迁移 5 新增 sessions.rp_logins 和到期索引，扩展 web_sessions.application 的 CHECK 允许 user-site；保留 account/admin/developer 旧行。已有账号、会话与签名密钥未重建，未修改已执行迁移 1–4。API readyz 要求新表存在。其他工作区确认自己的目标库后按相同顺序运行，不重复初始化密钥；若 API 已因缺迁移退出，在自己的 TUI 选 api 按 r。

初始化是显式受控操作，重复执行只校验已有客户端；配置或秘密不匹配、被停用时失败，不覆盖配置或重新启用。secret 从既有加密根密钥按专用用途派生，数据库保存 bcrypt 验证值，终端不输出 secret。正式密钥轮换需为此安排客户端凭据迁移，不能直接替换根密钥。

迁移会修改 CHECK 约束并取得表锁，开发库已执行不代表生产锁影响已经验收。Down 删除 RP 登录事务并结束 user-site 会话，保留其他应用会话；受控客户端由显式初始化创建，不由 Down 自动删除。禁止在新版仍运行时直接回退；旧程序没有 RP 能力，不能靠回退程序维持这些会话。

当前可从 [开发用户站](http://account.localhost:3000/) 的正常登录入口或 [直接发起 RP 登录](http://account.localhost:3000/api/v1/portal/login?return_to=/developer/applications) 验证链路。请继续使用 account.localhost，不用 localhost/127.0.0.1 替代配置 Origin。

## 验证事实与边界

- Go test/vet、真实专属 PostgreSQL/Valkey 集成与 OpenAPI 漂移检查通过。覆盖身份与 RP Cookie/CSRF 隔离、固定目的页、浏览器绑定、state、过期/拒绝/重放、并发单次回调、错误签名/issuer/audience/nonce/exp/iat/azp/auth_time/at_hash、会话列表/退出、停用客户端及初始化不自动恢复。
- `pnpm test:oidc` 的 1 项真实 Chromium 场景通过，保留独立 openid-client 的原有协议验证，新增 Go 自有 RP 经 Nitro 启动、授权、回调、恢复开发者目的页、Cookie 和本地退出验证。使用临时数据库、测试邮件和独立端口/构建，运行器清理；没有用 mock 替代协议响应。
- Go OIDC 使用锁定的 coreos/go-oidc v3.21.0 与 x/oauth2；govulncheck v1.9.0 复扫 0 个可达漏洞，保留既有模块级告警，见 [依赖审计](dependency-audit.md)。

2026-10-19 前端接入已由本后端再次运行 `pnpm test:oidc` 验证，保留 portal-ui 正常 developer 页面入口、同意、回调、会话和退出的浏览器检查。账号管理新增契约见 [账号安全](run-account-management.md) 和 [开发者应用](run-developer-applications.md)。管理员协议/MFA 由独立 `pnpm test:admin` 验证，成品后台接线另计；匿名用户从 RP 全流程进入身份注册的专用回归、中央退出、正式部署及性能验收仍未完成，不代表完整身份产品或 OIDC 认证。
