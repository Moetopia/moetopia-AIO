# 账号注销、取消与到期清理

更新时间：2026-10-21。后端实现已授权的七天冷静期，接口由 Huma/OpenAPI/TypeScript 生成；页面仍由前端接入。是否通过各项验证及开发库迁移以 [实施进度](implementation-status.md) 和 [后端交接](coordination/backend.md) 为准。

## 接口

| 方法与完整路径 | 身份和 body | 结果 |
| --- | --- | --- |
| POST `/api/v1/portal/deletion` | portal Cookie/CSRF/Origin，current_password、User.revision | requested_at、due_at，UTC RFC 3339 |
| POST `/api/v1/account/deletion-cancellations` | account 匿名 Cookie/CSRF/Origin，email、locale=zh-CN/en | challenge_id；成功不代表账号存在或邮件已送达 |
| POST `/api/v1/account/deletion-cancellations/confirm` | account 匿名 Cookie/CSRF/Origin，challenge_id、code、current_password | 204；不建立任何登录会话 |

类型别名为 AccountDeletion、AccountDeletionRequest、DeletionCancellationRequest。申请成功后所有会话已失效，前端清空 portal 本地状态；取消页必须允许匿名访问，不能被 RP 守卫重定向循环。取消成功后重新完成正常身份登录及 Go RP，不自动恢复旧 Cookie 或 OAuth 凭据。没有通过登录自动取消的隐式规则。

## 申请与边界

需要有效用户站会话、固定 Origin/CSRF、当前密码和最新资料 revision。密码校验后，在同一事务内锁定用户及安全版本，检查管理员保护和应用状态；到期时间固定为申请时刻加 7×24 小时，不按客户端时间、时区或浏览器日历计算。

任何已发布且所有者 enabled 的应用都必须先由所有者停用，即使该应用已被平台 suspended。检查不通过返回 409 account_published_applications，账号和应用保持原状态。通过后剩余测试应用也会停用并撤销授权；取消注销不自动启用它们。不能通过注销把仍在正式提供服务的应用静默转移给其他用户。

任何管理员主体（包括停用角色）均拒绝普通注销，403 admin_target_protected。角色/最后管理员恢复不借此流程绕过；角色授予与注销须共同遵循先锁用户再检查角色的顺序。

账号 pending 标记独立于平台 enabled。申请递增 revision/security_version，撤销全部自有会话并消费原挑战，禁止登录、创建新会话、资料/头像访问和新用户授权凭据使用。已有授权码/刷新/UserInfo/内省因账号状态或旧安全版本失效；下游离线 JWT 仍有原到期窗口。reset 邮件仍可用，但重置密码不会清除注销期限。

管理员列表新增可选 deletion_due_at；不能只据 enabled=true 将该账号显示为正常可登录。管理员重新 enabled 不等于取消注销，也不延长期限；平台冻结期间不能用取消流程解除封禁。

## 取消与找回

取消开始的反馈不披露邮箱存在性；不存在、非 pending、冻结或已经到期都返回同形状挑战，并抑制实际发送。可用挑战绑定用途、用户、邮箱、安全版本；8 位验证码、10 分钟、最多 5 次错误、单次消费，不能使用注册/改邮箱/重置挑战替代。

确认需当前密码及验证码，先进行密码哈希验证，再按用户 → 挑战顺序取得行锁；锁后重新检查当前密码 hash、安全版本、挑战状态、邮箱与实际截止时间，防止等待行锁期间越过截止时间。恰好到期及以后拒绝取消，无额外宽限。成功清除 pending、再次递增安全版本、撤销全部会话及挑战并排队通知；不恢复旧凭据。

忘记密码时先走既有密码恢复，随后重新申请取消验证码并使用新密码。恢复本身不登录，不取消注销，已发出的旧取消挑战也随安全版本变化而失效。冻结账号须先由平台通过普通管理流程处理；取消不承担申诉或解封功能。

错误沿用 401 reauthentication_failed、409 account_revision_conflict、422 account_details_invalid、403 CSRF/管理员保护、429 限流和 503 依赖故障。成功反馈不代表外部邮件投递必达。

## 到期清理

现有 worker 每分钟检查，单轮最多 10 个到期账号，以 FOR UPDATE SKIP LOCKED 协调多个实例。每个账号独立事务，取消与清理竞争同一用户行；清理前再次检查截止时间和管理员保护。失败整笔回滚，pending 记录保留供下轮重试，不用成功日志掩盖部分清理。

编排层调用各模块公开命令：developer 删除所有应用及配置版本，authorization 删除所拥有客户端/关联授权和该用户的授权快照，identity 删除账号/头像/密码/挑战/注册验证及待发邮件，sessions 删除所有自有会话。旧授权缺少 user_id 时，从经过 AEAD 校验的原始授权码快照恢复归属，包含已撤销/过期数据；不能因旧元数据缺失留下用户快照。损坏的旧数据会使清理失败并报告脱敏故障，需维护者修复，不跳过后声称删除完成。

已删除邮箱可以重新注册，产生不同 UUID，不能取回旧凭据。审计仅保留随机主体/目标 ID、动作/结果/时间/request ID，不保留资料或凭据，继续按 90 天清理；该工程行为不代替正式数据政策或法定合规验收。已交给 SMTP 的邮件和下游站点数据不能由本地事务撤回，不承诺删除第三方持有的数据。

## 迁移

迁移 10 给 identity.users 增加可空的 requested/due 时间及成对约束/到期索引，挑战用途增加 deletion-cancel；notifications 增加 user_id 索引以按主体删除加密待发邮件，authorization.clients 增加 owner_id 索引。历史账号默认无 pending，历史邮件最长原有效期 24 小时，早于七天到期清理；新的账号/管理/应用通知均标记主体。

部署顺序：核对专属数据库 → `pnpm api:migrate` → 重建 API → 接前端。不可在仍有 pending 账号时切回不识别注销状态的旧服务；Down 会明确拒绝此状态。已完成的删除不可由 Down 恢复。本阶段不执行 Down，也不申请任何日常账号注销。开发库迁移执行事实另记，不能把随机测试库的应用结果当作开发库已经迁移。
