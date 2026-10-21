# 身份、授权与安全约束

状态：目标安全约束；不代表通过合规或生产安全验收。已实现账号基础、授权码/OIDC、刷新轮换和协议撤销，选型和参数见 [ADR-0013](decisions/0013-implementation-baseline.md)、[ADR-0014](decisions/0014-account-preview.md)、[ADR-0015](decisions/0015-authorization-code-oidc.md) 及 [ADR-0018](decisions/0018-refresh-token-rotation.md)。授权页和独立 RP 的真实 Chromium 闭环已验证；2026-10-21 新增受限令牌内省，见 [ADR-0023](decisions/0023-introspection-and-application-history.md)，标准退出仍待完成，不能把下述目标全部当作已实现。协议依据初次核对：2026-10-17；授权码、刷新和撤销复核：2026-10-18；RFC 7662/8414 与锁定 Fosite 内省源码复核：2026-10-21。

## 规范依据与项目策略

| 资料 | 本项目使用范围 |
| --- | --- |
| [RFC 6749](https://www.rfc-editor.org/rfc/rfc6749.html) | OAuth 授权码、客户端和 token 响应 |
| [RFC 6750](https://www.rfc-editor.org/rfc/rfc6750.html) | Bearer token 使用及资源接口错误 |
| [RFC 7636](https://www.rfc-editor.org/rfc/rfc7636.html) | PKCE challenge 与 verifier |
| [RFC 9700](https://www.rfc-editor.org/rfc/rfc9700.html) | OAuth 安全最佳实践、刷新保护及重定向 |
| [OIDC Core 1.0](https://openid.net/specs/openid-connect-core-1_0.html) | 登录身份、ID Token、claims 与验证 |
| [OIDC Discovery 1.0](https://openid.net/specs/openid-connect-discovery-1_0.html) | issuer 元数据和密钥发现 |
| [RFC 7009](https://www.rfc-editor.org/rfc/rfc7009.html) | token 撤销接口 |
| [RFC 7662](https://www.rfc-editor.org/rfc/rfc7662.html) | 受鉴权保护的 introspection |
| [OIDC RP-Initiated Logout](https://openid.net/specs/openid-connect-rpinitiated-1_0.html) | RP 发起中央退出与回跳 |

下文区分规范语义与 Moetopia 的收紧策略。文档未宣称 OAuth 2.1 或 OIDC 认证；元数据只能声明经过实现和验证的能力。采用 Fosite 不自动提供全部功能。

## 威胁与信任边界

保护对象包括密码、验证流程、账号绑定、登录会话、授权码、令牌、客户端凭据和签名密钥。主要场景为账号接管、钓鱼应用、回跳劫持、重放、跨应用越权、CSRF、XSS 和管理员凭据失窃。

第三方应用、浏览器输入、转发头、供应商返回值均需验证。前端无法决定应用是否获批或用户是否有管理权限。共享账号不代表向所有应用披露全部资料。

## OAuth/OIDC 协议基线

- 建议首期实现 Authorization Code、Refresh Token、Client Credentials；不启用 Implicit 或密码授权。邮箱密码登录属于身份交互，不是 OAuth 密码授权。
- Moetopia 策略建议所有授权码客户端强制 PKCE S256。verifier 与 challenge 由协议层验证；缺失、错误、降级与重放均拒绝。依据见 RFC 7636。
- 按 RFC 6749 绑定授权码、client 和回调；授权码短期、单次消费，通过事务保证并发下不能重复兑换。
- 回调使用预注册精确匹配，禁止通配和任意跳转。生产 HTTPS；开发 loopback 例外单列策略，不据此允许任意 HTTP。原生客户端端口例外如将来支持，另按 [RFC 8252](https://www.rfc-editor.org/rfc/rfc8252.html) 定义。
- 自有客户端与联邦接入使用一次性 state 绑定浏览器事务；OIDC 请求使用 nonce，并在存在 nonce 时正确核验，作为项目收紧策略。不把 state 当作任意回跳 URL。
- 根据 OIDC Core 校验签名、算法、issuer、audience、有效期及适用的 azp/nonce；ID Token 不是 API access token。UserInfo 的 sub 必须与已验证身份匹配。
- resource/audience 与 scope 由服务器批准，不能由客户端任意扩权；身份 scope 不等于后台操作权限。
- Client Credentials 只提供应用主体，不伪造用户或签发用户 ID Token。公开客户端不能依靠内嵌 secret 被视为 confidential。
- discovery 的 issuer 是稳定配置，JWKS 只公开公钥；不得从 Host 头动态拼装可信 issuer。依据见 Discovery 规范。
- token 响应与 Bearer 错误保留标准状态、字段和必要 header，不套业务信封。协议错误不得泄露内部栈或数据库信息。

刷新令牌默认轮换并检测重用；轮换、旧凭据失效和新凭据创建需要原子操作，重用时处理整个令牌族。当前开发实现采用严格单次刷新、无并发宽限，时效、清理及失败语义由 [ADR-0018](decisions/0018-refresh-token-rotation.md) 唯一定义；不是已经确认的生产参数。安全方向依据 RFC 9700，生产 TTL 与并发体验仍需发布前评估。

## 账号注册、密码与恢复

注册支持后台切换开放与邀请模式，首期默认邀请，已确认；策略由 Go 执行，见 [ADR-0011](decisions/0011-registration-policy.md)。邮箱注册仍需验证邮箱，频率限制待确认。邮箱规范化规则必须统一；不擅自套用特定供应商去点、去加号规则。数据库保证已确认的唯一约束。

邀请模式下，第三方首次登录如需创建本地用户，也必须通过注册准入，不能通过 GitHub/Google 绕过邀请。已有账号登录和显式绑定不视为新注册。已确认邀请码由管理员创建、不绑定邮箱、单次使用、可到期及撤销；时效、额度和并发兑换细节见 ADR-0011，仍待完善。邀请不替代邮箱验证。

密码只保存带版本参数的强哈希，候选 Argon2id；不使用可逆加密。client secret 的哈希策略需满足协议库接口，不能未经验证强行与用户密码共用实现。

验证码与恢复令牌绑定用途、目标和事务，短期有效、次数受限、单次消费。低熵验证码使用服务端秘密保护的验证值并限流，避免数据库泄露后可直接穷举。不同用途不能互相兑换。SMTP/outbox 敏感载荷管理见 [后端规范](backend.md)。

找回密码与发送验证码的公开反馈避免账号枚举；限流综合账号、IP 和行为，不能只锁账号给攻击者提供拒绝服务工具。参数在目标环境验证后确定。

改邮箱、改密码、解绑最后一种登录方式和注销需要近期认证。邮箱变更验证新邮箱并通知旧邮箱；不能用单次修改资料请求绕过验证。重置密码、凭据泄露和封禁时撤销关联会话及刷新令牌，access token 残留窗口需按令牌形态明确。

## GitHub 与 Google 联邦绑定

2026-10-21 用户确认首次第三方注册保留纯第三方登录，不强制设置本地密码。当前实现、has_password、补全/绑定接口及验证边界由 [联邦运行契约](run-federation.md) 定义；没有配置真实供应商凭据，不声称外部登录已验收。

外部身份以 provider/issuer 与稳定 subject 标识，记录来源与验证信息。GitHub OAuth 与 Google OIDC 按各自接口验证，不假设两者返回结构相同。

不因返回邮箱相同自动合并账号。检测到冲突时要求登录既有账号并显式绑定，避免邮箱控制差异导致接管。新账号缺少可信邮箱时进入补充验证流程，不伪造已验证状态。

绑定与解绑检查当前会话和近期认证；解绑后必须仍有可用登录方式。供应商访问令牌只按实际功能所需保存，默认登录完成后不保留长期第三方 API 权限。回调 state 和临时事务只可使用一次。

## 会话、Cookie 与 CSRF

中央身份会话、自有应用会话、OAuth grant、令牌族各有生命周期。Go 统一管理自有 Web 服务端会话已确认；建议 Cookie 只携带随机不透明句柄，详见 [ADR-0006](decisions/0006-web-sessions.md)。

2026-10-19 用户要求自有用户中心/开发者中心也通过 OAuth/OIDC。Go RP 与普通页面接线已完成开发验证；当前实现严格区分身份 account Cookie 与用户站 portal Cookie，CSRF 不可交叉，完整契约见 [自有 RP 接入](run-first-party-oidc.md)，生命周期及开发参数见 [ADR-0019](decisions/0019-first-party-oidc.md)。管理员 RP 需独立角色/MFA，不能复用普通用户会话放行。

生产 Cookie 使用 Secure、HttpOnly、适当 SameSite 与 Host-only 范围；登录和权限提升后旋转 session ID。写请求校验 CSRF 与允许 origin；SameSite 只是防护之一。GET 不执行普通业务写操作，标准退出交互单独按协议保护。

SSO 通过中央站点顶层跳转。授权页避免嵌入诱导点击，配置适当 CSP/frame-ancestors、Referrer-Policy 和缓存控制。日志、URL、页面 payload 不携带令牌或恢复秘密。

## 撤销、退出与应用暂停

2026-10-21 新增 RP 发起中央确认的后端，实际对象、字段及限制见 [退出契约](run-logout.md)。确认页尚待前端接入；没有前/后通道广播退出，不等于全部第三方应用会话立即结束。

| 操作 | 建议语义 |
| --- | --- |
| 退出当前应用 | 结束当前本地应用会话 |
| RP 发起中央退出 | 验证 RP 及退出参数，结束中央身份会话，按注册地址回跳 |
| 撤销设备 | 撤销对应登录会话及明确定义的关联授权凭据 |
| 撤销应用授权 | 撤销该用户对该应用的 grant 与刷新令牌族 |
| 应用暂停/账号冻结 | 拒绝新的授权、签发和刷新，并处理现存会话与令牌 |

退出回跳依据 RP-Initiated Logout 校验，不接受任意 redirect。撤销端点按 RFC 7009 处理无效 token，不能借响应枚举有效令牌。introspection 按 RFC 7662 保护调用方权限，不公开为匿名 token 检查器。

不将 RP 发起退出宣传为全 RP 即时退出。JWT 离线验证不会自动获知撤销：需短时效、在线状态检查或其他已验证方案；opaque token 依赖在线验证。用户倾向 JWT，当前建议为非对称签名/JWKS 与敏感接口状态校验，见 [ADR-0012](decisions/0012-access-token-profile.md)；具体算法、时效和暂停传播目标仍在 [Q05](decisions/README.md) 确认。

## 管理员、密钥与审计

常规 RSA 签名轮换、十五分钟旧公钥窗口、事务固定 kid 和旧私钥清理由 [签名密钥契约](run-signing-keys.md) 定义。该实现不代替私钥泄露应急，也不轮换 Cookie/HMAC/数据加密根秘密。

已确认首期包含管理员 TOTP MFA、单次恢复码及一次性初始化，普通用户 MFA 后置，见 [ADR-0009](decisions/0009-admin-mfa.md)。建议管理角色独立授予、敏感管理操作要求近期认证；初始化采用一次性 CLI，不提供公开管理员注册。初始化载体、恢复权限和流程仍需细化，不能允许初始化或恢复长期绕过 MFA。TOTP 种子需要可恢复读取，采用加密而非不可逆哈希；恢复码保存验证值。

签名私钥与加密主密钥分离，持久化保存并控制访问；不能每次启动生成新 issuer 密钥。轮换记录 kid、算法与有效边界，旧公钥至少覆盖尚有效令牌和允许时钟偏差。泄露应急流程与普通轮换分开。

审计记录行为、时间、操作者、目标、结果与关联 ID，访问受限；不记录密码、token、Cookie、验证码、MFA seed 或原始敏感请求。IP、设备和身份信息的保留期与账号注销策略一起确认。

具体时效、限额、注销政策、算法及恢复权限仍为未决项，不在本轮填入假定生产值。验证要求见 [测试发布](testing-release.md)。

## 2026-10-19 账号与应用实施补充

资料 revision、安全版本、邮箱变更和密码恢复已实现，具体事务、参数与剩余边界由 [ADR-0020](decisions/0020-account-security-management.md) 和 [运行契约](run-account-management.md) 维护。安全变更撤销自有会话、使旧安全版本的授权码/刷新/UserInfo 失效；离线 JWT 的到期残留仍存在，不声称全站即时撤销。

开发者所有者接口、仅所有者测试、版本提交和原子生效已实现，见 [ADR-0021](decisions/0021-developer-application-lifecycle.md)。平台暂停与所有者停用独立，原批准版在待审期间继续服务，审核入口只能由独立管理员角色/MFA gate 调用。

独立管理员 RP/角色/TOTP/恢复/一次性 CLI 与受保护的审核、暂停和身份管理后端由 [ADR-0022](decisions/0022-administrator-security-and-review.md) 实现；新增审计目标、结果与 request ID。种子/恢复码和会话、权限期限、失败语义及当前生产缺口由 [管理员运行契约](run-administration.md) 唯一定义。成品后台 UI、完整角色管理、所有失败路径审计与生产应急恢复仍待完成，不能把本批测试当作整个安全目标已验收。

2026-10-21 注销、独立取消与最终清理由 [ADR-0024](decisions/0024-account-deletion-lifecycle.md) 和 [生命周期契约](run-account-lifecycle.md) 定义：pending 不与 enabled 混用，不以登录或恢复密码自动取消，不提供普通管理员注销入口；删除与审计保留、邮件和下游数据边界均在该契约维护，不等同于生产数据政策验收。
