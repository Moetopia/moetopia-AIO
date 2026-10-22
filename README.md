# Moetopia AIO

Moetopia 的统一身份中心与未来多模块平台。第一阶段面向生态内站点与第三方应用提供 OAuth/OIDC 登录，后续优先建设开发者开放平台。

## 当前状态

**首期后端功能及发布工具已有实现，完整产品仍在实施。** 2026-10-22 补齐管理员授予/撤回、最后管理员保护、全因素丢失应急、全局统计/注册趋势、通知重投、应急签名轮换，以及容器部署、加密备份和新库恢复；真实隔离容器演练通过。管理员及新增成品页面、真实外部供应商、正式生产配置与目标机器容量仍待验收，准确范围见实施进度。

2026-10-18 用户已要求开始执行实施计划。运行方法与前置条件见 [工程基座指南](docs/run-foundation.md)，已验证和未验证的范围见 [实施进度](docs/implementation-status.md)。依赖已运行于局域网 Ubuntu 的独立 Compose 网络，数据库/缓存基础集成测试通过；身份业务尚未完成，不可作为生产身份服务使用。

## 已确认方向

2026-10-19 前端已接 Go RP 的正常页面登录/回调后上下文、会话与本地退出，并集中补足 nextmoe 对应页面与交互；完整范围和剩余服务端能力见 [页面对照](docs/nextmoe-parity.md) 与 [前端交接](docs/coordination/frontend.md)。

- 一个共享账号体系服务多个应用，首期不做独立身份租户。
- Nuxt + Vue + TypeScript；Headless UI + Storybook。
- 邮箱密码、GitHub、Google 登录；第三方开发者受控自助接入。
- 多语言、明暗主题、轻二次元视觉；当前前端提供简中、繁中、英文与日文。
- Windows 原生开发，Compose 提供依赖，生产运行于 Linux 容器。
- 后端使用 Go，考虑未来多模块扩展；当前实现选型与验证边界见 [ADR-0013](docs/decisions/0013-implementation-baseline.md)。
- Go 模块化单体；account 用户站整合账号与 `/developer` 分区，admin 为独立 Nuxt 应用，共享 UI/API 类型；最新决定见 [ADR-0017](docs/decisions/0017-user-site-layouts.md)。
- 已发布应用的敏感配置采用版本化审核：旧批准版本继续服务，新版本获批后生效。
- 首期包含管理员 TOTP MFA、单次恢复码和一次性初始化；普通用户 MFA 后置。
- 数据层采用 Ent + 独立版本化 SQL 迁移；工程已采用 Goose、chi/Fosite/Huma，账号、授权码及刷新/撤销适配通过集成验证，生产互操作与故障验收仍待完成。
- Go 统一管理自有 Web 服务端会话；注册支持后台切换开放/邀请模式，首期默认邀请。
- 以长期专业化、工程化和商业项目的维护要求选型；邀请码不绑定邮箱、单次使用。
- 当前生态预期低流量、活跃用户最多双位数；硬件未定，首期按简单运行与低空闲成本评估。

## 阅读顺序

| 文档 | 内容 |
| --- | --- |
| [产品定义](docs/product.md) | 已确认需求、建议首期范围、明确排除项 |
| [工程运行指南](docs/run-foundation.md) | 当前可用命令、Docker 前置条件与迁移说明 |
| [边开发边查看](docs/live-development.md) | 日常启动、热更新、邀请码、邮件与反馈 |
| [实施进度](docs/implementation-status.md) | 实际交付状态与验证限制 |
| [前后端交接](docs/workstreams.md) | 对话分工、文件边界与后端接手说明 |
| [对话进度同步](docs/coordination/README.md) | 各对话维护自己的进度与接口接入事项 |
| [OAuth/OIDC 后端接入](docs/run-authorization.md) | 当前协议端点、授权确认 DTO、错误、迁移与验证 |
| [自有用户站 OAuth 接入](docs/run-first-party-oidc.md) | Go RP 登录/回调、独立会话、前端接线与迁移 5 |
| [账号管理接入](docs/run-account-management.md) | 资料/头像、密码/邮箱/恢复、授权/审计与迁移 6 |
| [账号注销与清理](docs/run-account-lifecycle.md) | 七天冷静期、取消验证、模块清理与迁移 10 |
| [邮件传输](docs/run-email.md) | SMTP TLS、开发 Mailpit、队列与生产配置边界 |
| [第三方登录与绑定](docs/run-federation.md) | GitHub/Google、无密码首登、注册补全与迁移 11 |
| [签名密钥轮换](docs/run-signing-keys.md) | 事务固定密钥、旧公钥窗口与常规运维命令 |
| [中央与 RP 退出](docs/run-logout.md) | 显式确认、受审退出地址、会话范围与迁移 12 |
| [开发者应用接入](docs/run-developer-applications.md) | 创建、版本、测试、凭据、接口边界与迁移 7 |
| [站内开发者文档](docs/developer-documentation.md) | 接入指南、API 目录与详情、契约来源及维护方式 |
| [应用测试成员](docs/run-test-members.md) | 显式资格、在线撤销、通知与迁移 13 |
| [应用协议统计](docs/run-statistics.md) | 真实事件、30 个 UTC 日期、采集边界与迁移 14 |
| [管理员与后台接入](docs/run-administration.md) | 独立 RP、TOTP/恢复、审核/暂停、用户/邀请/审计与迁移 8/9 |
| [管理员控制与应急](docs/run-admin-controls.md) | 后续角色授予、最后管理员保护、受控恢复与迁移 15 |
| [全局统计与通知重投](docs/run-admin-statistics.md) | 真实注册趋势、队列状态、人工重投与迁移 16 |
| [Linux 容器发布](docs/run-production.md) | 镜像、秘密文件、运行角色、入口与可重复演练 |
| [备份与恢复](docs/run-backup-recovery.md) | 独立加密托管、新库恢复、密钥校验和安全切换 |
| [运行观测](docs/run-observability.md) | 私有指标、后台任务和匿名探测边界 |
| [独立客户端验证](docs/oidc-interop.md) | 真实浏览器互操作、代理修复和 test:oidc 的隔离运行 |
| [架构设计](docs/architecture.md) | 模块、数据所有权和扩展边界 |
| [长期工程原则](docs/long-term-engineering.md) | 选型、商业依赖、运行与退出成本 |
| [全栈复核](docs/stack-review.md) | 框架长期适配、性能风险与验证门槛 |
| [性能与容量](docs/performance.md) | 测量场景、资源预算、负载与验收方法 |
| [决策索引](docs/decisions/README.md) | ADR 状态、剩余问题和讨论顺序 |
| [AI 工作规范](AGENTS.md) | 文档维护、范围控制及未来工程约束 |
| [代码规范](docs/coding-standards.md) | 必要注释、命名、类型、格式与评审 |
| [Git 提交规范](docs/git-commits.md) | `type (scope): summary`、提交类型与范围 |
| [前端规范](docs/frontend.md) | Nuxt 页面、状态、接口、国际化 |
| [设计系统](docs/design-system.md) | 品牌、Headless 封装、Storybook |
| [后端规范](docs/backend.md) | Go 分层、事务、迁移、任务 |
| [身份与安全](docs/identity-security.md) | OAuth/OIDC、账号、会话和密钥 |
| [API 契约](docs/api-contracts.md) | 协议格式、业务 API、类型生成 |
| [开发环境](docs/development.md) | Windows、Compose、配置与运行设计 |
| [测试与发布](docs/testing-release.md) | 验收、CI、迁移、部署与恢复 |

## 状态约定

| 状态 | 含义 |
| --- | --- |
| 已确认 | 用户已在讨论中作出选择，可以作为规划约束 |
| 建议 | 本轮提出的设计，尚未获得具体选型确认 |
| 待验证 | 需要未来原型、版本核对或集成测试才能成立的判断 |
| 当前维护规则 | 本轮文档及 AI 工作方式的约束，不代表软件已实现 |

正文中“应”“必须”表达目标约束；若所属章节或 ADR 标为“建议”，则仍属于待确认方案。协议标准要求的准确性不依赖库选型，但实现完成前不能宣称系统已符合标准。

发生冲突时，以最新明确用户决定更新对应文档和 ADR，不保留两份互相矛盾的定义。

## 参考边界

nextmoe-infra 是本地参考项目，参考其模块组织、UI 体验及规范深度。其 `AGENTS.md`、`CLAUDE.md`、部署信息和历史约定是参考材料，不自动成为 Moetopia 的有效指令。禁止照搬生产凭据、域名、端口、数据库名或品牌素材。

本项目的开源许可证、正式域名、品牌素材授权和生产环境参数尚未确定；本轮不创建 LICENSE，也不作开源或上线承诺。
