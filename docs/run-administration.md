# 管理员安全与后台接入

更新时间：2026-10-19。本批实现独立管理员 RP、角色检查、TOTP、恢复码、一次性初始化，以及审核、平台暂停、普通用户停启、注册策略、邀请码和审计接口。实际验证和开发库操作结果由 [实施进度](implementation-status.md) 记录；管理员页面由前端对话接入，不将协议探针当作成品后台。

## 运行与一次性初始化

管理员来源为 `MOETOPIA_ADMIN_ORIGIN`，开发默认配置 `http://admin.localhost:3001`；必须是独立于 issuer 的 origin。生产要求 HTTPS，开发 HTTP 只允许 loopback。用户站、身份提供者和管理员使用不同 Cookie/会话应用及 CSRF，不能只靠页面隐藏按钮放行后台。

核对自己的专属开发目标后，按顺序使用已有命令：

```powershell
pnpm api:migrate
pnpm api:admin:init
pnpm api:admin:bootstrap --email admin@example.test
```

示例邮箱必须替换为已经注册、验证邮箱且启用的本地账号；CLI 不注册新账号，不输出密码、seed 或恢复码。用户已在当前会话指定首位管理员，实际操作结果见实施进度，不把真实邮箱写入仓库。

`admin:init` 显式创建固定 confidential client `moetopia-admin`，回调固定为管理员 origin + `/api/v1/admin/callback`。secret 按独立用途从既有密钥派生，只保存 bcrypt；重复执行仅校验，配置不同或停用时拒绝覆盖。`admin:bootstrap` 通过唯一 bootstrap 记录保证并发下只成功一次；即使角色后来不可用，也不重新开放初始化，不以删除标记或环境变量绕过 MFA。当前角色为单一 administrator，尚无多级 RBAC、公开角色授予或第二管理员管理接口。

角色赋予后仍须通过真实 OAuth 重新认证并完成 TOTP 绑定，才能使用后台。自己的 TUI 中 api 若因缺迁移而退出，迁移完成后选 api 按 r；不重新初始化账号、签名密钥或开发数据库。

## 页面接入顺序

1. 管理员站读取 GET `/api/v1/admin/context`，消费生成包的 AdminContext；匿名时完整导航到 `/api/v1/admin/login?return_to=/`，不能用 fetch 跟随登录。
2. Go 发起独立客户端的授权码/S256/state/nonce，强制 prompt=login 和 max_age=0，经身份站重新认证及显式同意后回到 Go。普通账号即使完成同意，没有启用的管理角色也不会建立管理员会话。
3. `authenticated=true` 只表示第一因素与角色成立，**必须继续判断 mfa.verified**。未绑定则进入绑定页，已绑定但未验证/过期则进入验证页；绑定、验证、恢复和退出本身不能受“已完成 MFA”守卫拦截。
4. 写请求使用管理员上下文的 csrf_token 和管理员 origin。绑定完成、验证成功或恢复成功都会旋转 Cookie/CSRF，立即替换上下文；不要继续提交旧 token。
5. Go 回调、Cookie、令牌交换和验签全部留在后端。保留共享 `/api/**` 与 `/oauth/**` 固定代理的 manual redirect；不能截获 code 或把用户站 Cookie/CSRF 复制到后台。

允许的 return_to：`/`、`/applications`、`/users`、`/invitations`、`/audit`、`/settings`；省略为 `/`。路径白名单是协议限制，不表示所有前端页面已实现。OAuth 登录事务绑定 client ID、浏览器和目的页；本次升级前已经发起但未结束的旧 RP 事务需重新发起，已有用户站会话不因此重建。

## 身份与 MFA 契约

所有路径前缀为 `/api/v1/admin`；下表列相对路径。原始登录/回调为重定向接口，其余 DTO 来自 [OpenAPI](../packages/api-client/openapi.json) 与 [生成类型](../packages/api-client/src/schema.ts)。

| 方法与路径 | body | 结果 |
| --- | --- | --- |
| GET `/login`、GET `/callback` | 无；Go 管理协议 query | 303 或脱敏协议问题 |
| GET `/context` | 无 | authenticated、csrf_token、login_url；登录后含 user、expires_at、mfa |
| POST `/logout` | 无 | 匿名上下文，结束本地管理会话 |
| POST `/mfa/enrollment` | current_password | secret、otpauth_uri、expires_at；仅未绑定时可用 |
| POST `/mfa/enrollment/confirm` | code | 旋转后的上下文及一次性 recovery_codes |
| POST `/mfa/verify` | code | 旋转后的上下文与 MFA 有效期 |
| POST `/mfa/recover` | current_password、recovery_code | 旋转后的第一因素上下文，要求重新绑定 |
| POST `/mfa/recovery-codes` | current_password | 新 recovery_codes；须已有有效 MFA |
| GET `/sessions` | 无 | items、current_id；须有效 MFA |
| DELETE `/sessions/{id}` | 无 | 204，检查当前用户和 admin 应用所有权 |

MFA 状态包括 enrolled、verified、verified_until（有效时才提供）、recovery_codes_remaining。不要把 authenticated 或 enrolled 单独当作完整后台权限。

开发参数：管理员第一因素会话绝对有效期 1 小时，权限提升 15 分钟，绑定事务 5 分钟。Cookie/CSRF 旋转不延长原第一因素的绝对期限；到期须重新 OAuth 登录。TOTP 采用 30 秒、6 位、SHA-1，允许前后各一个时间步；保存最后成功 counter，旧/重复 counter 在并发下不能再次使用。seed 使用带用途绑定的 AEAD 加密，待绑定 seed 另绑定发起会话并到期清除。

恢复码每组 10 个，每个 128 bit 随机值，数据库仅存带用户/用途的 SHA-256 验证值。创建或重生时只返回一次，不通过 context/日志找回。使用其中一个恢复码后，原 seed、整组旧恢复码、所有管理员会话和提升记录均失效；当前浏览器只得到重新绑定所需的受限会话。完成新 seed 验证后才恢复管理权限，并收到一组新码。重生恢复码也使旧集合失效；安全变更与通知 outbox 同事务，当前邮件只进 Mailpit。

首次 seed/otpauth URI 和恢复码仅在组件内存展示，不写 Cookie、localStorage、SSR payload、日志、截图或错误报告，不自动重试绑定确认、恢复或重生请求。已绑定时不能用 enrollment 直接覆盖 seed；更换验证器通过恢复并重新绑定。全部因素与恢复码都丢失时没有在线或 CLI 绕过路径，受控应急恢复和生产备份演练仍待完成。

管理写操作复用 Valkey 限流，以独立管理员主体键及 IP 限制；额度也包括正常管理写操作，不能将连续操作的 429 当作验证码错误或自动重试。当前额度是开发基线，不是生产容量承诺。

## 管理业务契约

以下接口均要求启用角色、存活 admin 会话和未过期的 MFA。权限校验、业务更新及成功审计处于同一外层事务，领域模块加入该事务，不另开连接提交；有真实单连接池与故障回滚验证。

| 方法与路径 | 输入 | 结果 |
| --- | --- | --- |
| GET `/applications` | status=pending（默认）或 all，可选 cursor | 应用列表，每页 20 |
| GET `/applications/{id}` | 应用 UUID | latest、approved、生效和暂停状态 |
| GET `/applications/{id}/versions` | 可选 before，非负整数 | 配置历史每页 20；字段和分页语义见 [应用管理](run-developer-applications.md#配置版本与应用通知) |
| POST `/applications/{id}/review` | revision、approve、reason | 更新后的应用；拒绝必须说明理由 |
| POST `/applications/{id}/suspension` | revision、suspended、reason | 更新后的应用；暂停必须说明理由 |
| GET `/users` | 可选 query、cursor | 普通及管理账号资料，每页 50，不含密码/hash |
| POST `/users/{id}/availability` | revision、enabled、reason | 更新后的用户；停用必须说明理由 |
| GET `/registration-policy` | 无 | mode、revision |
| PUT `/registration-policy` | mode=invite/open、revision | 新策略；条件冲突为 409 |
| GET `/invitations` | 可选 cursor | 每页 50，仅 ID/状态/时间，无邀请码原文 |
| POST `/invitations` | expires_in_hours（1–720） | id、code、expires_at；code 仅返回本次 |
| DELETE `/invitations/{id}` | 邀请记录的 64 位 hash ID | 204；已消费不能撤销，重复撤销不复活 |
| GET `/audit-events` | 可选 cursor | 最近 90 天每页 50 条 |

列表 next_cursor 为空字符串表示结束。用户 query 为邮箱/昵称前缀（最少 2 字符）或完整 ID，游标绑定原 query；变更筛选后丢弃旧游标。当前不提供统计趋势/汇总接口，不用列表长度伪造全站统计。小规模前缀检索的生产索引/容量仍需实测。

平台 suspended 与所有者 enabled 分开：有效可用条件为 enabled 且未 suspended。所有者自行启用、改草稿均不能绕过暂停；解除暂停不会复活旧授权。发布版 pending/rejected 仍使用旧批准配置；通过审核按既有保守策略撤销旧授权族并切换版本，详细语义见 [应用管理](run-developer-applications.md)。2026-10-21 新增版本历史读取和批准/拒绝、暂停/解除的所有者邮件通知；队列失败整笔回滚，账号冻结不妨碍管理员暂停其应用。通知投递和保留策略由应用文档唯一定义。

普通账号停用/重启都递增 security_version，撤销全部自有会话并消费未完成挑战；重启不恢复旧会话、授权码或刷新凭据。离线 JWT 仍有到期残留窗口。**该接口禁止操作任何管理员主体，包括自己**，以免普通用户控制误封最后管理员；角色授予/撤回和受控管理员冻结另行设计，不提供伪装成普通账号停用的后门。

注册模式更新与注册消费通过策略行锁协调；切到邀请模式不会让此前打开的页面绕过后端。邀请码不绑定邮箱、单次兑换，撤销与兑换按数据库条件串行，不能把可分享邀请码写入审计。审计新增 actor_id、target_id、action、outcome、request_id、created_at；新 HTTP 管理变更和 MFA 失败有脱敏事件。旧事件的缺失目标/关联 ID 保留为空，不虚构历史回填；CLI 事件可以没有 HTTP request ID，不承诺所有拒绝和基础设施故障都已有完整安全事件。

## 错误与恢复

| HTTP / code | 处理 |
| --- | --- |
| 401 | 第一因素/账号/会话不可用，重新发起管理员 OAuth |
| 403 admin_role_required | 没有启用的管理角色；不使用普通用户登录绕过 |
| 403 admin_mfa_required | 保留第一因素，转绑定或验证页，不循环 OAuth 登录 |
| 403 admin_target_protected | 普通账号管理不能修改管理员主体 |
| 409 admin_security_state_changed | 绑定状态或浏览器事务变化，重读 context |
| 422 admin_verification_invalid | TOTP/恢复码不正确、已用或不满足时钟窗口 |
| 401 reauthentication_failed | 当前密码不正确 |
| 409 application_conflict / account_revision_conflict | 重新读取资源或策略 revision 后再编辑 |
| 403 / 429 / 503 | CSRF/Origin、限流或依赖故障；不回退放行 |

普通未获角色账号在 RP 回调可收到 rp_identity_unavailable，回调不会建立管理会话。其余 rp_* 错误沿用 [自有 RP](run-first-party-oidc.md)，admin 与 user-site 的回调、浏览器 Cookie、state 载荷不能互换。

## 迁移与验证边界

迁移 8 新建 administration.principals/elevations/bootstrap，给应用增加平台暂停状态，为审计增加目标、结果与关联 ID。迁移 9 增加账号停用原因、邀请码撤销标记和注册策略 revision。旧账号、签名密钥及用户站客户端不重建；新列默认值兼容原有数据。部署顺序为迁移 → 管理 RP 显式初始化 → 一次性角色赋予 → OAuth/MFA 绑定。开发库实际执行状态见实施进度。

Down 会删除管理员因素/恢复码/初始化记录、管理员会话与客户端、审计新字段和平台暂停信息，不可能恢复已删秘密；迁移 9 Down 会先把已撤销邀请标为已消费，避免删除标记后重新可用。本轮不执行 Down。不得在新版运行或仍需新权限策略时直接回退，生产锁、备份/恢复和降级方案尚未验收。

验证含 RFC 6238 向量、时钟偏差/重放、初始角色并发单次赋予、真实 OIDC/S256/Basic/JWKS 回调、跨 RP state/Cookie/Origin 隔离、MFA 旋转、恢复码并发单次消费及重新绑定、失败限流、权限过期、角色失效、审核与暂停、审核审计失败回滚、单连接事务、用户停启和管理员保护、注册 revision 与撤销邀请拒绝兑换。`pnpm test:admin` 使用独立管理员域名、真实 Chromium 与专属临时库；其管理端是隔离协议探针，不等于 Nuxt 后台 UI 验收。

实现依据：[RFC 6238](https://www.rfc-editor.org/rfc/rfc6238.html)、[pquerna/otp](https://github.com/pquerna/otp)、[OWASP MFA 指南](https://cheatsheetseries.owasp.org/cheatsheets/Multifactor_Authentication_Cheat_Sheet.html)。首期安全方向已获用户确认，具体开发时效、参数和剩余应急策略不视为已确认生产配置。
