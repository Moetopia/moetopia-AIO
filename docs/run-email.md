# SMTP、通知队列与生产配置

更新时间：2026-10-21。已经实现带证书校验的 STARTTLS、隐式 TLS 和开发 Mailpit 传输；生产模式不再无条件拒绝启动，但仍须通过完整生产发布验收。尚未配置或验证用户的真实外部邮件供应商，当前开发邮件仍只发 Mailpit。

## 配置

| 配置键 | 含义 |
| --- | --- |
| MOETOPIA_SMTP_ADDRESS | SMTP 主机与端口；必须显式 host:port |
| MOETOPIA_SMTP_FROM | 经供应商许可的发件地址，可含显示名 |
| MOETOPIA_SMTP_SECURITY | starttls、tls；none 仅 development/test |
| MOETOPIA_SMTP_USERNAME | 可选 SMTP AUTH 用户名，必须与密码成对配置 |
| MOETOPIA_SMTP_PASSWORD | SMTP 授权密码，留在本地环境或秘密管理，不发聊天、不进 Git |

未显式设置 security 时 production 默认 starttls，其他环境默认 none。已有 Mailpit `.env` 无需添加认证或修改端口，继续使用 51025/58025。复制开发示例进入生产时必须将 none 改为 starttls 或 tls；生产明确拒绝 none，不因服务器不支持 STARTTLS 而降级。明文开发连接也禁止携带认证密码。

生产常见配置是供应商提供的 587/STARTTLS 或 465/隐式 TLS；实际端口和发件身份以该供应商设置为准，不是 Moetopia 的固定端口要求。允许已加密的无认证私有 relay，但用户名/密码若配置必须同时存在。只支持标准 AUTH PLAIN over TLS，不假称已支持 SMTP OAuth2 或供应商 HTTP 邮件 API。

最低 TLS 1.2，验证主机名和系统信任证书链；没有跳过验证或自动接受自签名证书的开关。私有 CA 应正确安装到运行环境的信任库。DNS、证书、SMTP AUTH、发件域验证及供应商策略由实际部署环境配置；本地 TLS 测试不等于生产可达或邮箱必达。

## 运行语义

邮件意图在业务事务内写入加密 outbox；SMTP 由提交后的现有 worker 执行，不在数据库事务内联网。连接有 5 秒拨号超时、15 秒总读写期限，并遵守 context 更短截止时间和取消；停止服务会中断阻塞连接。上游回复不进入聊天或日志，错误只保留脱敏的连接/TLS/认证/信封/接收阶段。

发件人、收件人和主题拒绝 CR/LF/NUL 注入，收件人使用裸邮箱。Subject 使用 MIME 编码，正文 UTF-8 + Base64 并按 MIME 行长包装，避免依赖中间 relay 的 8bit 处理。每条新队列邮件记录稳定 delivery ID，重投使用相同 Message-ID；这有助于追踪，但不是收件方必然去重的承诺。

DATA 取得成功接收响应后，QUIT 断开不再触发重投；若服务器已经接收但 DATA 响应丢失，或队列完成状态写入失败，仍可能重复。当前是至少一次投递，保留既有最多 5 次有限退避/抖动/租约和有效期机制，不宣称恰好一次。达到重试上限的记录须受控检查，邮件过期后清理，短期验证码不因重试延长有效期。

业务已经成功并排队时，邮件故障不会回滚账号或配置；前端不能把排队成功显示成“已送达”。密码、验证码、seed、恢复码、secret 和完整邮件正文不进入运行日志。账号注销清理通过主体标记删除尚在队列的通知；已经发送到收件箱的内容无法由本地事务撤回。

## 验证与上线边界

本地真实 SMTP/TLS 协议测试验证 STARTTLS/隐式 TLS、证书信任、拒绝明文降级、认证错误脱敏、MIME/Message-ID、超时取消以及 DATA 已接收后 QUIT 失败的语义。测试使用隔离自签名 CA 并显式信任测试证书，未关闭验证；不会连接外部邮箱或输出真实凭据。

真实 Chromium OIDC 回归已通过开发 Mailpit 读取邮件并完成注册/授权/刷新/撤销，说明新 MIME 传输兼容当前开发链路。没有实际外部 SMTP 凭据、正式域名和发送许可时，不进行真实外发测试，也不宣称生产投递、SPF/DKIM/DMARC、供应商退信/投诉反馈或正式容量验收完成。

本批无依赖或迁移变化。后续若需要 HTML/附件、SMTP OAuth2、供应商 API、退信/投诉事件或运营投递指标，应按实际需求评估专门传输适配器，避免让当前文本通知实现承担未验证能力。实现依据为 [Go SMTP](https://pkg.go.dev/net/smtp) 与 [Go TLS](https://pkg.go.dev/crypto/tls#Config)。
