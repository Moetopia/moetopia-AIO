# ADR-0010：ORM 优先与 Ent/GORM 比较

- 状态：Ent + 独立版本化 SQL 迁移已确认为文档基线；Goose 优先评估，兼容与性能待验证。
- 记录与方向确认日期：2026-10-17。
- 确认依据：用户保留 chi/Fosite/Huma 候选栈，数据层改为 ORM 优先，再比较 Ent/GORM。
- 实施状态：仅资料比较，没有安装、生成或运行。
- 第三轮要求：以长期专业化、工程化及商业项目维护判断；当时未指定 Ent/GORM，详见 [长期工程原则](../long-term-engineering.md)。
- 后续确认：2026-10-17，用户选择 Ent 方案作为文档基线，要求同时复核全栈的长期工程适配与性能。确认不构成原型或业务实施授权。

## 比较依据

以下为官方资料能力说明与本项目适配判断，资料核对日期为 2026-10-17；不代表实测性能或维护成本排名。

| 维度 | Ent | GORM |
| --- | --- | --- |
| 建模 | 以 schema 声明字段、索引和 edges，并生成实体及查询 API | 以 Go struct、约定和标签建模，提供传统及泛型调用 API |
| 变更工作流 | schema 变化后更新生成代码，需要固定生成器并检查漂移 | 常规 CRUD 可直接围绕模型维护；动态字段和更新语义需要仔细审查 |
| 模块边界 | 类型与查询清楚，但全局 edges 容易制造跨模块遍历 | 容易融入 repository，但关联及 preload 同样可能跨模块 |
| 事务 | 提供事务与事务 client；需要验证协议存储与原始 SQL 如何加入同一事务 | 提供事务回调、手动事务及嵌套事务能力；仍需显式定义业务原子边界 |
| 原始 SQL | 复杂操作需要在适配层使用驱动或扩展能力，不能假设所有 SQL 都由生成 API 表达 | 提供 SQL Builder/Raw；不意味着可绕过参数化、权限或事务 |
| 迁移 | 官方有基于 Atlas 的版本化迁移工作流，需确认所用能力的版本、许可与本地执行方式 | 提供 AutoMigrate/Migrator，也可配合独立版本工具；本项目禁用服务启动自动迁移 |
| 适配本项目 | 倾向明确模型、生成 API 和长期模块维护；代价是代码生成和 schema/迁移工具链 | 接近 nextmoe 的现有使用方式，常规后台开发直接；需约束零值更新、关联与 hook 隐式行为 |

能力来源：[Ent schema](https://entgo.io/docs/schema-def/)、[Ent 事务](https://entgo.io/docs/transactions/)、[Ent 版本迁移](https://entgo.io/docs/versioned-migrations/)、[GORM 概览](https://gorm.io/docs/)、[GORM 事务](https://gorm.io/docs/transactions.html)、[GORM 更新语义](https://gorm.io/docs/update.html)、[GORM SQL Builder](https://gorm.io/docs/sql_builder.html)、[GORM 迁移](https://gorm.io/docs/migration.html)。

## 已确认方案与理由

已确认文档基线：**Ent 作为目标 ORM，采用独立、经过审查的版本化 SQL 迁移；优先评估 Goose 执行迁移，不把 Atlas Pro 作为默认必需项。** 具体版本、驱动与迁移组合尚未实测；Goose 是优先验证对象，而非已安装或已通过兼容检查的工具。

选择依据是长期模型演进：显式 schema 与生成查询 API 让字段及查询变更更易在构建和评审时发现；固定生成器并在 CI 检查漂移，有助于多模块持续维护。接受生成步骤作为工程成本，不把减少几个首期步骤当作主要收益。

同时，GORM 可以通过规范、生成工具与测试建设长期商业系统。推荐 Ent 是项目适配判断，不能推出“GORM 不专业”或“Ent 自动更安全”。

如果更重视延续 nextmoe 的 GORM 维护经验、减少生成步骤，GORM 是合理选择。GORM 已提供泛型 API，不应把它描述为完全没有类型约束；但泛型本身也不能消除更新遗漏或字段选择错误。

## 长期风险与商业依赖核对

1. **维护与许可**：Ent 官方仓库当前列明由 Atlas 团队维护，许可为 Apache-2.0；GORM 为 MIT。仓库信息不能代替具体版本及全部传递依赖审计，也不承诺任一项目永久维护。来源：[Ent 仓库](https://github.com/ent/ent)、[GORM 仓库](https://github.com/go-gorm/gorm)。
2. **迁移成本**：Ent 官方跨数据库 schema 迁移流程标为 Atlas Pro；Atlas Community 与标准发行物的能力、许可不同。因此不能将 Ent、Atlas 和所有企业能力当作同一个免费包。来源：[Ent 多 schema 迁移](https://entgo.io/docs/multischema-migrations/)、[Atlas Community](https://atlasgo.io/community-edition)。核对日期：2026-10-17。
3. **推荐规避方式**：迁移执行采用独立 SQL 历史与单一账本；Ent schema 和运行时 schema 映射必须另行验证。Goose 可执行审查后的 SQL，但不自动提供 Ent 的多 schema diff 能力。若需自维护迁移，明确计入审查和漂移验证成本，不能声称成本为零。
4. **模块隔离**：不能因为 Ent 支持 edges 就生成全系统可遍历的数据图。生成包与模块接口边界需要设计并检查；协议事务可能跨适配对象，不能通过私自启动第二个事务解决。
5. **退出路径**：业务 DTO、用例与授权语义不依赖 Ent 类型。更换 ORM 可以保留数据与 SQL 迁移历史，但仍需改写存储适配并重跑集成测试；不承诺无成本替换。

若多 schema 运行时或事务适配无法以可维护成本实现，优先重新比较持久化布局、迁移预算与 GORM，不能擅自放弃领域所有权，也不应为了保住 Ent 选择而购买未确认服务。

## 无论选择哪种都保持的约束

- 首期只选一种 ORM，不按模块混用 Ent/GORM，不并行引入 sqlc 维护同一批 CRUD。
- ORM 持久化实体不得进入 API 契约或越过模块公开边界。跨模块关联通过稳定 ID 与拥有者接口完成。
- 原子消费授权码、刷新轮换、单次恢复码、邀请码及 outbox 都要由数据库事务和约束证明。
- 必要 SQL 留在存储适配层，与 ORM 使用同一事务；不在 hook 中暗中进行权限判定或外部通知。
- schema 声明目标模型，已审查的版本迁移表达演进历史；模型与迁移漂移要检查。禁止运行时自动修改生产表。

## 待验证与迁移工具选择

获实施授权后验证：Fosite 存储接口与 ORM 事务的整合、条件更新和行数判断、模块 schema、类型与空值、代码生成/查询维护、Windows 开发及迁移锁。

按当前建议先验证 Ent + 独立版本 SQL/Goose：重放迁移后的结构与 Ent 目标模型对照，检查多 schema 与 PostgreSQL 特定索引等能力。不依赖 Pro 的迁移生成路径若不可行，保留手写 SQL 与结构对照方案并评估总成本。若将来采用 Atlas，明确发行物、能力和预算，再重新决定执行器；同一 schema 始终只有一个迁移账本。

ORM 选择不自动确认 Atlas、Goose、驱动或商业服务。所需命令的实际版本、授权条件、是否依赖云账号和 Windows 可用性仍需核对，文档不预设必须购买服务。
