# 应用测试成员

状态：后端实现；页面由前端对话接入。所有者之外的既有用户可被明确授予未发布应用的测试登录资格，不获得配置、凭据、审核或其他管理权限。架构取舍见 [ADR-0029](decisions/0029-application-test-members.md)。

## 接口与字段

使用用户站 portal Cookie；写操作同时要求固定 Origin、portal CSRF 和最新应用 revision。路径使用应用 UUID，不是 client_id。

| 方法与路径 | 输入 | 成功结果 |
| --- | --- | --- |
| GET `/api/v1/developer/applications/{id}/test-members` | 无 | 200，revision、limit、items |
| POST 同路径 | revision、user_id | 200，更新后的同一列表结构 |
| DELETE `/api/v1/developer/applications/{id}/test-members/{user_id}` | revision | 200，更新后的同一列表结构 |

items 每项仅含 user_id 和 created_at，不提供邮箱搜索、用户名或邮箱目录。目标用户主动分享平台用户 UUID；这不是邀请码，不创建账号、不绕过注册邀请和邮箱验证。空列表为 []。生成包提供 DeveloperTestMembers / DeveloperTestMemberRequest。

开发基线每应用最多 20 名显式成员，所有者隐式可测试且不计入额度，不能将其加入或移除。目标须为可用、未申请注销的既有用户；通知需要已验证邮箱。只允许向尚未首次发布的应用添加，已发布应用允许清理旧名单。用户冻结/注销状态仍由身份层独立校验，名单不覆盖身份限制或应用暂停。

添加、移除都递增应用 revision，不产生配置版本、不改变待审/批准配置。名单变更不需要配置审核；第三方应用仍不能未经审核公开登录。待审期间变更名单会使管理员手中旧 revision 失效，审核者须重新读取。

401/403 表示会话或请求保护失败，其他所有者或已删应用为 404 application_unavailable。重复成员、已发布添加、不存在成员的移除及过期 revision 返回 409 application_conflict；满额为 409 application_quota_reached；非法、自身或不可用 UUID 为 422 application_invalid。依赖失败返回 503，前端不要无条件重试写请求。

## 凭据与原子性

未发布时，授权详情、同意决定、代码/刷新兑换、在线 UserInfo/内省和中央退出检查所有者/显式成员资格。移除成员在客户端排他锁内撤销该用户在该测试应用的授权族及快照；重新添加不会复活旧凭据，须重新明确同意。发布后普通用户已可授权，清理旧名单不会撤销其正常公共授权。

在线校验读取有效名单，不缓存旧资格。纯离线 JWT 验证仍存在既定 access token 时效窗口，不能声称下游离线立即撤销。撤销端点保持既有幂等语义。

名单、族撤销、revision、审计及所有者/成员通知共用事务；任何一步失败均回滚。成功只表示排队，邮件仍为至少一次投递。审计记录应用及用户 UUID，不含邮箱或凭据。添加按 UUID 排序锁定所有者和目标身份，再锁应用、客户端、授权族，避免互相添加死锁并与账号清理协调；读取名单用应用共享锁保持名单与 revision 一致。

authorization 拥有唯一有效名单；developer 通过协议模块命令编排，不维护另一份测试权限。物理注销删除目标在其他应用的名单，删除自有 client 通过同模块外键级联清理名单。仅软删或停用应用不复活任何资格。

## 迁移与验证边界

迁移 13 在迁移 12 后新建 authorization.client_test_subjects，包含 client 内外键、client/user 唯一约束和 user 查询索引，不建立跨 identity schema 的用户外键。没有新依赖。实际应用状态与测试结果见 [实施进度](implementation-status.md)。旧二进制仍只允许所有者测试，升级 API 后才支持成员；应先迁移再启动新版，缺表 readiness 失败。Down 会清空名单，不能在新版运行时执行，本轮不执行 Down。

成品成员页面、真实邮件收件、下游离线验证及生产并发容量不由 HTTP/数据库测试证明。
