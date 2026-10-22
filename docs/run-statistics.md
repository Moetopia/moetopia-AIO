# 应用协议统计

状态：后端已实现应用级真实事件统计，前端用量页尚待接入。此接口不代表整个管理台统计已完成。取舍见 [ADR-0030](decisions/0030-protocol-statistics.md)。

## 接口与口径

GET `/api/v1/developer/applications/{id}/statistics`，使用 portal Cookie 和应用 UUID；只有所有者可读。无需 CSRF 的只读接口仍要求真实 RP 会话，中央登录 Cookie 不能替代。其他所有者或软删应用返回 404，未登录 401、依赖失败 503。已停用或平台暂停的自有应用仍可查看。生成类型别名为 DeveloperApplicationStatistics。

| 字段 | 含义 |
| --- | --- |
| timezone | 固定 UTC，不采用浏览器或数据库会话时区 |
| recording_since | 此 client 的采集起始时间；已有 client 为迁移 14 时间，新 client 为创建时 |
| window_start / window_end | YYYY-MM-DD，最近 30 个 UTC 日期的左闭右开区间，包含今天 |
| generated_at | 查询完成时间，不是全库快照一致性保证 |
| totals | 窗口内已记录事件合计，不是应用全生命周期累计 |
| days | 长度 30、日期升序；每项 day、partial 和可选 counts |

counts 在早于采集起始日期时**省略**，不是 null，也不是零。采集开始当天和今天 partial=true，表示不完整的自然日；其他已采集日没有事件时返回真实零值。前端应显示“未记录”及部分日期提示，不能将缺失 counts 变成零值后计算增长率。

counts 和 totals 使用同一组字段：

| 字段 | 计数事件 |
| --- | --- |
| authorizations | 用户明确批准且授权码事务成功提交 |
| code_exchanges | 授权码成功兑换一份 access token |
| refreshes | 刷新成功签发一份新 access token |
| machine_tokens | client_credentials 成功签发一份 access token |

一次明确同意产生一个 authorizations；只有随后兑换成功才增加 code_exchanges。ID Token 与 Refresh Token 不另算一次签发；刷新重放、无效密码/客户端、失败事务、拒绝同意、UserInfo 和内省读取不计成功。HTTP 响应是否被客户端收到不可由数据库证明：这里的成功是服务端事务提交。当前开发者应用不开放 client_credentials，machine_tokens 因此通常为真实零，不能为展示数字修改 grant 权限。

这些不是独立用户数、登录成功数、业务 API 调用数、活跃在线人数、额度消费或计费账单。没有唯一用户追踪字段，也没有复制 nextmoe 的商店/API 套餐概念。管理员全局概览、注册趋势、用户级历史与可配置查询范围仍待实现。

## 数据、事务和成本

authorization.protocol_events 只存随机事件 ID、client_id、UTC day 和事件种类，不保存用户 ID、邮箱、IP、token/signature、请求体或 secret。每次成功授权/签发增加一条短记录，与凭据事务共同提交；计量写入失败会回滚授权或兑换，返回失败，未成功消费的凭据可按现有协议重试。不会先发 token 再异步丢弃统计。

查询按 client/day/kind 索引聚合，在数据库返回最多 120 个分组，不将所有事件载入 Go。没有每天 30 次查询、按用户无限标签、热点全局计数行或独立指标服务。窗口之外的记录由现有 worker 每轮最多删除 1000 条；关闭 worker 可能延迟物理清理，但查询范围仍限定 30 天。物理删除 client 由同模块外键级联删除事件；删除其他用户不保留其身份标识，也不倒扣已匿名计量的事件。

代价是每次成功协议事务增加一次数据库插入，故障时不能绕过记录继续签发。低流量下优先可验证口径；生产吞吐、存储量和保留策略仍待目标机器测量，不宣称本设计已支撑高流量或财务计量。

## 迁移与接入

迁移 14 在迁移 13 后增加 authorization.protocol_events 及 clients.statistics_since。不从会清理的 grants/protocol_sessions 回填虚构历史，不重建密钥/账号，不增加依赖。先执行 `pnpm api:migrate` 再启动新版 API/worker，readiness 会检测缺表/列。旧 API 在迁移后仍可运行但不采集，升级期间不能视作无缺口统计；单实例开发按迁移后立即重启 API/worker执行。若生产曾回滚旧版而出现采集缺口，须修订采集覆盖元数据或明确标注缺口后再展示，不可声称完整。

Down 删除统计与起始列，已有用量无法恢复；新版运行时不能执行。本轮未执行 Down。实际开发库版本、真实数据库/浏览器回归结果见 [实施进度](implementation-status.md)。前端图表、管理员看板及生产资源预算尚未验收。
