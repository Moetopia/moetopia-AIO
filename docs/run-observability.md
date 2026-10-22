# 运行指标与探测

API 提供内部 `GET /internal/metrics`，Prometheus 文本格式；默认开发 API 可直接访问，生产 API 不发布端口，Caddy 不转发该路径。采集器需受控接入本项目私有网络；不为了抓取把数据库/API 或完整内网开放到公网。

指标包含固定路由分组的 HTTP 请求数、失败数与耗时桶，固定后台任务的执行/失败/耗时，以及 Go 堆和 goroutine。仅使用有界枚举，不将 URL 参数、邮箱、UUID、token 或 client secret 作为标签。重启计数器归零；HTTP 失败不等于业务拒绝事件数，指标不代替审计。邮件重投/失败积压使用[管理员统计接口](run-admin-statistics.md)的真实队列状态。

`deploy/probe.mjs` 可对 status、匿名 account context、OIDC discovery 发起有界并发读取并计算 p50/p95/p99：

```bash
node deploy/probe.mjs https://account.example.test 100 4
```

上述地址需替换为已配置真实入口。脚本不保存响应正文、Cookie 或凭据，不进行登录/改密/令牌签发负载。内部 CA 的测试专用选项只用于 `.test`/回环隔离验证，正式 TLS 不跳过校验。发布演练在随机私有目录保存 probe.json 和容器资源快照。

这些样本只作为启动后的匿名接口冒烟结果，不能推断真实登录容量、Argon2 并发、长期内存、生产延迟 SLO 或硬件规模。上线后结合实际机器按[性能规范](performance.md)验证混合业务负载和故障恢复，再决定资源上限与告警阈值。当前不预装 Prometheus/Grafana 等常驻服务，避免为低流量增加无依据的运行成本。
