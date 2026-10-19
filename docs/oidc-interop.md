# 独立 OIDC 客户端的真实浏览器验证

更新时间：2026-10-18。状态：Windows 上 1 项真实端到端场景通过，覆盖下表多个分支；不代表完整身份产品、OIDC 认证或生产验收。

## 本轮完成

使用开发依赖 openid-client 6.8.8 作为独立 RP，Chromium 实际经过 account 的 Nitro 代理、Go API、专属 PostgreSQL 和 Mailpit。没有拦截浏览器请求或用模拟响应替代身份业务。

| 分支 | 已取得的证据 |
| --- | --- |
| Discovery | 经过 Nuxt 的内容与专属 Go API 相同，标准客户端按 issuer 发现端点 |
| 首次登录与同意 | 从 RP 发起 PKCE/nonce/state；浏览器邀请注册、真实邮件验证码、密码登录、显式同意、回调换码 |
| 会话旋转 | 登录后 account Cookie 改变，授权事务 Cookie 保持；RP 不收到另一 host 的 account Cookie |
| 令牌校验 | 独立库启用 JWKS 验签，验证 ID Token；UserInfo subject 和获准资料匹配 |
| 拒绝 | 用户点击拒绝，独立库识别 access_denied，并校验 state |
| 重新认证 | 已登录时 prompt=login 重新输入密码，再显式同意成功 |
| 重放 | 再次兑换同一码得到 invalid_grant，原 access token 的在线 UserInfo 校验失败 |
| RP 的异常拒绝 | RP 使用错误的期望 state/nonce，或测试性篡改 ID Token 签名时拒绝登录 |
| 刷新轮换 | 显式请求 offline_access，独立库刷新成功，refresh 凭据变化，sub/auth_time 与 UserInfo 保持一致 |
| 刷新重用 | 重用先前 refresh 返回 invalid_grant，当前 access 的 UserInfo 在线校验失败 |
| 协议撤销 | 独立库调用 RFC 7009 撤销并重复撤销成功，之后刷新被拒绝 |
| Go 自有 RP | 独立 portal 登录经过真实 Nitro/Fosite/Go 回调，建立 HttpOnly 独立会话，恢复 developer 目的页 |
| RP 本地退出 | portal/context 从匿名变为已认证；退出后变回匿名，身份 account 会话仍保留 |

## 本轮修复

1. 当前共享翻译入口缺少已有 authorizationMessages 导出，导致 account 构建失败；只恢复该导出，未改页面逻辑或样式。
2. `/oauth/**` 的 Nitro 代理原先自动跟随上游重定向，授权入口的中间 Set-Cookie 未交给浏览器，详情请求返回 404。代理显式使用 `fetchOptions.redirect = manual`，让浏览器处理 303、Location 和事务 Cookie。测试保留源配置中的代理选项，以防重现此缺陷。

上述两处属于联调所需的共享接线修复，已写入 [后端交接](coordination/backend.md)；不改写前端对话的状态文件。页面行为继续由前端维护。

2026-10-19 自有 Go RP 联调同样发现 `/api/**` 需要 manual redirect：用户站登录启动与回调位于该前缀，服务端跟随会丢失中间 Cookie。已同步固定代理配置并重新通过完整场景。测试库增加迁移 5 和显式受控 RP 初始化；普通页面入口/守卫仍待前端接线，说明见 [自有 RP 接入](run-first-party-oidc.md)。

## 自行运行

在项目根目录，沿用现有 .env、Go/Node/pnpm 和远程开发依赖：

```powershell
pnpm test:oidc
```

测试固定使用 account 54305、RP 54310、Go API 58490；启动前检查，端口被占用时拒绝运行，不接管已有服务。可与 3000/3001/8080 的日常开发实例并行；本测试不修改 .env、开发库或签名密钥。

运行器新建随机 moetopia_test 数据库、合成账号/邀请/公开客户端和临时密钥。Go fixture 仅接受 test 环境和与显式测试名一致的数据库，不创建默认开发/生产客户端。独立测试构建位于 `.data/oidc-<pid>-<random>`，不覆盖应用 .nuxt/.output；Nitro 构建钩子先核对源代理路径，再仅替换上游端口，保留其余代理选项。

退出时清理自己的测试进程、随机数据库、构建产物及该合成收件人的测试邮件。token、verifier 和回调仅存在 RP 进程内存；不把它们输出到结果页面。浏览器 trace、截图和视频关闭，临时失败上下文也随本轮产物清理。浏览器下载或运行环境缺失时，该命令失败，不能记录为通过。

RP 在本测试的 loopback HTTP 明确启用独立库的开发例外；不据此放宽生产 HTTPS 策略。脚本是测试设施，不是可上线 RP 示例或长期运行服务。

## 验证限制与后续

最初互操作轮次没有新增迁移；后续刷新/撤销轮次增加迁移 4，并重新通过扩展场景，规则见 [ADR-0018](decisions/0018-refresh-token-rotation.md)。公开内省、RP 退出、联邦登录、开发者审核、管理员 MFA、生产邮件、TLS 代理、Safari/Firefox、正式目标硬件负载和 GitHub CI 的真实互操作仍未验收。当前业务接口未发生变化，无需前端适配新的 DTO。

Go 测试、vet 和 account 类型检查通过。新增独立客户端依赖后使用官方 registry 复查 audit，仍为已记录的 node-forge/braces 两项 high，未出现该客户端路径的新告警；不能称为审计全通过，详见 [依赖审计](dependency-audit.md)。

实现依据：[openid-client 验签选项](https://github.com/panva/openid-client/blob/v6.8.8/docs/functions/enableNonRepudiationChecks.md)、[H3 代理选项](https://github.com/unjs/h3/blob/v1.15.11/src/utils/proxy.ts)。
