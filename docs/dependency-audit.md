# 依赖审计记录

日期：2026-10-18。范围为当前工程基础，不是完整身份系统或生产部署的安全认证。

## Go

使用 `govulncheck v1.9.0` 检查实际代码调用。初次发现 JOSE 和遥测 exporter 的已知问题，已将 go-jose/v3 升级至 3.0.5、相关 OTLP/Zipkin exporter 升级至 1.47.0。升级后 Go 测试通过，复扫结果为 **0 个可达漏洞**。

模块层仍报告 `GO-2026-5932`：`golang.org/x/crypto/openpgp` 未维护。当前项目没有导入或调用该包；使用同模块的其他包不等于使用 OpenPGP。保留报告，不声称整个依赖图绝对无漏洞。来源：[Go 漏洞记录](https://pkg.go.dev/vuln/GO-2026-5932)。

Ent 的 CLI 通过 Go tool 依赖锁定，displaywidth 0.11.0 修复了本机遇到的生成工具兼容问题；没有修改下载缓存中的第三方源码。Fosite 的部分遥测间接依赖存在上游弃用提示，后续协议集成仍需关注维护与升级成本。

2026-10-19 自有 RP 增加 coreos/go-oidc/v3 v3.21.0（间接 go-jose/v4 v4.1.4），将既有 x/oauth2 v0.36.0 作为直接依赖；没有替换 Fosite 授权服务器。实际 `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` 复扫通过：0 个可达漏洞、0 个导入包漏洞、1 个既有模块级告警。Go 单元与数据库集成、真实浏览器协议测试通过；此结论仅覆盖当前代码，不作为生产安全认证。

同日账号管理将 golang.org/x/image 升级并作为直接依赖锁定 v0.47.0，用于头像解码后缩放；`go mod tidy` 后复扫仍为 0 个可达、0 个导入包及 1 个模块级告警。已有 npm 告警未在此轮重新扫描，不把 Go 结果扩展为 JavaScript 或镜像检查通过。

管理员 MFA 使用 pquerna/otp v1.5.0（Apache-2.0），带入 barcode 间接依赖；未手写服务端 TOTP 算法。已核对库接口与 RFC 6238，`go mod tidy` 后 govulncheck v1.9.0 复扫仍为 0 个可达、0 个导入包和 1 个既有模块级告警。独立浏览器测试中的 Node HMAC 只作为测试客户端生成验证码，不用于产品服务端验证。npm 与镜像本轮未复扫。

## JavaScript

当前本机 npm 镜像不提供 audit endpoint，审计单次显式使用官方 registry，没有更改用户全局 registry。

初次生产依赖图扫描报告 3 项 high：glob CLI 注入、node-forge 签名校验及 braces 深层嵌套耗尽栈。

- glob 的受影响 10.x 路径通过精确版本范围 override 升级为 10.5.0。官方公告将 10.5.0 列为已修复版本；保持同一 major，避免为消除提示随意跳到不兼容版本。来源：[glob 公告](https://github.com/advisories/GHSA-5j98-mcp5-4vw2)。该旧 major 仍有上游弃用提示。
- node-forge 由 Nuxt → Nitro → listhen 带入；当前审计未列修复版本。来源：[node-forge 公告](https://github.com/advisories/GHSA-86w9-cpqp-85rv)。
- braces 由 Nitro 的文件匹配工具链带入；当前审计未列修复版本。来源：[braces 公告](https://github.com/advisories/GHSA-vfj7-8cjw-p6xm)。

当前账号服务的已生成生产 manifest 未列 node-forge/braces；JWT 在 Go 侧签发，不使用 node-forge。文件匹配与开发服务不接收产品用户提供的任意 glob 或签名数据。但这些观察不等于全面不可达证明：保留两项未修复告警，不新增静默忽略列表，不宣称 npm audit 全通过。

真实互操作阶段新增开发依赖 openid-client 6.8.8（用于独立 RP 验证）。2026-10-18 执行 `pnpm audit --json --registry https://registry.npmjs.org`，仍为上述 node-forge/braces 两项 high，新增客户端路径未出现告警；未修改全局 registry。

正式发布前重新检查上游修复、最终镜像和实际可达路径，并记录修复或经过明确评审的处理结果。当前工程本就不满足生产身份系统交付条件，不因构建通过放行上线。

## 复核命令

```powershell
go -C apps/api run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
pnpm audit --prod --registry=https://registry.npmjs.org
```

第三方数据库和 advisory 会变化；本记录只反映上述日期，不代替未来每次发布检查。开发容器镜像已在 Ubuntu 拉取并运行，尚未进行镜像漏洞扫描。
