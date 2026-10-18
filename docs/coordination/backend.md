# 后端进度与前端接入

更新时间：2026-10-18。所有者：本后端对话。分工基线见 [workstreams](../workstreams.md)，同步方式见 [本目录说明](README.md)。当前工具不能向另一个对话发送消息，未声称前端已收到通知或读取本文件。

## 已完成

- 原有账号接口兼容：邮箱验证、邀请注册、密码登录、context/CSRF、退出和会话列表/撤销。
- 新增 Authorization Code + PKCE S256、OIDC RS256 ID Token、Discovery、UserInfo 后端。
- 新增授权详情/批准/拒绝业务接口，OpenAPI 与 TypeScript 类型已生成。
- 授权浏览器 Cookie 与 account 会话 Cookie 分开；原子消费交互、代码/PKCE/OIDC 状态，错误 verifier 可重试，代码重放撤销旧 access 状态。
- 迁移 3 新增 authorization.interactions；没有修改迁移 1/2。专属 Ubuntu 开发库已应用迁移 3，不重建数据、不轮换密钥、不修改远程容器配置。

## 前端本轮可接入

完整字段、错误、运行步骤与限制见 [run-authorization.md](../run-authorization.md)，实现理由见 [ADR-0015](../decisions/0015-authorization-code-oidc.md)。请按生成包类型接入，不创建第二份 DTO。

1. Nuxt 增加 `/.well-known/openid-configuration` 到固定 Go 上游的代理；已有 `/oauth/**` 保留，不能从 Host 或用户参数选择上游。
2. account 首页读取并保留 authorization_request；GET `/api/v1/account/authorizations/{id}` 展示真实应用、回调、权限、账号，不自动批准。
3. 根据 login_required 完成或重新完成密码登录；登录后重读详情和 CSRF。事务 Cookie 不随登录旋转丢失。
4. POST 同一路径，body 为 decision（approve/deny）和 subject_id（详情的 user.id），继续带 Cookie、Origin、X-Moetopia-CSRF。
5. 成功使用完整浏览器导航前往 redirect_to。处理 401/403/404/409/422/429/503；过期或已消费从 RP 重启，账号变化重读详情并重新确认。
6. 授权页采用 no-store、no-referrer 及禁止嵌入策略；补充真实授权详情、拒绝、过期、账号切换、强制登录的页面测试。前端 mock 只能做页面验证，不声称真实协议互操作。

## 验证证据与限制

Go 测试与 vet 已通过。真实 Ubuntu PostgreSQL/Valkey 的专属临时库集成通过，覆盖独立 JWT 验签/issuer/audience/sub/nonce/at_hash、资料 scope、错误 verifier 回滚、并发单次兑换、重放撤销、浏览器 Cookie/Origin/CSRF、拒绝/过期/重复确认、账号切换、强制认证、停用用户/客户端及只有一个数据库连接时的兑换。临时库已由运行器清理。

本轮没有修改三个 Nuxt 应用、UI、翻译、Storybook、前端测试或根依赖文件，也没有结束任何开发进程。授权页、Nuxt Discovery 代理和独立 RP 的完整浏览器互操作尚未验收。

没有 refresh_token、公开 revoke/introspect、RP 退出、自动记忆同意、开发者审核、管理员 MFA、生产邮件或生产发布。当前 Discovery 不声明这些能力。开发公开客户端 CLI 已编译；未自动创建默认客户端，待 RP 提供回调后按运行说明显式创建。

## 后续后端事项

2026-10-18 启动器增量：用户要求优化混合日志，已将 pnpm dev 改为 mprocs TUI，提供 dev:plain 和按服务选择；更新了根 package.json、pnpm-lock.yaml、scripts/dev* 及开发运行文档，未修改页面。各对话可以运行 pnpm dev api / pnpm dev account，但不能重复启动同一服务；完整键位与行为见 [开发预览](../live-development.md)。本轮 Windows 四服务启动、单独停止/重启 account、TUI 退出及端口释放已实测，不代表身份业务重新验收。

本轮检查发现工作区现在已有 Git 元数据（main 尚无首个提交，存在大量预先暂存与未暂存内容），取代早期“没有 Git”的环境快照；本对话未初始化 Git、调整暂存区或提交。

待授权页接入后完成真实 RP/浏览器互操作，再推进刷新轮换及撤销、账号恢复/资料变更、开发者配置版本审核和管理员 MFA。生产 TTL、正式 issuer、轮换与退出策略仍需发布前验收，不能将本轮后端测试称为完整身份产品交付。
