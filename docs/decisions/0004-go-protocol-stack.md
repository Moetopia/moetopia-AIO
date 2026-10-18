# ADR-0004：Go HTTP 与 OAuth/OIDC 候选栈

- 状态：建议。
- 记录日期：2026-10-17。
- 第二轮结论：2026-10-17，用户确认保留 chi/Fosite/Huma 作为候选栈；没有确认具体采用或授权原型实施。数据层 ORM 优先不改变协议候选，见 [ADR-0010](0010-orm-comparison.md)。

## 建议决定

使用标准 `net/http`，chi 负责路由，Fosite 承担 OAuth/OIDC 协议处理，Huma 仅用于业务 API 与 OpenAPI。日志使用 `slog`，追踪候选为 OpenTelemetry。

## 依据与边界

[chi](https://github.com/go-chi/chi) 面向 `net/http`；[Fosite](https://github.com/ory/fosite) 是 OAuth/OIDC 框架；[Huma](https://huma.rocks/) 提供类型化 API 与 OpenAPI 能力。资料核对日期：2026-10-17。

Fosite 不是账号系统或整套 Logto 替代品。登录交互、同意、持久化、应用管理、MFA、退出和运营能力仍需自行组合。使用它不等于获得 OIDC 认证，支持面必须逐项核实。

## 理由与代价

标准 HTTP 便于集成协议库和测试；SQL 与协议适配仍会产生不少工程工作。Huma 的业务模型和 OAuth 原始响应需要分开，避免验证、错误包装或中间件修改标准行为。

## 替代方案

Gin、Fiber 等 HTTP 框架需要证明具体收益及协议适配成本；不以基准跑分决定身份架构。独立成熟授权服务器可减少协议适配，但会增加服务运维和产品整合成本。ZITADEL OIDC 等提供方库可作为比较对象，不能把仅支持客户端验证的库当作替代。

## 待验证与重新评估

检查所选版本的维护、许可证、安全公告、支持面和存储接口；验证事务与并发刷新、client credentials、发现文档、密钥轮换及退出集成。失败时记录缺口和替代成本，在继续实现前修订决策，不自己补写未审查的协议核心。

安全基线见 [identity-security.md](../identity-security.md)。首期不锁定未验证版本号。

2026-10-17 全栈复核结论：chi 继续推荐，Huma 限业务 API；Fosite 保持候选，重点验证 Ent 生产持久化、协议事务、客户端认证成本及刷新并发。不能因其提供协议 handler 就声称它附带生产存储或完整身份产品。资料和替代门槛见 [stack-review.md](../stack-review.md)，性能验收见 [performance.md](../performance.md)。
