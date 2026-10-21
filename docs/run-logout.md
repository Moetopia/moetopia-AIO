# 中央确认与 RP 发起退出

状态：后端及契约已有，确认页和新增退出入口待前端接入。现有本地退出保持原行为。不能把后端 HTTP 测试称为完整退出页面或全站即时退出验收。

## 行为

本地 `/api/v1/portal/logout`、`/api/v1/admin/logout` 只结束本应用当前会话。新 central-logout 先结束发起 RP 当前会话，再导航中央 OP 询问是否结束中央身份会话。用户拒绝时中央身份仍在，已结束的 RP 会话不恢复。

OP 确认只撤销浏览器当前 account 会话，不撤销其他设备、OAuth grant/refresh 族或其他 RP 本地会话；需撤销授权时使用已有授权管理。没有实现 front-channel/back-channel logout，不宣称自动退出全部第三方应用。身份 Cookie 的旧句柄不再对应有效会话，页面完成后重新读取 context，不能凭 Cookie 是否还在判断登录。

## 协议和地址登记

Discovery 提供 end_session_endpoint，GET/POST `/oauth/logout` 接受：

| 参数 | 当前支持 |
| --- | --- |
| client_id | 有效启用的 OIDC 客户端；无 ID hint 时必需 |
| id_token_hint | 可选；验证本 OP 的 RS256 签名、kid、issuer、audience、subject、iat/exp 和 JWT 类型，可据单一 aud 推导 client_id |
| post_logout_redirect_uri | 可选；精确匹配当前生效的注册地址 |
| state | 可选，最多 512 字节，回跳时原样作为 state 参数编码 |
| ui_locales | 可选，最多 128 字节；未知语言不会报错，当前仍使用站内语言偏好 |

POST 使用 application/x-www-form-urlencoded，不混用 URL 参数。拒绝重复或不支持的参数、错误客户端/令牌和任意回跳，错误不重定向至请求者提供的位置。ID hint 当前只接受未过期或一分钟容差内的令牌；较旧 hint 不作为最近会话证明，RP 可使用 client_id 路径。提示账号与当前中央账号不同则返回 logout_account_changed，需重新发起并核对账号。项目不是 OIDC 认证产品，不将受限实现说成覆盖所有可选能力。

应用 Config 新增可选 post_logout_redirect_uris，最多十项。HTTPS，开发 loopback HTTP 例外，禁止凭据/fragment/query/通配，不能由任意输入生成信任目标。与登录回调不同用途，不能直接复用 redirect_uris 作为退出白名单。创建时缺省为空数组；更新草稿时省略新字段保留已有值，显式空数组表示清空，兼容旧编辑器。已发布应用的变更进入完整配置版本审核，旧批准列表继续生效，新列表获批后才可回跳；每次详情及确认重新校验生效配置，已移除的地址不能被旧事务继续使用。

自有 RP 固定登记 `/api/v1/portal/logout/callback` 与独立 admin origin 下的 `/api/v1/admin/logout/callback`。Go RP 不为退出保存 OAuth token，而以 client_id、固定回调、随机 state 发起；OP 始终要求显式确认。RP 回调核验独立 state、浏览器 Cookie、client/purpose、五分钟期限与单次消费，回调不清除浏览器后来新建的 RP 会话。

## 前端接线

| 接口 | 字段及保护 |
| --- | --- |
| POST `/api/v1/portal/central-logout` | portal Cookie/CSRF/Origin，返回 redirect_to 及两个 Set-Cookie（清本地会话、设置浏览器证明） |
| POST `/api/v1/admin/central-logout` | 独立 admin RP Cookie/CSRF/Origin；退出不要求完成 MFA，但仍校验管理员 RP 主体 |
| GET `/api/v1/account/logout-requests/{id}` | 浏览器事务 Cookie；返回 client_id/client_name/post_logout_redirect_uri/expires_at/requires_confirmation 和可选 user |
| POST `/api/v1/account/logout-requests/{id}` | account Cookie/CSRF/Origin；decision=approve/deny，subject_id 为展示的中央 user.id，无中央会话时为空字符串；返回 redirect_to |

1. 将“仅退出当前应用”和“同时请求退出中央身份”分开说明；新操作成功后清掉本地 RP 上下文并完整导航 redirect_to。后续中央拒绝或出错，不伪造 RP 会话仍在。
2. 新增公开 `/auth/confirm-logout?logout_request=...`，不走 portal 守卫。先读取 account/context 得到 account CSRF，再读请求详情，展示真实应用、中央账号、回跳地址和实际影响。始终等待用户确认，GET 不直接结束登录。当前该页面尚未实现；前端接入前不要开放新按钮作为可用成品。
3. 提交展示的 subject_id。409 logout_account_changed 时重读上下文及详情，不自动替新账号重试；404 logout_request_unavailable 时从 RP 重启。其余 401/403/429/503 沿用账号错误，不把失败当成功。
4. 完整导航返回的 redirect_to，不在 Nuxt 服务端跟随。保留固定 Go 代理和 manual redirect；尤其不能折叠丢失多个 Set-Cookie。认证确认页使用 no-store/no-referrer/防嵌入，不把 state、ID hint 或一次性句柄发到分析服务。
5. `/auth/logout?central=1` / completed=1 显示中央流程反馈，并重新读真实 context；cancelled=1 明确中央未结束。旧无标记页继续表示仅本地退出。RP 返回的 state 是浏览器关联保护，不是可以代替真实 context 的签名退出证明。

普通用户站返回原有退出反馈页，admin 返回自己的首页；后台成品页面仍由前端实现。生成包提供 LogoutDetails/LogoutDecisionRequest/LogoutDecisionResponse/CentralLogoutResponse，不创建第二套 DTO。

## 持久化、并发与迁移

OP 新增加密 logout_interactions；关联数据含事务 ID 和 client_id。随机句柄只存摘要，按独立用途摘要绑定浏览器 Cookie，五分钟到期。RP 复用短期 rp_logins 存储，但使用独立 state 摘要、浏览器摘要和 AEAD 用途，与登录证明不能互换。worker 清理过期记录，不持久化原始 ID hint。

确认撤销当前会话、写审计、消费并清空证明同事务，失败可重试且保留会话。先锁定/撤销当前会话，再读取受锁保护的客户端配置和确认事务，避免与管理审核/凭据撤销形成反向锁依赖。只允许一次有效确认；拒绝也消费事务。

迁移 12 在 11 后新增 client 退出地址和退出事务表，并仅为已存在的两个受控 confidential RP 从其固定登录回调派生退出回调；不改 secret、域名或第三方应用版本。新初始化 CLI 创建客户端时直接写固定退出回调。确认专属目标后使用 `pnpm api:migrate`，实际应用记录见 [实施进度](implementation-status.md)。Down 删除新增运行字段与事务，属于显式回退操作，本轮未执行。

依据：[RP-Initiated Logout 1.0](https://openid.net/specs/openid-connect-rpinitiated-1_0.html)。
