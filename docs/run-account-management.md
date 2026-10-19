# 账号资料、安全与恢复接入

更新时间：2026-10-19。下列接口已有 Go 实现和专属 PostgreSQL/Valkey 集成测试；页面接入由前端对话维护。身份认证与用户站的边界沿用 [自有 RP](run-first-party-oidc.md)，不要为方便表单接入混用两类 Cookie/CSRF。

## 一次接入的契约

输入、输出和校验以 [生成类型](../packages/api-client/src/schema.ts) 为来源，不另写 DTO。下表字段全部位于 JSON body；GET、DELETE 的资源标识按路径传递。

| 方法与路径 | 输入 | 成功结果 |
| --- | --- | --- |
| PATCH `/api/v1/portal/profile` | display_name、bio、revision | 200 更新后的 User |
| PUT `/api/v1/portal/avatar` | image_base64、revision | 200 更新后的 User；空字符串删除头像 |
| POST `/api/v1/portal/password` | current_password、new_password | 204；所有旧会话失效 |
| POST `/api/v1/portal/email-changes` | current_password、email、locale | 200 challenge_id |
| POST `/api/v1/portal/email-changes/confirm` | current_password、challenge_id、code | 204；邮箱变更且所有旧会话失效 |
| GET `/api/v1/portal/authorizations?cursor=…` | 可选游标 | 200 items、next_cursor |
| DELETE `/api/v1/portal/authorizations/{id}` | 授权族 ID | 204；撤销该族的授权码、access 与 refresh 状态 |
| GET `/api/v1/portal/security-events` | 无 | 200 items |
| POST `/api/v1/account/password-resets` | email、locale | 200 challenge_id |
| POST `/api/v1/account/password-resets/confirm` | challenge_id、code、new_password | 204；不自动登录 |
| GET `/api/v1/avatars/{id}?v=…` | 用户 UUID | PNG；不可用返回 404 |

portal 接口要求当前用户站 RP 会话；写操作检查固定 Origin 与 portal 的 X-Moetopia-CSRF。password-resets 接口使用身份提供者 account 的匿名或已登录上下文、Cookie 和 CSRF，适用于 `/auth/forgot-password` 与 `/auth/reset-password`，不能经过 RP 登录守卫。locale 当前仅接受 zh-CN/en，其他 UI 语言暂映射到其中一种。

User 新增 bio、avatar_url、revision。资料保存必须提交最近读取的 revision，成功后替换本地 User；409 时重新获取 portal/context，再提示重新编辑。头像返回相对 URL，前端通过同源 `/api/**` 代理读取，不拼接数据库或 Go 内部地址。

改密、确认邮箱及恢复成功后，当前 Cookie 仍可能留在浏览器，但服务端句柄已经撤销；清空旧页面状态并重新进入正常登录流程。重新注册或重置密码不会自动创建 RP 会话。旧邮箱会收到安全通知；所有邮件目前只投递开发 Mailpit。

## 错误与交互

| HTTP / code | 处理 |
| --- | --- |
| 401 reauthentication_failed | 当前密码错误；不要自动重试提交 |
| 401 | 会话不可用，按所在场景重建身份或 RP 登录 |
| 403 | Origin/CSRF 不匹配；重读正确上下文，不放宽校验 |
| 409 account_revision_conflict | 资料已变化，重新读取后再保存 |
| 422 account_details_invalid | 输入、验证码、用途、时效、尝试次数或绑定不满足要求 |
| 404 authorization_unavailable | 授权不存在或不属于当前用户 |
| 422 cursor_invalid | 放弃游标，从首页重新读取 |
| 429 / 503 | 展示限流或服务故障，保留适当的非敏感输入 |

恢复开始对已存在、未知或停用邮箱返回相同形状的 challenge_id；未知账号任务使用同种加密 outbox 但不发送邮件。变更到占用邮箱同样不给出存在性提示。不能根据成功响应宣称邮件必然投递，也没有宣称严格恒定耗时。

## 当前实现参数与边界

以下是开发基线，尚非用户确认的生产额度：验证码 8 位、10 分钟、最多 5 次错误，绑定用途、事务、用户与安全版本，邮箱变更另绑定发起 RP session ID。验证码验证值使用独立用途的 HMAC；outbox 载荷加密，成功消费与账号变更、审计和会话撤销同事务。失效验证码不会因密码恢复接口而转换为其他用途。

改密要求当前密码，邮箱变更在发起和确认时都要求当前密码。密码规则复用现有 Argon2id 策略。安全变更递增 security_version，撤销全部自有会话并消费未完成挑战；授权码兑换、刷新、UserInfo 及新建会话均检查安全版本。仅离线验签的下游仍可能接受最长到 access token 到期的旧 JWT，当前开发上限 10 分钟；不宣称跨站即时退出。

头像仅支持标准 Base64 编码的 JPEG/PNG，解码后最多 512 KiB，每边最多 4096、最多 400 万像素，中心裁剪并重编码成 256×256 PNG，删除原始元数据。SVG/GIF/WebP 和 data URL 不支持。只对头像上传路由放宽请求体至 768 KiB，其他路由仍为 64 KiB；处理并发最多 2 个。头像由 identity 持有，当前直接存 PostgreSQL，不提前建设文件平台；登录热路径只查询元数据，不读取图片 blob。

授权列表每页 50 个，按创建时间和 ID 稳定排序，next_cursor 为空字符串表示结束。**每项是一条授权族，不是按客户端合并后的应用记录**；同一应用多次同意可有多项，删除只处理指定族，不删除已经独立建立的第三方或自有 RP 会话。当前用户安全事件最多返回最近 90 天的 50 条；仅包含事件 ID、操作者、动作及时间，尚未覆盖完整管理操作的目标/结果/关联 ID。

限流键按环境、固定 issuer 和数据库配置派生命名空间，随机测试库与开发实例互不消耗额度；命名空间不采用请求 Host 或用户参数。生产限额及跨实例部署仍需验收。

## 迁移与运行事实

迁移 6 已于 2026-10-19 实际应用到专属 `192.168.101.41:55432/moetopia_dev`：增加资料/安全版本字段、identity.account_challenges、audit.security_events 及授权族主体元数据。未重建账号、旧会话和签名密钥。随后实际执行 `pnpm api:grants:backfill`，本开发库 updated=0；这不是所有环境都无旧授权的承诺。

其他工作区先核对自己的专属开发目标，再执行已有命令：

```powershell
pnpm api:migrate
pnpm api:grants:backfill
```

回填只处理保留的、可验证的加密授权码快照，幂等填充旧授权族主体；不会在启动时自动执行。迁移 7 的开发者能力顺序紧随其后，见 [应用管理](run-developer-applications.md)。API readyz 要求相关新表存在；自己的 TUI 中 api 若已退出，选 api 按 r。

迁移 6 Down 会删除资料、头像、安全版本、挑战、授权主体元数据及审计表，存在不可恢复的数据损失；本轮没有执行 Down，不能在新版运行时直接回退。版本回退和生产备份恢复尚未验收。

## 验证范围

真实数据库测试覆盖资料版本冲突、头像大于原 64 KiB 请求限制及尺寸归一化、拒绝 SVG、删除头像、Cookie/CSRF 隔离、错误当前密码、全部旧会话撤销、旧认证结果不能重建会话、恢复防枚举/错误次数/并发单次消费、邮箱用途及会话绑定、旧邮箱通知、授权所有权/回填幂等/撤销及旧安全版本拒绝。新表单尚未由本后端对话作真实浏览器交互验收。

仍未实现账号七天注销与取消、GitHub/Google 绑定、生产邮件和管理员安全管理；不要将以上批次视为完整账号生命周期交付。
