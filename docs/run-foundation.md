# 工程基座运行指南

更新时间：2026-10-19。本文件维护工程基座命令，早期验证记录按当时阶段保留；后续 OAuth/OIDC、账号/应用管理和管理员安全已增加实现与验证，当前范围统一见 [实施进度](implementation-status.md)。仍不满足生产身份认证交付条件。日常启动和试用步骤见 [边开发边查看](live-development.md)。

## 本机前置条件

- Go 1.27、Node.js 24、pnpm 12.14.0。
- 依赖可运行在本机 Docker Desktop（Linux 容器）或局域网 Ubuntu；Windows 使用远程依赖时不需要安装 Docker。
- 已连接 `192.168.101.41:22`，用户 `misaka10843`，依赖目录 `/home/misaka10843/code-stack/moetopia-dev`。SSH 凭据不写入项目；当前远程 Docker 操作需要 `sudo`。
- PostgreSQL、Valkey 由本项目 Compose 提供，不连接 nextmoe 或生产数据库。当前远程三个容器已启动且健康。

Windows 下使用 PowerShell，进入仓库根目录。使用当前 Ubuntu 依赖时运行：

```powershell
pnpm dev:doctor:remote
```

远程体检检查本机工具和四个远程 TCP 端口，不打印秘密或自行安装软件，不代替数据库认证检查。本机 Docker 模式则使用 `pnpm dev:doctor`。

## 当前远程环境

本轮已生成本机 `.env` 与 `.env.compose`，未回显秘密；仅把 Compose 和 `.env.compose` 复制到上述 Ubuntu 专属目录，未上传应用加密密钥或 SSH 密码。目录权限为 700，依赖秘密文件为 600。开发库迁移和签名密钥初始化已执行，正常开发直接运行 `pnpm dev`，不要重复初始化密钥。

网络为 `moetopia-dev_dependencies`，只连接本项目的三个依赖容器，不复用服务器已有网络。发布端口绑定 `192.168.101.41`，均为容器端口加 50000；这不是防火墙隔离，服务器防火墙和公网路由本轮未修改。Mailpit 是开发邮件工具，不应将其端口映射至公网。

### 新环境初始化

首次配置新的 Windows 工作区时运行：

```powershell
pnpm install --frozen-lockfile
node scripts/setup-dev.mjs --services-host 192.168.101.41
```

脚本以独占写入方式创建 `.env` 和 `.env.compose`，不会覆盖既有文件。`.env.compose` 仅包含依赖绑定地址与数据库/Valkey 密码。新建环境时，将本项目 `compose.dev.yml` 和 `.env.compose` 通过 SSH/SCP 放入自己的专属依赖目录；不要复制整个 `.env`。已有 PostgreSQL 数据卷的密码不会随环境文件自动改变，必须保留匹配的配置，不能靠重新生成文件重置。

在 Ubuntu 的本项目目录执行：

```sh
chmod 600 .env.compose
sudo docker compose --env-file .env.compose -f compose.dev.yml config --quiet
sudo docker compose --env-file .env.compose -f compose.dev.yml up -d --wait
sudo docker compose --env-file .env.compose -f compose.dev.yml ps
```

Windows 首次使用新的开发数据库时执行：

```powershell
pnpm dev:doctor:remote
pnpm api:migrate
pnpm api:keys:init
pnpm test:integration
pnpm dev
```

停止远程依赖在 Ubuntu 本项目目录执行 `sudo docker compose --env-file .env.compose -f compose.dev.yml down`，默认保留数据卷。Windows 的 `dev:up`/`dev:down` 操作本机 Docker，不会自动连接 Ubuntu，不应在远程模式误用。

## 可选的本机容器模式

本机具备 Docker 且尚无 `.env` 时，使用默认回环地址：

```powershell
pnpm install --frozen-lockfile
node scripts/setup-dev.mjs
pnpm dev:up
pnpm api:migrate
pnpm api:keys:init
pnpm dev
```

`setup-dev.mjs` 创建被忽略的秘密文件，生成数据库/Valkey 密码和 32 字节加密主密钥，不打印值、不覆盖既有文件。备份该密钥；随意更换会导致已加密的协议状态和签名私钥无法解密。不要将 `.env` 提交或发送到聊天。

迁移针对 `.env` 中明确配置的本项目开发库 `moetopia_dev`。版本 1 创建 authorization schema，版本 2 独立新增 identity、sessions、notifications schema，旧版协议表不变；本轮开发库已应用至版本 2。服务不会自动创建表。`api:keys:init` 只初始化一次，存在活动密钥时拒绝重复操作，不充当轮换命令。

当前可通过账号页注册用户，`pnpm dev:invite` 只发行开发邀请码；没有默认管理员、固定账号密码或正式 OAuth client 创建命令。协议端点当前只装配 client_credentials 路径，其真实持久化与 JWT 行为已通过集成测试；不声明完整 OIDC discovery。

## 本地入口

| 服务 | 地址 |
| --- | --- |
| account | `http://account.localhost:3000` |
| admin | `http://admin.localhost:3001` |
| developer | `http://account.localhost:3000/developer` |
| API 存活/就绪 | `http://127.0.0.1:8080/healthz`、`/readyz` |
| OpenAPI | `http://127.0.0.1:8080/openapi.json` |
| Mailpit | `http://192.168.101.41:58025`，SMTP `192.168.101.41:51025` |
| PostgreSQL | `192.168.101.41:55432` |
| Valkey | `192.168.101.41:56379`，启用密码认证 |

`.localhost` 用于隔离应用 origin；若本机环境不能解析，应检查本地回环域名解析，不将生产域名指向开发服务。端口均为开发默认值，未修改参考仓库服务。

`pnpm dev` 启动 Go 构建监视器、用户站和独立管理台两个 Nuxt 开发进程；Go 文件保存后自动重建，Nuxt 热更新。启动器已实测兼容本机独立 pnpm.exe，端口占用时拒绝重复启动。Ctrl-C 清理本轮启动的进程，保留依赖容器。上表为当前远程模式；本机容器模式的依赖主机改为 `127.0.0.1`，端口不变。迁移文件变化不会自动修改数据库。

前端开发代理固定转发到 `127.0.0.1:8080`。admin/developer 的静态构建不包含该开发代理；真实生产入口与部署包将在后续阶段交付，不能将静态目录单独上线后假定 API 已接通。

## 无 Docker 也可执行的验证

```powershell
pnpm api:test
go -C apps/api vet ./...
pnpm api:generate
pnpm api:check:client
pnpm typecheck
pnpm build
pnpm storybook:build
pnpm exec playwright install chromium
pnpm test:browser
pnpm docs:check
```

基础浏览器测试启动专属本地预览服务，验证无后端页面、主题/语言和 Storybook 组件；它不代表账号链路测试。真实账号链路由下述独立命令验证。测试不复用已有同端口进程。

`pnpm api:generate` 只生成 Ent 模型；`pnpm api:generate:client` 从真实 Huma 路由生成 OpenAPI 和 TypeScript，`api:check:client` 只检查漂移。

## 数据库集成测试

当前远程依赖已可用，在 Windows 直接执行，无需本机 Docker CLI：

```powershell
pnpm test:integration
```

运行器要求 development 环境、`moetopia_dev` 库和 55432 端口，通过 pgx 创建随机命名的 `moetopia_test_*` 库，测试后只删除本轮库，不重置开发库或生产库。Go 工具校验角色、库名和操作，不接受任意 SQL 标识符。运行器显式设置 `TEST_DATABASE_DSN`、`TEST_VALKEY_ADDRESS` 和密码，缺失环境时失败，不以跳过替代通过。

当前集成测试覆盖迁移、Ent 的 schema 映射、密钥重开连接恢复、client_credentials JWT、错误权限/audience/凭据、邀请注册、验证码尝试上限、邀请码并发单次兑换、CSRF、登录会话轮换、所有权检查和退出失效。它不覆盖尚未实现的授权码或刷新流程。

真实 Mailpit + 浏览器账号流程：先构建 account 应用，再运行独立测试。

```powershell
pnpm --filter @moetopia/account build
pnpm test:account
```

运行器创建独立临时数据库、独立签名/加密密钥和测试邀请码，使用 4305/58080 专属本地端口；通过页面注册并读取真实 Mailpit 邮件，登录、刷新、退出。结束后清理测试数据库和仅本轮投递的测试邮件，不修改开发用户。测试中的 API 转发显式连接该临时 API，不借用日常 8080 服务。该测试关闭 trace 和截图，避免将验证码及密码写入调试产物。

## 依赖与已知限制

- 已锁定 pnpm 与 Go 依赖。Ent 生成工具的 displaywidth 间接依赖已升级至兼容版本；未修改模块缓存内源码。
- pnpm 12 使用 `allowBuilds`，只允许明确列出的 esbuild 构建脚本；不能继续使用已移除的旧配置。
- 为 unctx 3.0.1 补充其要求的 oxc-parser 0.140.0，避免绑定到 Nuxt 自身使用的旧 parser。共享层构建与类型检查已验证；未来上游修复后重新评估该 package extension。
- Compose 三个镜像已在 Ubuntu 拉取并运行；真实集成测试通过。CI workflow 仍未在 GitHub 执行，容器运行通过不代表镜像漏洞审计通过。
- Storybook 的文档引擎弃用提示及其开发构建体积提示仍存在，不属于生产前端包的体积结论。
- 依赖安全扫描的修复与未修复告警见 [dependency-audit.md](dependency-audit.md)，不能将构建通过描述为所有依赖安全检查通过。
