# 开发者应用管理接入

更新时间：2026-10-21。所有者管理接口及协议接线已有实现和真实数据库测试，前端已完成原批次接入。本批新增配置版本历史和应用邮件通知。独立管理员 RP/角色/MFA 及受保护的审核/平台暂停路由也已实现，完整契约见 [管理员接入](run-administration.md)；成品后台及历史页面仍需前端接线，不以领域测试替代页面验收。

## 前端可一次接入的接口

所有接口使用用户站 portal Cookie，写操作带固定 Origin、portal 的 X-Moetopia-CSRF。DTO 由 [OpenAPI](../packages/api-client/openapi.json) 和 [共享类型](../packages/api-client/src/schema.ts) 生成；应用 UUID 与 client_id 是不同标识，管理路径使用 UUID。

| 方法与路径 | body / 查询 | 结果 |
| --- | --- | --- |
| GET `/api/v1/developer/applications` | 可选 cursor | items、next_cursor；每页 20 |
| POST `/api/v1/developer/applications` | public、config | 200 application；confidential 额外返回一次 client_secret |
| GET `/api/v1/developer/applications/{id}` | 无 | 当前应用、latest 及可选 approved |
| GET `/api/v1/developer/applications/{id}/versions` | 可选 before，非负整数 | 配置版本逆序分页；items、next_before、active_version、latest_version |
| PUT `/api/v1/developer/applications/{id}/draft` | revision、config | 更新后应用 |
| POST `/api/v1/developer/applications/{id}/submit` | revision | 待审应用 |
| POST `/api/v1/developer/applications/{id}/availability` | revision、enabled | 更新后应用 |
| POST `/api/v1/developer/applications/{id}/secret` | revision、current_password | 仅本次 client_secret；随后重读详情取得新 revision |
| DELETE `/api/v1/developer/applications/{id}` | revision | 204；必须先停用 |

config 包含 name、description、homepage_url、redirect_uris、scopes。创建后 public 不可切换；public 无 secret，confidential 使用一次返回的随机 secret。详情和列表永不返回 secret 或 hash；创建/轮换结果不得记录在日志、分析事件、持久化页面状态或截图。若丢失，应重新轮换，不能提供“查看旧密钥”。请求失败时不要无条件重试创建或轮换，它们尚无幂等键。

应用含 enabled、revision、active_version、latest；approved 在尚未发布时省略，不能假定总存在。latest.version 与应用 revision 各有含义：配置版本只在前一版本进入 approved/rejected 后再编辑时增加；任何管理变更都递增应用 revision。保存、提交、停启、轮换和删除传最新 revision，409 后重读详情。所有权在后端检查，不能通过 body 指定 owner。

## 测试与发布语义

初始版本 draft，active_version=0，只允许所有者和显式测试成员授权；名单及移除后的凭据处理见 [测试成员契约](run-test-members.md)。提交后 latest=pending，不允许修改正在审核的配置版本；当前没有撤回接口。

首次批准将测试应用切为对普通用户开放；发布后编辑产生新 draft，再提交 pending。等待审核或拒绝时，协议层仍使用旧 approved 的名称、回调和 scope，新配置不会提前生效。批准在同一事务内更换协议配置、版本状态及审计；中途数据库失败整笔回滚，旧服务及凭据仍保留。

当前采用保守凭据策略：初始测试配置编辑、首次批准及后续新版本批准都会撤销此前授权族；用户重新授权。所有者停用立即阻止新授权/兑换并撤销已存在的族，重新启用不会复活旧凭据。协议事务先固定本次 incoming client 的共享行锁，再取得授权族锁；配置变更以相同客户端的排他锁协调，避免在停用扫描期间漏掉并发签发。多个正常协议读取可并发。

secret 轮换立即使旧 secret 无法通过后续客户端认证，不提供重叠宽限；本次之前已经成功认证的事务可先完成。轮换本身不结束既有 access token，必要时先停用应用。公开客户端不能轮换 secret。

这些业务管理接口不等于 RFC 动态客户端注册。固定自有 RP 客户端不归开发者管理，不能通过用户站修改其凭据或注册配置。管理员暂停独立使用 suspended/suspension_reason；有效可用条件为 enabled 且未 suspended，所有者重新启用或更新草稿不能绕过。暂停期间不能提交/批准，解除暂停不恢复旧凭据。[应用协议统计](run-statistics.md) 已有后端，管理台全局统计、站内通知/Webhook 和所有权转移尚未实现。

## 配置版本与应用通知

版本路径用应用 UUID，使用 portal Cookie；管理员另用 `/api/v1/admin/applications/{id}/versions`，必须有效管理会话、角色和 MFA。每页最多 20 项，版本号降序；首请求省略 before 或传 0，后续将 next_before 原值传入，next_before=0 表示结束。before 是排他的版本上界，不是页码。空页 items=[]；没有其他所有者或已删除应用的可见入口。

每项是原有 Version（version/config/status/review_reason）加 created_at，即版本创建时间；不返回审核人的用户 ID，也没有 reviewed_at。active_version 标记当前协议使用版本，latest_version 标记最新保存版本；旧批准记录即使不再生效，status 仍为 approved。draft 可以继续修改，pending 会变成 approved/rejected，因此这里只是已保存配置版本列表，不是每次编辑的审计历史。跨页不持有快照，审核或新增版本后从首页重读；没有直接恢复旧版的写入口，仍须提交新草稿并审核。

批准、拒绝、平台暂停/解除以及 confidential secret 轮换将给所有者的已验证邮箱排队通知。当前只发中英邮件到开发 Mailpit，没有站内通知箱、WebSocket 或 Webhook；应用 ID/事件/审核或暂停理由可出现在通知，密码和 secret 不进入邮件。队列写入与业务/协议配置/审计同事务；队列失败返回 503 并回滚，提交后 SMTP 失败由 worker 重试，不回滚已生效配置。通知至少一次投递、24 小时过期，是开发参数，不承诺恰好一次或真实外部邮件送达。账号冻结不阻止管理员暂停其应用或通知该邮箱。

这批能力无需新迁移、初始化或依赖，现有版本 9 即可运行。取舍见 [ADR-0023](decisions/0023-introspection-and-application-history.md)，生成包新增 DeveloperApplicationHistory/AdminApplicationHistory 别名。

## 校验和错误

开发基线：每用户最多 20 个未删除应用；name 1–80 字、description 最多 500 字，1–10 个不重复精确回调。scopes 只允许 openid/profile/email/offline_access 且必须包含 openid；offline_access 才启用 refresh_token。配额在用户行锁内检查，并发创建不会突破上限；正式额度仍待评审。

回调/主页最多 2048 字符、必须有主机，不接受用户信息、query、fragment、反斜杠或换行。默认 HTTPS，仅开发 HTTP issuer 下允许 localhost、127.0.0.1、::1 和 .localhost 回调；不允许任意局域网 HTTP。实际授权仍精确匹配注册 URL，并强制 S256、nonce 和显式同意。

404 application_unavailable 隐藏不存在/其他所有者资源；409 application_conflict 表示版本或状态变化；409 application_quota_reached 表示额度已满；422 application_invalid 表示配置不合法。身份、CSRF、限流及依赖错误沿用既有 401/403/429/503，当前密码错误为 reauthentication_failed。列表游标为空字符串表示结束。

## 迁移与验证事实

迁移 7 已于 2026-10-19 实际应用到专属 `192.168.101.41:55432/moetopia_dev`，新建 developer.applications、developer.application_versions，为 authorization.clients 增加 owner_id/test_only。迁移 6 先执行，未改写已执行迁移或重建账号/密钥。其他工作区使用现有 `pnpm api:migrate`；此功能不需要创建默认应用。自己的 TUI 中 API 已退出时选 api 按 r。

Down 会删除开发者应用/版本及关联受管 OAuth client，存在不可恢复的数据损失；本轮没有执行，不在新版仍运行时回退。真实测试使用可销毁随机测试库。

集成验证覆盖所有权隔离、confidential secret 哈希/不回显/轮换、测试主体限制、真实授权码/S256/Basic 兑换和 UserInfo、pending 不污染已发布配置、拒绝保留旧版、新版批准、停用/重启不复活授权、删除、并发 revision 冲突、配额及审核写入失败原子回滚。2026-10-19 新增管理员 MFA gate、HTTP 审核/平台暂停、所有者不能覆盖暂停及审计失败整体回滚/单连接池验证。前端所有者管理已有真实浏览器批次，覆盖范围见其协调记录；管理员真实后台审核页面仍未验收。
