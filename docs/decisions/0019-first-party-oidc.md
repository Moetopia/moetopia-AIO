# ADR-0019：自有中心通过 OAuth/OIDC 登录

- 日期：2026-10-19。
- 状态：用户已要求；Go RP 实现和端到端接入尚未完成。
- 更新范围：ADR-0006、ADR-0017 中直接复用身份认证 Cookie 登录用户站的实施路径；Go 服务端会话所有权、两个前端、同源 developer 分区保持。

## 用户决定与参考事实

用户明确要求用户中心和开发者中心都调用 OAuth 登录。nextmoe 源码中 account/useAuth 调用密码接口，developer/admin/useOAuthLogin 则发起 OAuth；因此本决定是 Moetopia 的新要求，不宣称参考项目的 account 已经使用相同方式。

## 目标职责

身份提供者处理密码、注册、账号认证和授权交互；自有用户站作为 RP 发起授权码 + PKCE/OIDC 登录，Go 接收回调、兑换授权码、验证 ID Token，再建立 RP 的 HttpOnly 会话。身份会话与 RP 会话必须能区分，不能让提供者认证页受 RP 登录守卫保护，从而循环跳转。

account 与同源 developer 可共用一个受控 RP，登录后通过受验证的站内路径返回；是否最终采用一个 client 由后端结合权限模型验证，不默认开放动态客户端注册。admin 独立 RP、管理员权限和 MFA 继续由 Go 执行。

state/nonce/PKCE 及回调校验、秘密、令牌和会话归 Go；Nuxt 渲染页面和转发固定上游。不得为迁就参考源码把令牌写入 localStorage，不在前端捏造登录完成或把普通页面跳转称为 OAuth。

## 接入与验收

后端提供启动、回调、RP 上下文和退出契约后，前端替换当前直登入口和守卫。固定回调与允许的站内返回路径由后端校验；完成前保留现有流程可运行，但文档必须明确它不满足新目标。

需验证匿名/已有身份会话/已有 RP 会话、回调重放、错误 state/nonce/issuer/audience/签名、拒绝、过期、退出、恢复原目的页与管理员权限/MFA。不能仅用 mock 跳转代替真实协议验收。具体交接见 [前端协调](../coordination/frontend.md)。

参考：[OIDC Authorization Code Flow](https://openid.net/specs/openid-connect-core-1_0.html#CodeFlowAuth)、[OAuth 2.0 Security BCP](https://www.rfc-editor.org/rfc/rfc9700.html)。
