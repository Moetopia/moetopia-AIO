# 实施进度

2026-10-18 用户要求开始执行功能实施计划。本文件记录实际完成状态，设计文档不代替实现证据。

## 已授权基线

Nuxt/Vue/Reka/Tailwind/Storybook，Go/chi/Fosite/Huma，PostgreSQL/Ent/Goose，Valkey。账号默认邀请注册，JWT access token，管理员 MFA，应用版本审核，七天注销冷静期，审计保留 90 天。已发布应用所有者需先停用应用再申请注销。

## 阶段

| 阶段 | 状态 |
| --- | --- |
| 工程、依赖与集成验证 | 工程已建立，基础测试及构建通过；Ubuntu PG/Valkey 真实集成测试通过 |
| 共享设计系统 | 常用组件、页框/认证/工作区布局、角色配色及 Storybook 已有；清单与限制由 UI 包说明维护 |
| 账号与联邦登录 | 邀请注册、邮箱验证、密码登录及 account 会话管理已完成开发路径；联邦登录、密码恢复与资料修改待完成 |
| OAuth/OIDC 与独立客户端互操作 | 授权码/PKCE/OIDC 后端与真实数据库集成通过；授权页、独立 RP 浏览器互操作及刷新/撤销待完成 |
| 管理后台与开发者审核 | 待完成 |
| 注销、恢复及数据清理 | 待完成 |
| 端到端、性能与生产部署包 | 待完成 |

## 环境事实

Go、Node.js 与 pnpm 可用。当前目录不是 Git 仓库，未初始化、提交或推送。Windows 未安装 Docker；用户随后提供 Ubuntu `192.168.101.41`，已在 `/home/misaka10843/code-stack/moetopia-dev` 启动专属 Compose。网络 `moetopia-dev_dependencies`，端口为 55432/56379/51025/58025。未复用该主机其他项目网络、数据库或容器。

Go module 使用本地命名空间 `moetopia.local/aio/api`，不假定尚未提供的 Git remote 或正式域名。发布 Go 包之前再按实际仓库调整。

## 验证证据

- Go 基础单元测试与 `go vet` 已通过。
- Ent 代码生成已完成，authorization schema 的模型映射与真实持久化测试通过；其他领域尚无模型，不作泛化结论。
- 三个 Nuxt 应用及共享 UI 类型检查通过，account 服务端构建与 admin/developer 静态构建通过。
- Storybook 生产构建通过。
- 6 项 Playwright 浏览器测试通过，覆盖三个页面壳的语言/主题持久化、无后端状态、窄屏、弹窗焦点及输入错误关联。
- OpenAPI/TypeScript 生成及漂移检查通过。
- 带 integration tag 的测试已在 Windows 连接 Ubuntu PostgreSQL/Valkey 执行并通过，覆盖迁移、密钥重开连接恢复、重复初始化拒绝、JWT 签发与状态撤销、非法 scope/audience/凭据。临时测试库已清理。
- 本项目开发库已应用版本 1 迁移并初始化签名密钥。真实 API 的 healthz、readyz、业务状态与 JWKS 返回 200；检查后停止了本轮 API，依赖容器保留运行。
- 实测修复了 SQL 关键字 authorization 的引用；expires_in 接受取整后的 599–600 秒，同时检查 JWT 的 exp/iat 十分钟差值。
- CI workflow 已编写但未在 GitHub 运行，未创建生产部署包。
- 依赖审计修复了 Go 可达告警及 glob 问题；仍有两项 npm 上游未修复告警，详见 [依赖审计](dependency-audit.md)，不宣称安全检查全通过。

基础阶段的运行顺序与具体命令见 [run-foundation.md](run-foundation.md)。本节原始验证只覆盖 client_credentials；后续授权码/OIDC 后端增量见文末和 [授权接入](run-authorization.md)，仍没有内置默认客户端凭据。

## 账号与实时预览阶段

- 迁移 2 已应用于专属开发库，新增 identity、sessions、notifications schema；未改写已执行的迁移 1 或旧 authorization 表。
- 注册默认邀请制，邮箱验证码通过加密 outbox 发往 Mailpit；邀请码、验证码消费和创建用户在同一 identity 事务完成。
- 密码使用带参数的 Argon2id 哈希，限定并发。开发邀请 CLI 默认 24 小时有效，只允许 development 环境。
- account 使用 Go 管理的 HttpOnly、Host-only Cookie；登录旋转句柄和 CSRF，写请求检查固定 Origin，支持会话列表、按所有权撤销及退出。没有把该会话当作 admin 权限或 OAuth 令牌。
- Go 单元/静态检查、真实数据库集成和 API 契约检查通过，增加了邀请码并发单次消费、验证码错误上限、CSRF、会话轮换、所有权和退出测试。Huma 校验错误不回显输入密码或邀请码。
- 三个前端构建、类型检查与 Storybook 构建通过；6 项基础浏览器测试通过。另有 1 项独立浏览器测试，通过真实 Mailpit 邮件完成邀请注册、密码登录、刷新保持和退出失效；使用临时数据库并清理本轮测试邮件。
- `pnpm dev` 在本机实际启动并验证，修复了独立 pnpm.exe 启动问题，加入端口占用检测、Go 自动重建及 Nuxt 热更新。测试端口与日常开发端口分开，不停止归属不明的占用进程。

日常自行运行见 [live-development.md](live-development.md)，账号参数与事务边界见 [ADR-0014](decisions/0014-account-preview.md)。当前 SMTP 仅适配开发 Mailpit，production 启动明确拒绝；审计、密码恢复、生产邮件、OIDC、管理员和开发者业务尚未完成，不宣称完整身份产品已交付。

## UI 反馈后的页面迭代（历史）

2026-10-18 根据用户对通用模板感的反馈，参考本地 kungal/nextmoe 的页面源码，重排登录与注册页，改为淡蓝强调色、紧凑表单、下划线切换、输入框内密码显隐及并列验证码操作；共享控件和另外两个入口同步更新。移除大标语、宣传区和整页圆角卡片。插画由用户另行处理，只提供可选接入位置，不生成或复制角色素材。

三个应用和 Storybook 构建、类型检查、8 项基础浏览器测试已通过，包括手机英文深色布局与输入说明关联。配色和页面属于当前评审稿，不能将实现或自动检查通过记作用户已经认可视觉。

## 组件评审与首轮页面接入

用户随后否定上述页面效果，要求先完成组件。当前基于 KunUI 源码参考完成常用控件，并按用户提供的看板娘将配色调整为薄荷绿、雾蓝和深蓝灰；组件契约、清单和限制见 [UI 包说明](../packages/ui/README.md)。

2026-10-18 用户明确要求开始页面，并要求复用界面直接进入 UI 库。规则已写入 [前端规范](frontend.md#可复用界面的归属)，AGENTS 链接该定义。本轮新增 PageFrame、AuthLayout、WorkspaceLayout、PageHeading、SettingRow、DescriptionList、StatusIndicator，均有 Storybook；页面不再维护独立的顶栏、页框、标签切换和密码显隐样式。立绘通过 UI 资源入口共用，原图未重绘。

- account：登录与邀请注册使用插画布局；登录后进入资料与会话工作区，支持原有验证码、退出与会话结束操作。账号状态由页面级 composable 维护，切换布局不重新初始化账号请求。资料仍只读。
- admin/developer：使用共享顶栏、响应式侧栏和内容布局。开发者入口提供静态接入准备信息；管理页说明访问控制和功能开放前提。没有创建虚构应用、用户数量、审核记录或管理员身份，管理与开发者业务仍未接入。
- 共享偏好由 portal 层处理 Cookie/主题/语言；所有可复用视觉和通用交互进入 UI 包。中英文及明暗主题继续可用。
- 本轮三应用与 Storybook 构建、全项目类型检查通过；24 项独立组件检查、13 项页面浏览器检查通过。页面检查使用模拟 API，覆盖登录布局切换、注册验证码请求与语言、退出、会话刷新/撤销、移动导航和无后端状态；本轮没有重新运行真实数据库与 Mailpit 端到端测试。
- 已查看桌面明暗主题与窄屏页面，页面为当前评审实现。原始立绘约 1.61 MB，尚未制作生产图片变体；认证布局窄屏隐藏插画并使用原生延迟加载，不将现有原图当作生产资源预算验收结果。

日常查看地址及跨应用链接配置见 [边开发边查看](live-development.md)。本轮未修改 Go、数据库结构、迁移或远程依赖。

## 页面继续迭代与对话交接

2026-10-18 用户最新要求暂不使用正面立绘。此更新取代上一节的页面插画接入：三个应用和默认 AuthLayout 故事均不加载候选图，只保留薄荷、雾蓝和深蓝灰配色；原图仅在明确的品牌参考故事展示。

- account 新增独立 `/sessions` 和 `/security` 页面，`/` 保留只读资料。导航使用 Nuxt 路由，支持直接打开和浏览器历史；安全页提供可用的会话入口，其余缺失能力明确标为未开放。
- 账号状态改由应用入口创建并通过类型化注入提供给页面，站内切换不重复初始化账号请求；原有登录、邀请注册、验证码、退出与会话撤销继续复用原接口。
- 三应用构建、Storybook 构建、全项目类型检查通过；24 项组件检查与 14 项页面浏览器检查通过，包含无插画布局、直接打开安全页、会话导航及跨页面状态保持。页面检查使用模拟 API；本轮未重跑真实数据库/Mailpit 端到端，未修改 Go、迁移或远程依赖。
- 当前仍未完成完整 OIDC、第三方登录、资料修改、密码恢复、开发者应用审核及管理员 MFA。前端与后端对话建议分工见 [交接文档](workstreams.md)，不将页面入口当作后端能力已交付。

## 前后端分离后的授权后端增量

2026-10-18 用户指定本对话只推进后端。分工与共享文件同步见 [coordination](coordination/README.md)，前端接入清单见 [backend](coordination/backend.md)。当前工具无跨对话消息能力，未声称另一对话已收到自动通知。

- 完成 Fosite Authorization Code + PKCE S256、OIDC ID Token、Discovery 和 Bearer UserInfo；批准/拒绝使用独立浏览器事务、account 登录、Origin/CSRF 和 subject_id 确认。
- 授权码、PKCE、OIDC 与令牌状态使用外层原子事务及行锁。错误 verifier 和签发失败回滚；并发兑换仅一次成功，代码重放提交旧 access 状态撤销。identity 读取通过本模块服务复用事务连接，不在协议模块越界查询用户表。
- 迁移 3 已实际应用到 Ubuntu 专属 moetopia_dev，新增 authorization.interactions，未更改已执行的迁移 1/2，未重建数据或轮换密钥。
- Go 单元测试、vet 和真实专属 PG/Valkey 集成通过；新增 JWT issuer/audience/sub/nonce/at_hash、scope 隔离、事务失败与并发、单连接池、停用身份/应用、Cookie/CSRF/Origin、过期/拒绝/重复确认、强制认证及账号切换验证。测试库由运行器清理。
- OpenAPI 与 TypeScript 已生成；后端运行和前端 DTO/错误交接见 [run-authorization](run-authorization.md)，实施参数见 [ADR-0015](decisions/0015-authorization-code-oidc.md)。开发公开客户端 CLI 已编译，未自动创建默认客户端。
- 没有修改前端页面、组件、翻译或前端测试，未停止其他会话进程。Nuxt Discovery 代理、真实授权页和独立 RP 的完整浏览器互操作仍待前端接入后验收；刷新、公开撤销/内省、RP 退出、应用审核、管理员 MFA 和生产邮件尚未交付。
