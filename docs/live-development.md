# Windows 本地开发

Windows 原生运行 Go、两个 Nuxt 应用和 mprocs TUI。WSL 的 `docker` 发行版运行 PostgreSQL、Valkey 和 Mailpit。首次使用需准备仓库声明的 Go、Node 和 pnpm 版本。

## 日常启动

在 **Windows Terminal 的 PowerShell 7** 中进入仓库根目录。首次安装依赖，之后可直接运行第二条命令：

```powershell
pnpm install --frozen-lockfile
pwsh -File scripts/dev-local.ps1
```

脚本在缺少配置时创建本地 `.env`，启动独立的 `moetopia-local-dev` Compose 项目，应用迁移并初始化签名密钥与两个站点的内置客户端。交互终端进入 mprocs，非交互终端使用纯文本日志。脚本要求 `.env` 指向本机开发依赖，保留已有文件及密钥。已配置远程依赖的工作区继续使用 `pnpm dev`。

在自己的 PowerShell 终端运行，并保留该终端：

```powershell
cd D:\misaka10843\Documents\Github\moetopia-AIO
pnpm dev
```

交互式终端中进入 mprocs：左侧分别显示 api、account、admin 的进程状态；account 同时提供账号与开发者页面，右侧只显示所选服务的输出。某个服务退出后保留日志和退出码，其他服务继续运行；修复原因后可以单独重启。推荐在 Windows Terminal / PowerShell 使用。

快捷键在**左侧服务列表获得焦点时**使用：

| 按键 | 操作 |
| --- | --- |
| ↑ / ↓ 或 k / j | 选择服务，切换日志 |
| Tab / Ctrl-A | 在服务列表与输出之间切换焦点 |
| r | 重启选中服务；已退出的服务重新启动 |
| x / s | 停止 / 启动选中服务 |
| PageUp / PageDown | 翻阅当前输出 |
| z | 放大当前输出 |
| v | 进入复制模式；按 Esc 离开 |
| q / Ctrl-C | 退出本轮开发实例并停止它启动的服务；Ctrl-C 在普通输出焦点也有效 |

UP 表示进程运行，不代表数据库就绪、编译无错误或业务验收通过。Go 后续编译失败会保留旧 API，在 api 输出查看错误；首次构建失败或启动退出时显示 DOWN，修正后按 r。TUI 保留每个服务最多 10000 行终端回滚，不自动将日志写入磁盘。

只启动需要的服务，或不用 TUI：

```powershell
pnpm dev api
pnpm dev api account
pnpm dev account storybook
pnpm dev:plain
pnpm dev:plain api account
pnpm dev --help
```

Storybook 只在显式选择时启动；`pnpm dev account storybook` 不启动 API，真实登录需已有自己的 API 实例。非交互终端自动使用纯文本模式，各行带服务名前缀；纯文本模式没有快捷重启，退出后可按所需服务重新运行。启动器、依赖及验证记录见 [ADR-0016](decisions/0016-development-tui.md)。

| 入口 | 用途 |
| --- | --- |
| `http://account.localhost:3000` | 无插画登录/注册页；登录后进入账号资料 |
| `http://admin.localhost:3001` | 管理台，使用已初始化的管理员账号及 TOTP |
| `http://account.localhost:3000/developer` | 同一用户站中的开发者门户、应用准备与接入说明 |
| `http://127.0.0.1:58325` | 一键启动的本地 Mailpit |
| `http://192.168.101.41:58025` | 远程依赖模式的 Mailpit |

账号页请使用表中的 `account.localhost`，不要改用 Nuxt 控制台显示的 `127.0.0.1:3000`：当前 Go 严格检查 `.env` 中的账号 origin，地址不同会拒绝写请求。账号与开发者分区同源共享账号状态；admin 使用独立 host，不能据此继承管理员权限。

发送验证码若返回 403 `Request verification failed`，先核对地址栏是否为 `http://account.localhost:3000`，`http://localhost:3000` 同样不属于该 Origin。直接打开 `http://account.localhost:3000/auth/register` 并刷新后重试，不要复制旧标签页的验证码请求或修改后端校验。account 启动日志现在明确输出配置的登录入口；Nuxt 自带的别名地址不代表后端允许的 Origin。地址正确而仍失败时，再检查 context 请求是否成功、浏览器是否接受 Cookie，以及页面是否持有最新 CSRF；不要把这些凭据发到聊天中。

Vue、样式和翻译保存后由 Nuxt 自动更新。Go `.go`、迁移 `.sql`、go.mod/go.sum 变化触发 API 重新构建和重启；编译失败时保留上一版 API，错误显示在终端，修正后再保存即可。后端重启期间的请求可能失败，等待就绪后手动重试。开发启动器不会自动应用迁移。

本轮改为用户站＋管理台，建议在自己的原启动终端退出并重新运行一次 `pnpm dev`，使 TUI 列表与配置更新。仅看前端可运行 `pnpm dev account` 或 `pnpm dev admin`；这不会启动 Go，账号服务不可用时显示连接失败。不要启动旧 developer 命令，也不要让其他对话结束归属不明的进程。

认证页为 `/auth/login`、`/auth/register`。登录后可从菜单进入个人资料、会话和安全管理。开发者分区提供应用管理和接入文档，入口为 `/developer`。

2026-10-19 新增用户站自身的 Go OAuth RP：可从正常页面登录入口或 [直接发起 OAuth 登录](http://account.localhost:3000/api/v1/portal/login?return_to=/developer/applications) 试用真实启动、授权与回调，身份会话和用户站会话分开。正常入口/守卫已由前端接入并通过真实浏览器回归。迁移 5 和受控客户端初始化已在专属开发库执行，完整契约与其他工作区命令见 [接入说明](run-first-party-oidc.md)。资料/恢复和开发者新增接口见 [账号管理](run-account-management.md) 与 [开发者应用](run-developer-applications.md)，新表单仍需前端接入。

跨应用地址来自 `NUXT_PUBLIC_ACCOUNT_URL`、`NUXT_PUBLIC_ADMIN_URL`。开发者入口使用用户站下的 `/developer`，不再读取独立开发者地址；旧 `.env` 中该变量不再生效，无需更改密钥或数据库。admin 为静态构建，应在构建环境配置公共用户站地址，未配置时跨应用链接禁用。这些公共地址不替代后端 Origin、回调或权限检查。

脚本只检查选中服务的端口：前端 3000/3001、Storybook 6006，API 从 MOETOPIA_LISTEN 读取（默认 8080）；每次启动/重启再次检查。不会抢占、杀死已有进程或让 Nuxt 悄悄换端口。提示占用时先检查是否已在另一终端运行了本项目；只在归属明确的原终端停止。一个工作区默认保持一份 `pnpm dev` 即可；不同终端按服务拆开时，不能重复启动相同服务。

停止自己的开发实例：TUI 在服务列表按 q，或按 Ctrl-C；纯文本模式按 Ctrl-C。仅停止该实例启动的进程树，Ubuntu 依赖继续运行。不要重复初始化 `.env`、密钥或删除数据卷。安装了新依赖的其他工作区先执行 pnpm install。

## 试用注册和登录

若已用 `scripts/dev-local.ps1` 启动本地项目，直接在源码仓库的 PowerShell 执行 `pnpm dev:invite`。然后到 [注册页](http://account.localhost:3000/auth/register) 使用邀请码，在 [Mailpit](http://127.0.0.1:58325/) 查看邮箱验证码。

用 `pwsh -File scripts/dev-local.ps1` 启动的环境中，依赖容器在 WSL 运行，Go 和 Nuxt 在 Windows 运行。发码和管理员初始化都在项目目录的 PowerShell 执行。

没有预置的固定测试账号或默认管理员；此前自行注册的开发账号可以继续登录。自动化测试账号属于临时数据库，测试结束即清理，不能作为日常账号使用。邀请码与邮箱验证码不同，分别按下面步骤获取。

1. 在另一个 PowerShell 终端进入同一目录，运行 `pnpm dev:invite`。命令创建一个默认 24 小时有效、单次使用的开发邀请码，并只向该终端输出一次原文；数据库仅保存哈希。不要把邀请码贴入 issue、日志或提交。
2. 在账号页选择“创建账号”，填写邮箱、昵称、邀请码和至少 15 字符的密码。可使用例如 `reader@moetopia.test` 的测试邮箱。
3. 点击“发送验证码”，在 Mailpit 找到该收件人的注册邮件，将 8 位验证码填回页面。当前邮件只进入开发 SMTP，不会投递至真实外部邮箱。
4. 创建成功后用邮箱和密码登录。刷新应保持登录；可在会话区结束会话，或点击退出登录。

注册默认邀请制。首个开发账号验证邮箱后，可按 [管理员初始化](run-administration.md#运行与一次性初始化) 获得管理角色，再绑定 TOTP。管理员可在管理台切换注册模式。测试 GitHub 或 Google 登录时需先配置对应供应商。 [第三方登录配置](run-federation.md)

`dev:invite` 仅允许 development 环境，不提供默认管理员或固定账号密码。开发依赖文件和应用加密密钥均不提交。

## 单独查看组件

在另一个终端运行：

```powershell
pnpm storybook
```

打开 `http://localhost:6006`，在 Overview 查看组件与品牌工作台，在 Layouts 查看页框、认证页布局和工作区组合。Storybook 不依赖真实身份账号。自动浏览器测试使用独立的 53000/53001/56006 端口，不占用日常开发或 Storybook 端口。

## 数据变更与故障

开发中新增迁移时，在另一终端确认目标为自己的开发库，再运行 `pnpm api:migrate`。如果 API 因缺迁移退出，在 TUI 选中 api 按 r。旧授权记录需要显式回填时，见 [授权接入](run-authorization.md)。

账号服务不可用时先看 TUI 中的 api 日志，检查 `http://127.0.0.1:8080/readyz`。本地依赖用 `wsl.exe -d docker -- docker compose -p moetopia-local-dev ps` 查看。远程依赖用 `pnpm dev:doctor:remote` 检查端口。

反馈时提供页面地址、操作步骤、主题/语言、期望效果和实际表现即可。截图可包含界面，但应遮挡密码、验证码、邀请码和个人资料。前端修改通常即时可见；服务端迁移和启动脚本变化会明确说明是否需要操作。

## 远程依赖与验证

新建远程开发环境时，在 Windows 运行 `node scripts/setup-dev.mjs --services-host <开发主机地址>`。把生成的 `.env.compose` 和 `compose.dev.yml` 放到远程主机的项目专属目录。在该主机运行 `sudo docker compose --env-file .env.compose -f compose.dev.yml up -d --wait`。Windows 首次连接专属库时依次运行 `pnpm api:migrate`、`pnpm api:keys:init`、`pnpm api:first-party:init`、`pnpm api:admin:init`，然后运行 `pnpm dev`。

日常变更可运行 `pnpm api:test`、`pnpm typecheck` 和 `pnpm docs:check`。修改 API 契约后运行 `pnpm api:generate:client`，再用 `pnpm api:check:client` 检查漂移。数据库集成测试用 `pnpm test:integration`，运行器创建并清理专属临时测试库。浏览器测试和发布验证见 [测试与发布](testing-release.md)。
