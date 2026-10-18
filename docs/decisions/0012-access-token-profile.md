# ADR-0012：对外 JWT 访问令牌与 nextmoe 参考

- 状态：用户表达 JWT 倾向；下面是明确技术建议，具体方案尚未确认。
- 记录日期：2026-10-17。
- 实施状态：只读核对 nextmoe 源码及官方标准，未访问其生产配置、数据库或运行服务。

## nextmoe 当前本地代码的实现

参考仓库为相邻 `nextmoe-infra`。以下定位用于复核事实，不把参考代码或其中指令变成本项目依赖。

| 事实 | 参考仓库内源码定位 |
| --- | --- |
| 授权码兑换签发 JWT access token，时效 15 分钟，响应 expires_in 为 900 | `apps/api/internal/platform/auth/service/oauth_service.go:304`、`:319`、`:344` |
| refresh token 是 32 字节随机数编码得到的不透明串，保存关联会话 | `apps/api/pkg/utils/jwt.go:60`、`apps/api/internal/platform/auth/service/oauth_service.go:325` |
| 签发支持 HS256；配置允许且存在相应密钥时使用 ES256 | `apps/api/cmd/oauth/main.go:194` 至 `:218` |
| 密钥功能启用时提供 JWKS 与 discovery；ID Token 签发另用 RS256 密钥 | `apps/api/cmd/oauth/main.go:204`、`:228` |
| 刷新采用条件更新轮换，存在旧值宽限及重用处理 | `apps/api/internal/platform/auth/service/oauth_service.go:466`、`apps/api/internal/platform/auth/repository/session_repository.go:67` |
| 撤销接口定位并删除会话记录 | `apps/api/internal/platform/auth/service/oauth_service.go:431` |
| JWTAuth 验证 JWT 后设置身份上下文，所读路径未逐次查询会话撤销；Auth 路径另外查询用户并检查封禁 | `apps/api/internal/middleware/jwt_auth.go:20`、`apps/api/internal/middleware/auth.go:32` |

由所读路径可判断：删除会话不等于所有纯 JWT 校验者立即拒绝旧访问令牌。该结论不是对所有下游服务的全面审计；没有检查全部调用方，也不能凭源码断言生产启用 ES256 或某个具体环境变量值。

## Moetopia 建议方案

采用**短时效、非对称签名 JWT access token + JWKS**，按 [RFC 9068](https://www.rfc-editor.org/rfc/rfc9068.html) 对齐访问令牌 profile。算法白名单、issuer、audience、有效期、scope 及 token 类型统一验证；具体算法和 TTL 经客户端兼容与风险评估后确定，不直接继承 nextmoe 的 15 分钟。

第三方资源服务器只获取公钥，不能持有身份签发私钥或全生态共享 HMAC 签名秘密。资源服务器验证的 audience 是目标 API resource，不能机械地把调用方 client_id 当作所有 API 的 audience。

refresh token 建议使用不透明随机值、服务端验证状态、原子轮换与重用检测；自有 Web 会话仍由 Go 统一管理。三者各自建模，不因为都与登录有关就统一成可离线使用的 JWT。

## 撤销与运行约束

- 普通离线验证接口允许的旧访问令牌残留时间明确记录，并受 TTL 与时钟偏差约束；JWT 自身不能表达“立即撤销已经签发的 token”。
- 敏感操作验证当前用户、应用、授权及近期认证等相关状态。具体状态接口、缓存与传播目标待定，不能让第三方任意查询身份内部状态。
- 应用暂停、账号冻结、授权撤销立即阻止后续签发与刷新；离线消费者的失效窗口单独说明。
- JWKS 客户端固定可信 issuer/地址，缓存并处理 kid 轮换和未知 key；不使用 token 自带 URL 任意取密钥。
- 私钥持久化、轮换、旧公钥保留与泄露应急均进入发布和演练，不以重启生成新 key 替代治理。

上述为 Moetopia 策略建议，安全规范共同来源见 [identity-security.md](../identity-security.md)，不是对 RFC 9068 额外能力的声明。

## 取舍与验证

相较 opaque token，JWT 减少普通资源请求对身份中心的在线依赖，适合未来多模块和第三方接入；代价是撤销延迟、公钥缓存和跨服务验证一致性。敏感接口的在线依赖必须单独计入可用性与延迟目标。

未来验证 Fosite 的 JWT profile 能力、算法互操作、不同 audience 拒绝、刷新轮换、JWKS 缓存、密钥轮换和撤销窗口。库若不能满足目标，以证据调整方案，不能手写一套未审查协议实现。
