# ADR-0007：Windows 开发与 Linux 容器发布

2026-10-18 更新：前端数量、目录和默认启动列表已由 [ADR-0017](0017-user-site-layouts.md) 替代，当前为 account 用户站（含 developer）＋独立 admin；以下保留当时的工程或验证记录。

- 状态：建议；Windows 原生开发、Compose 依赖、Linux 容器方向已确认。
- 记录日期：2026-10-17。

## 建议决定

PowerShell 启动 Go 和 Nuxt；Docker Compose 在 bridge 网络提供 PostgreSQL、候选 Redis 与 Mailpit。宿主进程连接已映射的 localhost 端口，容器间通过服务名连接；不使用 host networking，不要求 WSL。

生产建议单机 Compose、TLS 反向代理、预构建不可变镜像和一次性迁移任务。CI 构建、镜像仓库、代理产品及生产资源尚未选定。

用户补充生态活跃最多双位数、硬件未定，支持以低空闲开销的单机部署为优先建议。保留三个前端的逻辑/发布边界，但不强制三个 SSR 常驻进程；worker 不需要独立机器。KV 缓存、追踪采集服务等依赖需说明首期实际用途，不能仅以未来商业化为由默认启动。没有据此指定机器规格或承诺容量。

## 理由与代价

开发环境符合当前 Windows 使用方式，并避免 Docker VM loopback 与宿主混淆。单机部署便于起步，但不是高可用承诺；容量、备份、监控与升级停机需要说明。

## 待验证

Windows 路径与空格、环境变量加载、Ctrl-C 进程清理、热更新、Go 依赖是否要求 cgo、容器访问地址与测试资源隔离。镜像用户、健康检查和迁移门禁须实际验证后写成操作指南。

## 重新评估条件

当 RTO/RPO、流量、运维团队或隔离需求超出单机能力时，评估托管数据库、多节点或编排平台。不能在需求未明确前引入 Kubernetes。

初始文档阶段未创建启动或部署脚本。2026-10-18 已建立开发脚本与 Compose 定义，后续记录见 [ADR-0013](0013-implementation-baseline.md)；生产部署包仍未交付。开发设计见 [development.md](../development.md)，发布要求见 [testing-release.md](../testing-release.md)。
