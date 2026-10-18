# ADR-0015：授权码、PKCE 与 OIDC 后端开发实现

日期：2026-10-18。状态：已在现有实施授权内完成后端开发与集成验证；参数是当前开发基线，不代表用户已确认生产策略或通过 OIDC 认证。前端授权页、独立 RP 浏览器互操作仍待接入。

## 背景与选择

沿用 Fosite 协议逻辑，不自行实现授权码、PKCE 或 ID Token 算法。授权入口、token、JWKS、Discovery 和 UserInfo 属于协议接口；账号授权详情和批准/拒绝属于 Huma 业务接口。公开路径与当前边界见 [后端接入说明](../run-authorization.md)。

- 授权码仅 `code`/`query`；所有客户端强制 PKCE S256。回调精确匹配，生产 HTTPS；HTTP 只在开发 HTTP issuer 下允许 loopback，暂不支持原生应用动态端口例外。
- 授权码有效期 5 分钟；access token 和 ID Token 为 RS256，有效期 10 分钟。前者 `typ=at+jwt`、API audience；后者 `typ=JWT`、客户端 audience。`openid` 请求必须有至少 8 字符 nonce；scope 和 audience 必须来自已启用客户端配置，每次只有一个 API audience。
- `profile` 仅披露 name，`email` 披露 email/email_verified；access token 不放邮箱或昵称。UserInfo 要求 Bearer access token、openid 和仍有效的服务端状态，并通过 identity 服务确认用户可用。
- 刷新、公开撤销/内省、RP 退出、动态注册、请求对象、claims 参数、ACR 和会话管理扩展未实现；拒绝 offline_access，Discovery 不声明这些能力。仅使用内部 introspector 支撑 UserInfo。
- 尚无历史授权自动批准逻辑，每次显式同意；`prompt=none` 返回 `interaction_required`。支持单值 login/consent，login 和 max_age 会检查账号会话创建时间；max_age=0 要求本次请求后的认证。其他 prompt 组合当前拒绝。

## 浏览器事务和前后端边界

新增 authorization.interactions，仅保存随机句柄、独立浏览器 Cookie 的哈希、加密参数、有效期和消费标记。有效期 10 分钟；登录旋转 account 会话不会破坏授权浏览器绑定，多个标签共享事务 Cookie、使用不同交互句柄。

授权入口回到固定 issuer 的 `/?authorization_request=<opaque>`，不把 state、nonce 或原始协议请求复制到账号页面地址。业务确认需要浏览器事务 Cookie、登录会话、固定 Origin、CSRF 和页面看到的 subject_id。账号已变化返回冲突，避免用户在旧页面把授权授予另一账号。批准和拒绝均单次消费；拒绝按协议携带 state 回到原注册回调。

## 事务与退出路径

代码、PKCE 和 OIDC 请求快照复用加密 protocol_sessions；原始授权码、verifier、client secret、token 不进入快照。OIDC 以原始码为查找输入时仍先做分用途 SHA-256，数据库不保存原始码。

token 整个处理共享外层 Ent 事务；Fosite 内层提交不提前提交。授权码行通过 `FOR UPDATE` 串行化兑换，错误 verifier 的 PKCE 删除随事务回滚；正常兑换的代码消费、PKCE/OIDC 消费和 access 状态一起提交。重放路径即使返回 invalid_grant 也提交对先前 access 状态的撤销。

identity.User 仍归 identity 所有，通过内部 persistence 上下文复用该事务连接；协议模块不直接读取 identity 表。此上下文仅用于已经明确参与的读取，不暗中让注册、通知或其他写操作加入协议事务。单数据库连接测试覆盖自我等待问题；将来模块拆成独立服务时必须替换为明确的远程状态检查及失败策略。

清理任务删除到期交互；协议快照到期后再保留 11 分钟，覆盖最晚签发的 10 分钟 access token 与开发余量，保留已消费码的重放检测窗口。调整 TTL 必须同步清理窗口和测试。JWT 被下游离线验证时仍有最长有效期内的撤销残留窗口，不声称即时全局撤销。

## 验证与限制

真实专属 PostgreSQL/Valkey 测试覆盖：独立 JWT 验签、issuer/audience/subject/nonce/at_hash、scope 资料隔离、并发单次兑换、错误 verifier 回滚、错误客户端/回调、重放撤销、匿名预览、Cookie/CSRF/Origin、账号切换、拒绝、重复及过期交互、强制认证、停用用户/应用和单连接池。

尚未完成第三方标准客户端的完整浏览器登录、Discovery 经过 Nuxt 的联调、生产密钥轮换、生产邮件及目标硬件负载测试。开发客户端 CLI 仅供本项目 dev 库建立显式公开客户端；不能替代应用所有者、版本审核或正式注册管理。

## 重新评估条件

增加刷新令牌、历史授权、原生客户端、更多 audience、资源服务或分离数据库时，重评事务、身份状态、撤销传播和保留窗口。对外发行前明确生产 TTL、正式 issuer/回调及审计策略，并完成独立 RP 与故障验收。

依据：[RFC 9700](https://www.rfc-editor.org/rfc/rfc9700.html)、[RFC 7636](https://www.rfc-editor.org/rfc/rfc7636.html)、[OIDC Core](https://openid.net/specs/openid-connect-core-1_0.html)、[Ent 行锁](https://entgo.io/docs/feature-flags/#row-level-locks)，本轮核对 2026-10-18，并检查当前锁定 Fosite 源码的事务与存储调用顺序。
