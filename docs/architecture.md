# 架构与模块边界

状态：模块化单体与三个 Nuxt 应用已确认。协议基础、identity、account sessions 和开发通知已有实现，完整目标拓扑尚未交付，实际边界见 [实施进度](implementation-status.md)、[ADR-0014](decisions/0014-account-preview.md)。已确认产品约束见 [product.md](product.md)。

## 架构目标

先用模块化单体交付统一身份中心，保留按领域拆分的边界。共享账号与跨模块整合不等于共享任意数据库表，也不要求一开始部署微服务。

## 建议拓扑

```text
浏览器 ── TLS 入口 ── account / admin / developer（Nuxt）
           │                │ 同源转发
           └────────────── Go API
                            ├── PostgreSQL
                            └── KV 缓存（Valkey/Redis 候选）
                         Go worker ── SMTP
                            └── PostgreSQL outbox（候选）
第三方应用 ── 标准 OAuth/OIDC 端点 ── Go API
```

图是目标拓扑，不是已存在的服务。浏览器会话所有权及代理可信边界见 [ADR-0006](decisions/0006-web-sessions.md)。

Nuxt 应用各自可选择 SSR、CSR 或预渲染，不要求三个常驻 SSR 进程。缓存发行物和具体用途未选定；完整资源及性能取舍见 [全栈复核](stack-review.md)。

## 模块与数据所有权

| 模块 | 自有数据与职责 | 对外能力 |
| --- | --- | --- |
| identity | 用户、验证邮箱、密码凭据、联邦绑定、账号状态 | 识别主体、修改身份资料、冻结/注销 |
| sessions | 登录会话、重新认证、MFA 状态、设备元数据 | 创建、验证、撤销会话 |
| authorization | 授权请求、同意记录、授权码、令牌族、签名密钥元数据 | OAuth/OIDC 协议、授权撤销 |
| applications | client、所有者、测试成员、配置版本、审核状态、客户端凭据 | 应用配置、审核和暂停 |
| access | 平台管理权限、API resource、scope、角色授予 | 判断主体对资源的操作权限 |
| notifications | 邮件模板、投递任务、尝试记录 | 提交及投递通知 |
| audit | 操作者、目标、行为、结果、时间与关联 ID | 写入和受权查询审计 |

authorization 通过 applications 的已批准配置视图工作；不能读取待审版本作为有效权限。sessions 与 authorization 各自管理登录会话和 OAuth 授权，不混用一张“token 表”表达所有状态。

## 调用与依赖

- HTTP handler 负责解析和协议适配；应用层编排用例；领域逻辑不依赖 HTTP、Nuxt、Redis 或具体协议库。
- 基础设施通过明确接口接入。协议库与生成 SQL 类型限制在适配层，不穿透到所有业务模块。
- 领域间通过拥有者的应用接口访问，禁止跨领域 SQL、直接修改外模块模型或导入其内部持久化包。
- 装配入口负责组合依赖，模块不能反向依赖入口。接口在消费处按需要定义，不为每个类型创建接口。
- 同进程必要一致性优先使用拥有者提供的同步用例；异步通知使用事件。避免每次调用都走 HTTP。
- 跨模块事务必须说明不变量、参与模块和未来拆分代价，不能依靠隐式共享 transaction 达成“模块化”。

安全判断失败时不以旧缓存放行账号或应用状态；短时访问令牌的失效窗口需明确。具体策略见 [身份与安全](identity-security.md)。

## 数据与异步处理建议

首期一个 PostgreSQL 实例、一个应用数据库，模块使用独立 schema 管理自有表。schema 是所有权约束，不宣称它提供已验证的安全隔离。未来按领域拆库时仍保持 API 与主体标识稳定。

跨模块不默认建立外键或联表；模块内使用数据库约束保证唯一性和一致性。用户 ID 使用稳定、不含个人信息的标识，具体格式待确认。邮箱、昵称和外部供应商 subject 不能替代平台用户 ID。

outbox 与业务状态同事务落库。worker 以至少一次语义处理，通过幂等键、领取租约、失败重试及人工重投避免静默丢失。首期不引入 Kafka 或通用工作流引擎。投递设计见 [后端规范](backend.md)。

## 目标目录与当前实现

已建立三个应用、Go API 与共享包；`cmd/worker` 等未来职责尚未创建，实际状态见 [实施进度](implementation-status.md)。

```text
apps/
  api/                 单 Go module；cmd/api、cmd/worker、internal/<domain>
  account/             用户账号与登录体验
  admin/               平台控制台
  developer/           开发者门户
packages/
  ui/                  Vue 基础组件与 Storybook
  api-client/          业务 API 生成类型及调用适配
  locales/             共享翻译
  portal/              三应用共享 Nuxt 层与页面壳
docs/                  规范、ADR 和契约说明
```

前端应用只通过公开包入口共享代码，不互相导入页面。当前未提供 Git remote，Go module 暂用 `moetopia.local/aio/api`；发布前按实际仓库调整，不复制参考仓库的 `module api`。

## 后续扩展与拆分条件

开发者 Webhook、图片、文件、社区分别形成独立领域，使用平台用户 ID 关联主体、使用有 audience 的令牌获得授权。积分、内容推荐、社区角色等不直接塞入身份核心。

只有独立伸缩、资源竞争、发布节奏或隔离需求有证据时才拆服务。拆分前完成契约、事件幂等、失败恢复、数据迁移和可观测性方案。账号冻结、应用暂停、删除事件的下游传播应有明确延迟边界；不能承诺未实现的实时全网一致。
