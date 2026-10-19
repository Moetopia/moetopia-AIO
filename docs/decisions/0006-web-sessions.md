# ADR-0006：自有 Web 会话与代理边界

- 状态：Go 统一管理自有 Web 会话已确认；Cookie、代理和外部令牌细节仍为建议或待定。
- 记录日期：2026-10-17。
- 确认日期：2026-10-17；确认范围为 Go 持有 account/admin/developer 的服务端会话，浏览器使用 HttpOnly Cookie，不在 localStorage 保存令牌。

## 已确认决定与具体建议

Go 统一管理自有 Web 服务端会话。建议会话使用随机不透明标识；浏览器仅保存 Host-only、HttpOnly、Secure 的会话 Cookie。Nuxt/Nitro 或入口只做固定目标的同源转发，不创建另一套可独立签发的登录态。具体入口选择在正式域名确定后落实。

原建议为 account 保有中央身份会话，admin/developer 为第一方 OIDC client。2026-10-19 用户进一步要求用户中心也走 OAuth/OIDC；最新设计见 [ADR-0019](0019-first-party-oidc.md)。身份提供者会话与自有用户站 RP 会话需区分，回调由 Go 处理；同源 developer 可与 account 共用受控 RP，不恢复第三个前端。每个 origin 的 Cookie 不共享父域。跨站 SSO 通过顶层跳转到中央登录站点完成，不依赖第三方 Cookie。

浏览器不持有客户端秘密，不将 access/refresh token 存入 localStorage。第一方客户端的令牌如需保存，仅在服务端保存。第三方应用仍可使用标准 public/confidential client 流程，不强制其采用本项目的会话方式。

## 理由与代价

减少浏览器令牌暴露，并集中会话撤销；代价是可信代理、CSRF、防会话固定、多 origin 配置和退出传播需要显式设计。HttpOnly 不能消除 XSS 的代操作风险。

Web 会话、中央身份会话、OAuth 授权与下游站点会话分别管理；RP-Initiated Logout 不等于全网所有 RP 已同步退出。

## 替代方案

Nuxt 自持 BFF 会话是讨论过但本轮未采用的替代方案，会增加会话持久层。浏览器直接持令牌适合部分 public client，却增加刷新、XSS 与存储设计成本。首期不混用两种自有 Web 会话所有者，不能在实现中自行改成 Nuxt 持有权威会话。

## 待验证与重新评估

验证 Host 转发白名单、跨站回调的 SameSite 行为、CSRF、端到端 TLS、多个第一方应用的用户上下文和代理超时。SameSite 建议 Lax，依实际回调方式验证，不能靠全局关闭防护解决。

外部 API access token 采用 JWT 还是 opaque 尚未确认。JWT 可离线验证但存在撤销窗口；opaque 依赖 introspection 的可用性。最终时效、暂停传播和旧密钥保留需与该决定一起确定。
