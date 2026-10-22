# 管理台概览与邮件运维

状态：后端实现。所有接口需要独立 admin RP、有效角色和 MFA；写入另要求 Origin/CSRF 与限流。页面由前端接入，验证事实见 [实施进度](implementation-status.md)。设计理由见 [ADR-0032](decisions/0032-administrator-statistics.md)。

## 真实概览

GET `/api/v1/admin/statistics` 返回 timezone=UTC、generated_at，以及三个明确来源的对象：

| 对象 | 字段与口径 |
| --- | --- |
| identity | total_users 为当前保留账号行数；enabled_users 为启用且未申请注销的账号；pending_deletions 为待注销账号 |
| identity.registrations | 最近 30 个 UTC 日期的新注册事件，每项 day、partial、可选 registrations；recording_since 是采集起点 |
| applications | total 为未软删应用；pending 为 latest 配置待审核；published_enabled 为已有批准版本且所有者启用、未平台暂停；suspended 为未软删且平台暂停 |
| notifications | queued 为未过期、未租用且尝试不足五次的记录（包括延迟重试）；in_flight 为有效租约内记录；retry_exhausted 为五次已耗尽且不在投递中的未过期记录 |

注册计数由 identity 创建账号的同一事务记录，密码和纯第三方注册共用此路径；失败事务不计数、不消费正确的邀请码/邮箱证明。事件只含独立随机 ID 与 UTC 日期，没有用户、邮箱、IP、供应商标签或具体时间。物理删除账号不会倒扣匿名注册事件；当前账号行数会减少。不能把两种数字混为累计历史总账号数。

采集前的 registrations 字段**省略**，不是 null 或零；采集开始日和当前日 partial=true。已经采集而没有事件的日期返回零。迁移 16 创建采集状态和日期事件表，不从现存用户行补造已删除账号的历史。worker 每轮最多清理 1000 条窗口外记录。generated_at 表达生成时间，各模块采用当前事务的常规读取，不承诺所有计数是跨表同一瞬间快照。

应用计数由数据库真实条件聚合，不使用列表页长度；pending 和 suspended 可同时成立，不能将它们相加当总数。接口不提供 DAU、在线人数、同比增长、付费额度或预测数据。所有者应用用量继续使用 [应用协议统计](run-statistics.md)，不能以它冒充全平台业务 API 消费。

## 失败邮件与人工重投

| 方法与路径 | 输入 | 成功结果 |
| --- | --- | --- |
| GET `/api/v1/admin/notifications/failed` | 可选 after，上页 next_after | 每页最多 50 个 items，UUID 升序 |
| POST `/api/v1/admin/notifications/{id}/retry` | attempts、reason（1–500 字） | 204 |

列表只返回 id、attempts、available_at、expires_at；不返回收件人、正文、加密载荷或验证码。只列五次耗尽且租约已结束、尚未过期的记录。确认 SMTP 等实际故障修复后，可按读取的 attempts 条件重排；状态/租约变化、已投递删除、重复操作或过期返回 409 notification_state_changed。

重排恢复为零次尝试、立即可领取，不延长原 expires_at。重排和带 reason 的审计同事务，审计失败回滚；提交后由现有 worker 投递，204 不表示已经送达。不能重投过期验证码、复活已删除任务或自动无限重试，也不提供“查看邮件原文”的管理入口。重新投递仍为至少一次语义。

新版接口与类型见生成包 AdminStatistics、AdminFailedNotifications、AdminNotificationRetryRequest；主体和应急类型由 [管理员控制](run-admin-controls.md) 定义。当前 worker 在 API 进程内运行，重启 API 即重启 worker，没有独立的 worker 启动命令。真实外部 SMTP 和后台图表/队列页面仍需相应环境与前端验收。
