# GitHub / Google 登录与绑定

状态：后端与生成契约已有。2026-10-21 用户确认第三方首次注册不强制设置本地密码。真实供应商应用凭据、正式回调注册及页面仍待接入；模拟协议和数据库验证不能代替真实 GitHub/Google 登录验收。

## 账号与准入

外部身份唯一键为 provider、固定 issuer 和稳定 subject；GitHub 使用数字 ID，Google 使用已验证 ID Token 的 sub。供应商昵称和邮箱均不能作为账号关联键。供应商已验证邮箱仅作为补全表单的 email_hint，不自动合并，不直接标记本地邮箱已验证。

首次登录未找到绑定时进入注册补全。复用 `/api/v1/account/verification` 发送本地验证码，邀请模式仍需要有效单次邀请码；本地验证码、邀请、账号创建、绑定和补全事务的消费在同一个数据库事务完成。失败保留未消费的合法证明，错误验证码计数独立保留。已存在同邮箱时返回 registration_invalid，要求登录既有账号后显式绑定，不能代为合并。

纯第三方新账号 `has_password=false`，没有可用的本地密码。密码登录仍执行占位 Argon2id 校验后统一拒绝；不会把不可用标记当作密码。通过已验证本地邮箱完成现有密码恢复后，`has_password=true`。当前绑定、解绑、改邮箱和注销等要求密码的敏感操作，需要先设置本地密码；此规则为用户已接受的首期交互。

冻结或待注销账号不能通过已有外部身份登录，也不会被当作未注册身份重新创建。解绑必须验证当前密码，因此成功后仍保留本地密码登录。绑定和解绑递增 security_version/revision、撤销自有会话和挑战并产生安全邮件及审计；已签发 JWT 的离线残留边界沿用 [授权规则](run-authorization.md)。

## 配置与回调

可选配置，每个供应商 ID/secret 必须成对设置；未配置不会出现在 providers 清单，也不会启动时请求外网：

| 供应商 | 环境变量 | 固定回调 |
| --- | --- | --- |
| GitHub | MOETOPIA_GITHUB_CLIENT_ID / MOETOPIA_GITHUB_CLIENT_SECRET | `<MOETOPIA_ISSUER>/api/v1/account/federation/github/callback` |
| Google | MOETOPIA_GOOGLE_CLIENT_ID / MOETOPIA_GOOGLE_CLIENT_SECRET | `<MOETOPIA_ISSUER>/api/v1/account/federation/google/callback` |

在各自控制台创建应用并精确登记回调，secret 仅进入本地秘密配置或生产秘密注入。当前 `.env` 没有自动添加凭据，也没有替用户创建供应商应用。供应商对开发域名和 HTTP 回调各有约束；现有 account.localhost 能否登记需在对应控制台核验，不承诺其接受。若需改变 issuer，必须连同 Cookie、Go RP、已注册客户端和代理一起规划，不能临时放宽 Origin 或随意换 localhost。

GitHub 请求 read:user/user:email，使用 S256 PKCE，后端读取 `/user` 与 `/user/emails`。Google 请求 openid/email/profile，使用 S256、nonce、RS256/JWKS 验签，验证 issuer/audience/azp/有效期/iat，以及存在时的 at_hash。固定 HTTPS 上游、10 秒超时、禁止跟随重定向；GitHub JSON 响应限制 256 KiB。不请求离线访问，不保存供应商 access/refresh token。

## 前端批量接入

API 类型使用 `@moetopia/api-client` 导出，禁止复制 DTO。所有写请求继续携带同类 Cookie、固定 Origin、X-Moetopia-CSRF；身份 account 与普通用户站 portal 不可混用。

| 接口 | 请求/返回 | 保护 |
| --- | --- | --- |
| GET `/api/v1/account/federation/providers` | items: github/google 中已配置的项；空数组正常 | 公开、无凭据 |
| POST `/api/v1/account/federation/{provider}` | body 可选 authorization_request、continue=developer-applications；返回 redirect_to 和浏览器事务 Cookie | account CSRF + 限流 |
| GET `/api/v1/account/federation/{provider}/callback` | 供应商回调；后端处理并 303 | 一次性 state、浏览器 Cookie、供应商绑定 |
| GET `/api/v1/account/federation/registration?completion=...` | provider/display_name/email_hint/expires_at | 五分钟证明及同浏览器 Cookie |
| POST `/api/v1/account/federation/registration` | completion/challenge_id/code/invitation/display_name；无 password；返回 redirect_to 和中央登录 Cookie | account CSRF + 限流 + 原子消费 |
| GET `/api/v1/portal/identities` | items: provider/created_at；不返回 subject/token | portal 会话 |
| POST `/api/v1/portal/identities/{provider}` | current_password；返回 redirect_to 和浏览器事务 Cookie | portal CSRF、密码再认证、限流 |
| DELETE `/api/v1/portal/identities/{provider}` | current_password；成功 204，要求重新登录 | portal CSRF、密码再认证、限流 |

1. 登录页按 providers 清单显示按钮；POST start 后完整浏览器导航到 redirect_to。已有 authorization_request 要原样传回；只接受有效随机事务句柄，不接受任意 return URL。
2. 首次回调进入 `/auth/federation?completion=...`。新增该页面，读取注册详情，允许修改昵称和邮箱；邮箱提示不等于已经完成验证码。复用验证码发送与邀请码输入，提交无密码的补全 API。
3. 已绑定登录成功会建立中央 account 会话，再进入现有 Go portal RP 登录；带授权请求时返回 `/auth/authorize?authorization_request=...`，不自动批准。补全返回 redirect_to 同样必须完整导航。
4. 普通安全设置读取 identities。绑定需要密码；成功回调到 `/auth/login?federation=linked` 并要求重新登录。解绑成功后同样结束旧会话。has_password=false 时提示先通过邮箱设置密码，不能展示可完成的假解绑。
5. 认证页及补全页采用 no-store/no-referrer，避免把一次性证明传给外站；不将 completion、验证码或 Cookie 发送至分析/错误采集。代理保留 manual redirect，浏览器必须收到原始 Set-Cookie/Location。

常见错误：400 external_identity_invalid（证明无效/供应商验真失败）、409 external_identity_conflict（身份或同类供应商已绑定）、503 external_provider_unavailable（未配置）、503 external_identity_unavailable（存储等故障）。密码/本地注册继续使用 credentials_invalid / registration_invalid，另有既有 401/403/429。证明过期、拒绝、回放或失败后从登录页重新发起；不要重放供应商 code。

## 事务和清理

state 与 completion 均为随机不透明句柄，数据库只保存摘要；独立 HttpOnly、Host-only、SameSite=Lax Cookie 绑定浏览器。生产使用 Secure/__Host-。授权事务含 nonce/verifier，使用 AEAD 和事务 ID 作关联数据；回调在外部网络交换前消费并清空载荷。注册补全另建五分钟事务，只保存加密身份提示和受控继续位置，不含第三方 token。

绑定除校验用户、安全版本及发起的 portal 会话 ID 外，还在提交事务中重新锁定并验证该会话仍有效，避免网络交换期间退出后继续绑定。凭据唯一约束阻止并发跨用户绑定。过期事务由现有 worker 清理；到期物理注销删除该用户的绑定事务，外部凭据通过 identity 模块内外键级联删除，未完成的新用户证明最多保留至五分钟过期。

迁移 11 在迁移 10 后执行，增加 has_password、identity.external_identities/external_logins；既有账号默认有密码，不修改旧哈希。确认专属目标后使用已有 `pnpm api:migrate`。Down 在仍有无密码账号时拒绝；不可丢弃这些账号的唯一登录方式。此文不代替 [实施进度](implementation-status.md) 中的实际执行记录。

依据：[GitHub OAuth web flow](https://docs.github.com/en/apps/oauth-apps/building-oauth-apps/authorizing-oauth-apps)、[GitHub emails API](https://docs.github.com/en/rest/users/emails)、[Google OIDC](https://developers.google.com/identity/openid-connect/openid-connect)。
