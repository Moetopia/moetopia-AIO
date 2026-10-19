# ADR-0018：刷新轮换、令牌族与协议撤销

日期：2026-10-18。状态：现有实施授权内的开发实现，数据库集成与独立 RP 浏览器验证通过；以下时效为开发基线，尚未确认为生产策略。

## 范围与选择

沿用 Fosite 的 Refresh Token、OIDC Refresh 和 RFC 7009 handler，不自行实现令牌算法。新增 `POST /oauth/revoke`；Discovery 声明 refresh_token、offline_access 和 revocation_endpoint。公开 introspection、RP 退出、用户自助授权管理及跨系统退出仍未实现。

只有允许 refresh_token grant 的客户端，且用户明确批准包含 openid/offline_access 的授权请求，才签发刷新令牌。普通授权码登录与 client_credentials 不签发刷新令牌。自有用户站继续使用 Go 服务端 Cookie 会话，不引入浏览器 localStorage 令牌。

| 参数 | 当前开发实现 |
| --- | --- |
| access token | RS256 JWT，10 分钟，不超过令牌族绝对到期时间 |
| refresh token | Fosite 不透明凭据，每次成功刷新轮换；单次有效期 24 小时且不超过族到期时间 |
| 令牌族 | 从原授权创建起最多 7 天，刷新不延长绝对期限 |
| 并发/重用 | 严格单次使用，无宽限；重复使用已消费凭据撤销整个族 |
| 刷新 scope/audience | 可省略，或保持原批准集合/单一 audience；本轮拒绝缩小 scope，也不允许扩大或更换 audience |

正常轮换后，先前 access token 可继续使用至到期；明确撤销或重用检测会使该族所有 access/refresh 状态失效。UserInfo 在线检查立即拒绝已撤销状态；下游仅离线验证 JWT 时仍有剩余有效期窗口。ID Token 是身份断言，不能据其判断实时会话或访问权限。

## 事务、重放与存储

新增 authorization.grants，使用 Fosite request ID 关联一次授权产生的代码、access 与 refresh 状态。请求先锁令牌族，再重新读取协议状态，串行化同族刷新、重用与撤销；所有签发动作仍在外层 Ent 事务内完成，Fosite 内部提交不提前提交。新令牌写入失败时连同旧刷新凭据消费一起回滚。

重用检测属于需要提交的失效操作：即使响应 invalid_grant，也提交对该族现有凭据的撤销。不同客户端、错误完整凭据或仅保留 signature 部分的篡改输入，不能触发他人令牌族失效；在向 Fosite 暴露已消费状态前先验证客户端归属及完整凭据真实性。已过期的消费记录仍需验真，临时验证副本仅用于检查签名，不修改持久化有效期或允许兑换过期凭据。

协议快照继续加密，数据库仅保存原凭据签名的用途隔离哈希作为索引。已消费代码与刷新状态保留到整个族绝对到期后再加 11 分钟，避免代码短期过期后丢失对后代刷新令牌的重放检测。清理按族加锁并删除全部关联状态，不能逐个按 token 到期立即删除。

客户端必须串行刷新并原子替换凭据；同一个旧令牌并发发送，即使一次成功，另一请求的重用检测也会撤销新令牌。网络丢失响应后盲目重试旧凭据同样可能触发此策略，需要重新授权。将来增加宽限机制必须单独评估持久化响应、重放风险及故障语义。

## RFC 7009 接口

使用 form 编码的 token 与可选 token_type_hint；客户端认证支持 Basic、Post，以及公开客户端的 client_id 归属校验。成功、重复撤销、未知令牌及无权撤销的其他客户端令牌均为空 200；错误客户端凭据仍返回 invalid_client，Basic 失败带 WWW-Authenticate。存储故障返回 temporarily_unavailable，不伪装成撤销成功。

当前策略：撤销一个属于调用客户端的 access 或 refresh token，即撤销整个族；已消费但仍可验真的 refresh token 也能撤销后代。不会借此结束 account Cookie 会话。端点具有现有协议凭据限流；不是匿名令牌有效性检查器。

## 迁移与回退

迁移 4 新建 authorization.grants 及索引，按旧 protocol_sessions 的 request_id/client_id 补齐关联；旧代码族的到期覆盖代码期加 10 分钟，已消费标记不变。2026-10-18 已使用 `pnpm api:migrate` 应用于专属 Ubuntu moetopia_dev，未重建账号、会话或签名密钥。API readyz 要求新表存在；其他工作区运行新版前须确认目标库再迁移。

上线回退应保留新增表与状态，不能在仍有新版进程或活跃刷新凭据时直接执行 Down。旧程序不能继续处理新增刷新流程，回退前需停止该能力、处理活跃授权并评估清理行为。Down 会删除令牌族状态，不是日常恢复命令；生产恢复演练仍未完成。

## 验证与重新评估

真实专属 PostgreSQL/Valkey 集成覆盖迁移 3→4 回填、轮换、独立 JWT 验签与身份保持、错误客户端/凭据隔离、并发重用、代码过期后的重放、过期/绝对时效、scope/audience、公开与 confidential 客户端、Basic 失败 header、重复/未知撤销及签发失败事务回滚。`pnpm test:oidc` 的独立 openid-client + Chromium 场景通过刷新、旧令牌重用及协议撤销；测试数据与进程由运行器清理。

生产 TTL、资源预算、正式 issuer、密钥轮换、账号冻结/恢复与授权失效联动、管理员审计、下游离线窗口和多实例故障测试仍待验收。增加资源服务器、scope 缩减、并发宽限、独立数据库或多设备授权管理时重评本决定。

依据：[RFC 9700 §4.14](https://www.rfc-editor.org/rfc/rfc9700.html#section-4.14)、[RFC 7009](https://www.rfc-editor.org/rfc/rfc7009.html)，并核对锁定 Fosite 0.49.0 源码的验证、存储及事务调用顺序。
