# 后端进度与前端接入

更新时间：2026-10-19。所有者：本后端对话。**当前接入状态优先读文末“账号安全与开发者管理整批交接”**；前文保留历史轮次。分工基线见 [workstreams](../workstreams.md)，同步方式见 [本目录说明](README.md)。本轮使用共享文档交接，不声称前端已收到自动通知。

## 已完成

- 原有账号接口兼容：邮箱验证、邀请注册、密码登录、context/CSRF、退出和会话列表/撤销。
- 新增 Authorization Code + PKCE S256、OIDC RS256 ID Token、Discovery、UserInfo 后端。
- 新增授权详情/批准/拒绝业务接口，OpenAPI 与 TypeScript 类型已生成。
- 授权浏览器 Cookie 与 account 会话 Cookie 分开；原子消费交互、代码/PKCE/OIDC 状态，错误 verifier 可重试，代码重放撤销旧 access 状态。
- 迁移 3 新增 authorization.interactions；没有修改迁移 1/2。专属 Ubuntu 开发库已应用迁移 3，不重建数据、不轮换密钥、不修改远程容器配置。
- 后续已完成 refresh_token/offline_access、严格轮换重用检测及 RFC 7009 `/oauth/revoke`；迁移 4 已应用。新状态及限制见文末。

## 前端接入契约（现已接入）

完整字段、错误、运行步骤与限制见 [run-authorization.md](../run-authorization.md)，实现理由见 [ADR-0015](../decisions/0015-authorization-code-oidc.md)。请按生成包类型接入，不创建第二份 DTO。

1. Nuxt 增加 `/.well-known/openid-configuration` 到固定 Go 上游的代理；已有 `/oauth/**` 保留，不能从 Host 或用户参数选择上游。
2. account 首页读取并保留 authorization_request；GET `/api/v1/account/authorizations/{id}` 展示真实应用、回调、权限、账号，不自动批准。
3. 根据 login_required 完成或重新完成密码登录；登录后重读详情和 CSRF。事务 Cookie 不随登录旋转丢失。
4. POST 同一路径，body 为 decision（approve/deny）和 subject_id（详情的 user.id），继续带 Cookie、Origin、X-Moetopia-CSRF。
5. 成功使用完整浏览器导航前往 redirect_to。处理 401/403/404/409/422/429/503；过期或已消费从 RP 重启，账号变化重读详情并重新确认。
6. 授权页采用 no-store、no-referrer 及禁止嵌入策略；补充真实授权详情、拒绝、过期、账号切换、强制登录的页面测试。前端 mock 只能做页面验证，不声称真实协议互操作。

## 验证证据与限制

Go 测试与 vet 已通过。真实 Ubuntu PostgreSQL/Valkey 的专属临时库集成通过，覆盖独立 JWT 验签/issuer/audience/sub/nonce/at_hash、资料 scope、错误 verifier 回滚、并发单次兑换、重放撤销、浏览器 Cookie/Origin/CSRF、拒绝/过期/重复确认、账号切换、强制认证、停用用户/客户端及只有一个数据库连接时的兑换。临时库已由运行器清理。

最初后端轮次未改前端。随后前端已完成授权页和 Discovery 代理；本次真实互操作已通过，细节和两处共享接线修复见下文。本对话未修改页面样式、UI 或结束其他对话的开发进程。

仍没有公开 introspect、RP 退出、自动记忆同意、开发者审核、管理员 MFA、生产邮件或生产发布。Discovery 只声明当前实现能力。开发公开客户端 CLI 已编译；未自动创建默认客户端，待 RP 提供回调后按运行说明显式创建。

## 后续后端事项

2026-10-18 启动器增量：用户要求优化混合日志，已将 pnpm dev 改为 mprocs TUI，提供 dev:plain 和按服务选择；更新了根 package.json、pnpm-lock.yaml、scripts/dev* 及开发运行文档，未修改页面。各对话可以运行 pnpm dev api / pnpm dev account，但不能重复启动同一服务；完整键位与行为见 [开发预览](../live-development.md)。本轮 Windows 四服务启动、单独停止/重启 account、TUI 退出及端口释放已实测，不代表身份业务重新验收。

本轮检查发现工作区现在已有 Git 元数据（main 尚无首个提交，存在大量预先暂存与未暂存内容），取代早期“没有 Git”的环境快照；本对话未初始化 Git、调整暂存区或提交。

以上是启动器轮次的 Git 快照；本次互操作开工时 main 已有 HEAD，保留用户/其他对话既有暂存和提交，本对话仍未暂存或提交。

## 真实 RP 互操作增量

已读取 frontend.md，完成其交回的真实浏览器联调。新增 `pnpm test:oidc`，独立 openid-client 6.8.8 + Chromium + 专属 Go/PostgreSQL/Mailpit，覆盖真实邀请邮件注册、Cookie 旋转、显式批准/拒绝、prompt=login、JWKS 验签、UserInfo、重放撤销及 RP 错误 state/nonce/签名拒绝；最终 1 项端到端场景通过。Go 测试/vet、account 类型检查通过，三个测试端口已释放。

前端请保留以下联调修复：

- `packages/locales/src/index.ts` 恢复 `authorizationMessages` 的公开导出；当前源码与前端先前“已恢复”的记录不一致，未确定移除来源，不能覆盖回缺失状态。
- `packages/portal/nuxt.config.ts` 的 `/oauth/**` 代理使用对象配置及 `fetchOptions.redirect = manual`。原先代理在服务端跟随跳转，浏览器拿不到中间事务 Cookie，真实授权详情返回 404；必须让浏览器接收原始 303/Location/Set-Cookie。

未改页面、组件或 frontend.md；业务契约无新增字段。测试源码、端口/产物隔离、验证限制见 [独立客户端验证](../oidc-interop.md)。无新迁移，开发库未改动，未创建默认客户端或停止开发实例。新增根开发依赖和 test:oidc 命令；测试用程序化 account 构建不覆盖日常 .nuxt/.output。

本轮互操作后的刷新轮换及撤销已完成，证据见下一节；后续是账号恢复/资料变更、开发者配置版本审核和管理员 MFA。生产 TTL、正式 issuer、密钥轮换与退出策略仍需发布前验收，不能将本轮互操作称为完整身份产品交付。

## 刷新与撤销增量

已读取前端用户站合并交接，保留 account/admin 两应用与 `/auth/*`、`/developer/*` 路由，不恢复旧 developer 服务。本轮只修改后端、独立 RP 测试设施及对应文档。

- 新增 offline_access、refresh_token、严格轮换重用检测、整族撤销及 RFC 7009 端点；Discovery 增加相应元数据。Go 会话 Cookie 与账号 API 不变，无新 Huma DTO 或 TypeScript 字段。
- 用户站自己的登录继续用现有 Cookie；无需为此保存或刷新 OAuth token。授权页可能收到 offline_access，需表达“离线访问/保持授权”的真实含义；本轮未改其翻译，当前可安全展示原 scope，文案增强由前端维护。不暗示用户站授权管理功能已开放。
- 迁移 4 新建 authorization.grants，已在专属开发库实际执行。若自己的 TUI 中 api 因缺迁移而 DOWN，选 api 按 r；不重建密钥/账号、不停止另一会话的服务。
- Go test/vet、专属 PG/Valkey 集成、API 契约检查及扩展的 `pnpm test:oidc` 通过。验证包含完整凭据验真与客户端隔离、Basic 认证失败 header、绝对时效、并发重用、签发失败回滚及旧代码重放撤销刷新后代。独立 RP 通过真实轮换、旧令牌重用和撤销；测试资源由运行器清理。
- 没有固定测试账号或默认管理员。`pnpm dev:invite` 创建一次性邀请码，注册页 `/auth/register`，Mailpit `http://192.168.101.41:58025` 取邮箱验证码；本轮确认其 API 返回 200。测试库账号不保留为日常账号。

当前策略、开发时效和严格并发限制见 [ADR-0018](../decisions/0018-refresh-token-rotation.md)，客户端请求方式见 [运行说明](../run-authorization.md)。生产邮件、公开内省、RP 退出、应用审核、管理员 MFA 与恢复业务仍未完成；没有自动通知其他对话。

## 开发验证码 403 排查

用户报告发送验证码返回 Request verification failed。已在当前运行的 API/Nuxt 和独立 Chromium 中复现：同样有效的匿名 Cookie/CSRF，使用 account.localhost:3000 通过保护层（故意无效邮箱返回 422），localhost:3000 返回 403。用户随后确认实际访问 127.0.0.1:3000，与唯一配置 Origin http://account.localhost:3000 不同。没有放宽 Origin、CSRF 或邮件保护；scripts/dev-service.mjs 增加明确的配置入口及别名提示，运行指南补充此排查。未修改页面或 frontend.md。

## 2026-10-19：用户站 Go RP 完整后端交接

已读取前端最新交接及 ADR-0019，按用户新要求实现用户中心/developer 共用 RP。登录启动、固定回调、PKCE/state/nonce、ID Token/JWKS 验证、独立会话、上下文、会话列表/撤销与本地退出已经一起交付并验证。此轮之后前端已完成普通入口/守卫接线，见文末；不再把 account Cookie 直登视为最终路径。

**前端请以 [run-first-party-oidc.md](../run-first-party-oidc.md) 为本轮完整接线清单。** 新接口是 `/api/v1/portal/context`、`login`、`callback`、`logout`、`sessions` 和 `sessions/{id}`。生成包新增 PortalContext/PortalSessions，字段由 Huma/OpenAPI 生成。用户站使用 portal Cookie/CSRF；身份密码、注册及授权继续使用 account Cookie/CSRF，认证/授权页不能受 RP 守卫保护。Go 授权 UI 已改为 `/auth/authorize`；前端旧首页入口兼容可保留。

保留本轮共享接线：`packages/portal/nuxt.config.ts` 的 `/api/**` 与 `/oauth/**` 均为固定上游对象配置，带 `fetchOptions.redirect = manual`。真实浏览器测试证实 `/api/**` 服务端跟随会让用户站登录丢失中间 Cookie；不能覆盖回字符串代理。测试构建继续只替换端口并保留源代理选项。没有修改页面、UI、翻译、前端状态文件或停止其他会话进程。

迁移 5 已于 2026-10-19 实际应用专属开发库；新增 sessions.rp_logins，扩展会话 application CHECK，保留旧账号、会话、密钥。`pnpm api:first-party:init` 已执行，固定 confidential 用户站客户端已创建；重复执行只校验，不恢复已停用客户端、不覆盖配置或输出 secret。其他工作区先确认目标库，再迁移和初始化；API 因缺迁移 DOWN 时在自己的 TUI 选 api 按 r。

验证：Go test/vet、PG/Valkey 专属库集成、OpenAPI 漂移检查通过；`pnpm test:oidc` 1 项 Chromium 场景通过，包含真实用户站 Go RP 发起/同意/回调/返回 developer 目的页和本地退出，不以 mock 代替协议。错误 state/浏览器绑定、过期、拒绝、回调重放/并发、签名/issuer/audience/nonce/时效/azp/auth_time/at_hash 及 Cookie/CSRF 隔离均有后端测试。govulncheck 0 个可达漏洞，模块级既有告警保留。测试自己的数据库/进程/邮件由运行器清理。

当前可从 `http://account.localhost:3000/api/v1/portal/login?return_to=/developer/applications` 直接试后端链路；正常页面接线及账号/开发者新增能力由下一节更新。此处是共享文件交接，不代表另一对话已自动读取。

## 2026-10-19：账号安全与开发者管理整批交接

已读取 frontend.md 中普通页面 Go RP 接入及剩余能力清单。前端正常登录入口、portal 上下文/守卫、会话/本地退出已完成，本后端重新运行真实 `pnpm test:oidc`，包含 portal-ui 正常 developer 页面流程并通过；不再记录成“待前端切换”。保留认证/授权页身份上下文与普通页 RP 上下文隔离、Nitro manual redirect 配置和浏览器测试辅助代码。

**以下接口与生成类型已可成批接入，无需等待后端逐项开放：**

| 页面/操作 | 当前契约 |
| --- | --- |
| 资料、简介、头像上传/删除 | PATCH portal/profile、PUT portal/avatar；携带 User.revision |
| 改密、新邮箱验证/确认 | POST portal/password、portal/email-changes 及 confirm；当前密码必需 |
| 忘记/重置密码 | POST account/password-resets 及 confirm；身份上下文，不走 RP 守卫 |
| 已授权记录与撤销 | GET/DELETE portal/authorizations；列表每项是授权族，不按应用聚合 |
| 个人安全记录 | GET portal/security-events；最近 90 天最多 50 条 |
| 应用创建/详情/列表 | `/api/v1/developer/applications` 及 `/{id}` |
| 草稿编辑/提交/停启/密钥/删除 | 同一应用下 draft、submit、availability、secret 及 DELETE |

表中 portal/account 业务路径完整前缀为 `/api/v1/`。字段、状态码、输入限额和交互顺序见 [账号管理](../run-account-management.md) 与 [开发者应用](../run-developer-applications.md)，请按生成包别名和 paths 使用，不复制 DTO。User 新增 bio/avatar_url/revision；新字段在类型中保留兼容性，真实后端响应已包含。

特别注意：改密/邮箱确认/恢复会让全部旧会话失效，成功后清理页面旧状态并重新登录；恢复开始成功不表示账号存在或邮件必然发送。邮箱挑战绑定发起 RP session，不能换设备确认。头像接受纯 Base64 JPEG/PNG，不接受 data URL/SVG；空字符串删除。新 secret 只展示本次，轮换后重读详情取得新 revision，不能持久化或自动重试轮换。列表 next_cursor 用空字符串表示末页，approved 未发布时省略。

应用创建后已能由所有者测试实际 OAuth。提交将进入 pending；**目前没有管理员 HTTP 审核入口**，不要制作假批准或调用内部 Review 绕过权限。发布版编辑/待审期间旧配置继续服务；测试成员、平台暂停、历史列表、统计和通知尚未开放。开发基线配额及保守重新授权策略见运行文档。

迁移 6/7 已实际应用专属开发库，原账号/密钥保留；显式主体回填已执行 updated=0。API readyz 要求新表，若自己的 TUI 中 api 已 DOWN，选 api 按 r；不重新初始化密钥/数据库。其他工作区运行前先核对目标，再用已有 api:migrate 和 api:grants:backfill，不能把随机测试库账号当开发账号。

验证：Go test/vet、真实 PostgreSQL/Valkey 随机库集成、全工作区类型检查与 OpenAPI 漂移检查通过；真实 Chromium OIDC 1 项场景通过。账号/开发者新页面尚未在浏览器接入验收，本轮未修改 UI、翻译或 frontend.md，也未停止其他会话进程。完整状态、缺口和安全扫描范围见 [实施进度](../implementation-status.md)。后端余项包括管理员完整安全链路/审核、注销、联邦、标准退出/内省及生产验收。
