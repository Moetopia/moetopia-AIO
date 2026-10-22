# 管理员角色与受控应急

状态：后端实现，真实验证与开发库版本见 [实施进度](implementation-status.md)。沿用独立 admin RP、Origin/CSRF 和 MFA，不使用用户站 Cookie。取舍见 [ADR-0031](decisions/0031-administrator-controls.md)。

## 角色管理

| 方法与路径 | 输入 | 成功结果 |
| --- | --- | --- |
| GET `/api/v1/admin/principals` | 可选 after，上页 next_after | items、next_after；每页 50 |
| POST `/api/v1/admin/principals/{id}/availability` | revision、enabled、current_password、reason | 更新后的管理员主体 |
| POST `/api/v1/admin/mfa/emergency` | current_password、token | 旋转后的受限 AdminContext |

前两项要求有效管理会话、角色及 MFA；写入同时要求 Origin/CSRF、限流和当前密码。主体只返回 id、enabled、enrolled、emergency_required、revision、created_at，没有邮箱目录、seed、恢复码或应急 hash。after 是排他的 UUID 排序边界，空 next_after 结束。原有用户检索可查询既有账号 UUID，但普通账号停启仍不得操作管理员，包括已撤回角色的历史主体。

首次授予使用目标既有、可用、验证邮箱且未申请注销的用户 UUID，revision=0、enabled=true。授予不注册用户、不跳过管理员 OAuth、密码证明或 MFA，也不自动生成用户的 TOTP。纯第三方账号可获得角色，但必须先通过已有邮箱恢复设置本地密码，再执行需要密码证明的管理安全操作。

revision 对应管理员安全版本；绑定、更换因素、角色变更和应急处理会使旧版本失效，409 后重新读列表。重复状态不视为成功。撤回立即清除该主体的 MFA/待绑定/恢复材料、全部管理员会话和提升；保留普通账号、普通站点会话及审计。重新授予须重新 OAuth/绑定，不能恢复旧 seed 或旧会话。撤回自身后前端转到匿名管理上下文。

不能撤回最后一位可用且已绑定 MFA 的管理员。仅授予另一位但尚未绑定不能作为替代；并发互相撤回只允许一方成功。角色记录、撤销、审计原因和目标邮箱通知同事务，通知失败整笔回滚。原因 1–500 字，不得填写密码、验证码或其他秘密。新审计 reason 在管理员审计列表可见，旧事件为空，不回填历史理由。

所有管理员控制事务先锁固定 administration.controls，再按用户 ID 排序锁定参与账号，随后锁角色/会话和业务记录。审核预取真实所有者并加入同一身份锁集合；普通账号停启也先锁目标。这避免管理员互相操作以及与测试成员的交叉身份锁倒序。当前低流量选择串行管理员控制事务；context 读取不进入此串行区。扩大管理并发前需测锁等待并按用例拆分，不能把这视作高并发保证。

## 全部 MFA 与恢复码丢失

这是服务器操作员的显式维护任务，不开放网络重置接口。命令只处理既有且启用的管理角色，不添加管理员、不重开 bootstrap、不重置密码。操作员应先确认账号归属、记录事件并使用可信主机；不要通过日志、聊天或工单传递凭据。

```powershell
pnpm api:admin:inspect --user-id 00000000-0000-4000-8000-000000000001
pnpm api:admin:emergency --user-id 00000000-0000-4000-8000-000000000001 --expected-revision 1 --reason "Verified operator recovery" --acknowledge-mfa-reset --output-file D:\private\moetopia-recovery.json
```

示例 UUID/revision/路径必须换成实际值。inspect 只输出非秘密主体状态；应急命令的 reason 必填，输出文件须为**新绝对路径**且不能覆盖现有文件。Linux 以 0600 创建；Windows 还必须使用当前用户独享的目录 ACL，Unix mode 不能代替 Windows ACL。终端只报告成功，不打印凭据。生产容器使用 `/app/administrator inspect` 和 `emergency-reset`，通过显式私有挂载目录取出文件，不把文件放进 Git 或构建上下文。

应急事务撤销所有管理员会话/提升、旧 seed、待绑定及整组恢复码，递增 revision，将 enrollment_blocked=true。一次性 256 bit 随机 token 只写本次私有文件，数据库仅存用途/用户绑定 hash 和十五分钟期限。过期清理 hash，保持阻断；重新签发必须使用最新 revision，原 token 失效。邮件只通知安全状态改变，不传递 token。文件写入或同步若在数据库提交后失败，不能假定状态未变；先 inspect，再用新 revision 写入另一个新文件。

用户重新完成管理员 OAuth 后，context 的 mfa.emergency_required=true；密码或普通 enrollment 均不能绕过。前端提供独立恢复表单向 `/mfa/emergency` 提交当前密码与 token，保留第一因素登录、退出及该恢复入口，不要求事先通过 MFA。错误/过期/重放为 422；请求保护与限流仍有效。

成功消费不会授予完整管理权限，只旋转出受限 Cookie/CSRF，并把重新绑定资格绑定到该会话。立即清空表单 token，使用新 CSRF 走原 enrollment/confirm；其他浏览器即使有密码也不能抢先绑定。完成新 TOTP 证明后才恢复管理权限并返回新恢复码。丢失这份受限会话需操作员重新签发，不能用另一个普通登录接续。原恢复码恢复流程也使用相同的会话绑定。

## 错误与发布

- 409 admin_last_administrator：先完成替代管理员的实际绑定。
- 409 admin_security_state_changed：角色或安全版本变化，重读状态。
- 422 admin_change_invalid：UUID/reason/版本格式不合法。
- 403 admin_emergency_required：必须先消费操作员凭据。
- 其余 401/403/422/429/503 沿用 [管理员契约](run-administration.md)。

迁移 15 在 14 后新增控制锁、应急/绑定字段及 audit.reason，不改变现有角色、seed 或账号。升级顺序为停止旧 API → 迁移 → 启动全部新版实例 → 开放新控制。旧二进制不理解 enrollment_blocked，**使用应急功能后不能与旧版混跑或直接回退**。Down 会先停用仍处于应急/绑定状态的主体，防止删字段变成无限制绑定；本轮不执行 Down。具体运行和测试事实只由实施进度记录，不把代码实现等同于成品后台或生产验收。
