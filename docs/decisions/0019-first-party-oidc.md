# ADR-0019：自有中心通过 OAuth/OIDC 登录

- 日期：2026-10-19。
- 状态：用户已要求；Go RP、普通前端入口/守卫、真实数据库与 Chromium 协议闭环已完成；独立管理员 RP 后端由 [ADR-0022](0022-administrator-security-and-review.md) 扩展，后台页面待接入。
- 更新范围：ADR-0006、ADR-0017 中直接复用身份认证 Cookie 登录用户站的实施路径；Go 服务端会话所有权、两个前端、同源 developer 分区保持。

## 用户决定与参考事实

用户明确要求用户中心和开发者中心都调用 OAuth 登录。nextmoe 源码中 account/useAuth 调用密码接口，developer/admin/useOAuthLogin 则发起 OAuth；因此本决定是 Moetopia 的新要求，不宣称参考项目的 account 已经使用相同方式。

## 目标职责

身份提供者处理密码、注册、账号认证和授权交互；自有用户站作为 RP 发起授权码 + PKCE/OIDC 登录，Go 接收回调、兑换授权码、验证 ID Token，再建立 RP 的 HttpOnly 会话。身份会话与 RP 会话必须能区分，不能让提供者认证页受 RP 登录守卫保护，从而循环跳转。

account 与同源 developer 可共用一个受控 RP，登录后通过受验证的站内路径返回；是否最终采用一个 client 由后端结合权限模型验证，不默认开放动态客户端注册。admin 独立 RP、管理员权限和 MFA 继续由 Go 执行。

state/nonce/PKCE 及回调校验、秘密、令牌和会话归 Go；Nuxt 渲染页面和转发固定上游。不得为迁就参考源码把令牌写入 localStorage，不在前端捏造登录完成或把普通页面跳转称为 OAuth。

## 接入与验收

后端提供启动、回调、RP 上下文和退出契约后，前端替换当前直登入口和守卫。固定回调与允许的站内返回路径由后端校验；完成前保留现有流程可运行，但文档必须明确它不满足新目标。

2026-10-19 后端已交付 [完整接入与迁移说明](../run-first-party-oidc.md)。身份与用户站分别使用 application=account/user-site 的服务端会话、独立 HttpOnly Cookie 和用途隔离的 CSRF。RP 会话当前开发有效期 8 小时；登录事务有效期 5 分钟，随机 state 只存哈希，PKCE verifier/nonce/目的页加密，浏览器绑定后行锁单次消费并擦除载荷。回调重试、拒绝、验证失败都不能复用已消费事务；不在交换期间持有数据库事务。

受控 confidential 客户端只允许固定回调及 openid/profile/email，初始化为显式 CLI，不在启动时自动改客户端或同步模型。x/oauth2 负责授权码/PKCE 兑换，coreos/go-oidc v3.21.0 负责 RS256/JWKS、issuer、audience、时效验签，适配层补充 nonce、azp、auth_time、iat 和 at_hash 校验。会话建立前检查身份与客户端状态；后续上下文仍检查可用性。

当前模块化单体使用受限的进程内 HTTP transport 调用真实 Fosite token handler 与 JWKS handler：只接受配置 issuer 的两个固定路径，不访问浏览器指定上游，也不绕过授权码、客户端认证或验签。RP 入口仍执行公开协议限流。此方式避免回调处理依赖 Nuxt 再代理回 Go；未来拆分服务时替换为受控网络 transport，并重新验证 HTTPS、超时、密钥更新与故障边界。

兑换所得 OAuth 凭据仅用于验证本次登录，随后丢弃，不请求 offline_access、不持久化刷新令牌。建立独立 RP 会话后不依赖 ID Token 每次充当访问令牌；本地退出只结束 RP 会话，不等于中央或跨 RP 退出。管理台的独立客户端、角色与 MFA 已由 ADR-0022 实现，不能复用普通用户 Cookie 放行。上述时效与密钥派生是开发基线，正式轮换和生产策略仍需验收。

需验证匿名/已有身份会话/已有 RP 会话、回调重放、错误 state/nonce/issuer/audience/签名、拒绝、过期、退出、恢复原目的页与管理员权限/MFA。不能仅用 mock 跳转代替真实协议验收。具体交接见 [前端协调](../coordination/frontend.md)。

参考：[OIDC Authorization Code Flow](https://openid.net/specs/openid-connect-core-1_0.html#CodeFlowAuth)、[OAuth 2.0 Security BCP](https://www.rfc-editor.org/rfc/rfc9700.html)。
