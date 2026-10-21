# ADR-0023：受限令牌内省、配置历史与应用邮件通知

- 日期：2026-10-21。
- 状态：已授权首期范围内的工程实现；不代表生产容量与部署验收。验证事实见 [实施进度](../implementation-status.md)。
- 扩展：ADR-0018、ADR-0021、ADR-0022。

公开协议路径 `/oauth/introspect` 使用 Fosite 的客户端认证、令牌解析与验真，按 [RFC 7662](https://www.rfc-editor.org/rfc/rfc7662.html) 区分认证错误与不可用令牌。当前只开放 confidential client 的 HTTP Basic，且只能查询本客户端签发的 access/refresh；不允许 public client、普通 bearer token、匿名或混用认证。将来独立资源服务器需要单独的身份与授权策略，不通过放开所有应用互查实现。

未知、过期、已撤销、轮换消费及他人令牌统一为 `200 {"active":false}`。有效令牌还需通过当前客户端、授权族、账号启用状态和 security_version 校验；尚未回填主体的旧授权从加密会话判断用户上下文，不能被误当成机器凭据跳过账号检查。内省读取不消费 refresh，也不触发重放撤销。基础设施/解密故障为 503，不由 Fosite 的 inactive 包装掩盖；存储读取在内省 context 内保留故障信号。

响应只允许 active/client_id/iss/sub/scope/exp/aud/token_type，不把 JWT ExtraClaims、邮箱、昵称或内部安全版本直接输出。refresh 的 exp 使用其自身有效期，不采用 access 的时间。响应 no-store；调用方自行缓存会延迟撤销可见性，不能据一次成功结果承诺后续持续有效。默认限流沿用协议 IP 预算，生产资源服务器负载与独立限额仍需测量。元数据字段按 [RFC 8414](https://www.rfc-editor.org/rfc/rfc8414.html#section-2) 增补到现有 Discovery；没有因此宣称实现独立 OAuth metadata well-known 或标准退出。

应用版本沿用现有 application_versions，通过 `(application_id, version)` 唯一索引逆序分页，每页 20，before 为排他的版本上界。所有者只能看自己的未删除应用；管理员须角色、活跃会话和 MFA，并加入既有管理事务。历史保存各版本的配置、当前审核状态/理由及创建时间，不返回 reviewer_id 或凭据。draft 会继续被编辑，pending 会被审核变更，因此这不是逐次编辑的不可变审计日志；没有历史审核时间字段时不补造 reviewed_at，也不提供一键回滚绕过重新审核。

批准/拒绝、平台暂停/解除及 confidential secret 轮换同事务写入加密 outbox，通知已验证邮箱，不含任何新旧秘密。暂停状态未变化时不重复发送。身份模块专门提供含已停用账号的已验证通知邮箱，避免账号冻结使管理员暂停其应用失败。邮件队列失败应回滚业务、协议配置和成功审计；实际 SMTP 投递留在提交后的 worker，允许至少一次重试。邮件 24 小时过期，属于开发基线；没有新增站内通知箱、Webhook 或生产邮件传输。

代价与重评条件：内省需要数据库和密码哈希认证成本；多个同族请求可能因授权族锁串行。高频调用应先测量再引入受限缓存或资源服务器独立凭据。历史查询跨页不持有快照，新增版本或审核变化时重读首页；正式通知保留期、多语言模板、站内通知、Webhook、独立资源服务器与历史回退流程另行评审。本批无新依赖与迁移，开发库仍使用版本 9。
