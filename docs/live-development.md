# 边开发边查看效果

更新时间：2026-10-18。当前 Windows 运行 Go/Nuxt，依赖由 Ubuntu `192.168.101.41` 提供。无需在 Windows 安装 Docker。本文面向当前已有 `.env`、依赖容器、迁移和签名密钥的工作区；首次设置见 [工程指南](run-foundation.md)。

## 日常启动

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
| `http://admin.localhost:3001` | 管理工作区与功能准备状态，管理操作尚未开放 |
| `http://account.localhost:3000/developer` | 同一用户站中的开发者门户、应用准备与接入说明 |
| `http://192.168.101.41:58025` | 开发邮件收件箱 |

账号页请使用表中的 `account.localhost`，不要改用 Nuxt 控制台显示的 `127.0.0.1:3000`：当前 Go 严格检查 `.env` 中的账号 origin，地址不同会拒绝写请求。账号与开发者分区同源共享账号状态；admin 使用独立 host，不能据此继承管理员权限。

发送验证码若返回 403 `Request verification failed`，先核对地址栏是否为 `http://account.localhost:3000`，`http://localhost:3000` 同样不属于该 Origin。直接打开 `http://account.localhost:3000/auth/register` 并刷新后重试，不要复制旧标签页的验证码请求或修改后端校验。account 启动日志现在明确输出配置的登录入口；Nuxt 自带的别名地址不代表后端允许的 Origin。地址正确而仍失败时，再检查 context 请求是否成功、浏览器是否接受 Cookie，以及页面是否持有最新 CSRF；不要把这些凭据发到聊天中。

Vue、样式和翻译保存后由 Nuxt 自动更新。Go `.go`、迁移 `.sql`、go.mod/go.sum 变化触发 API 重新构建和重启；编译失败时保留上一版 API，错误显示在终端，修正后再保存即可。后端重启期间的请求可能失败，等待就绪后手动重试。开发启动器不会自动应用迁移。

本轮改为用户站＋管理台，建议在自己的原启动终端退出并重新运行一次 `pnpm dev`，使 TUI 列表与配置更新。仅看前端可运行 `pnpm dev account` 或 `pnpm dev admin`；这不会启动 Go，账号服务不可用时显示连接失败。不要启动旧 developer 命令，也不要让其他对话结束归属不明的进程。

认证页为 `/auth/login`、`/auth/register`，登录后进入 `/profile`。账号中心通过顶栏或手机菜单进入 `/profile`、`/sessions`、`/security`；`/` 兼容原入口与授权请求。开发者分区为 `/developer`、`/developer/applications`、`/developer/docs`，见 [ADR-0017](decisions/0017-user-site-layouts.md)。会话页复用现有查询与撤销接口；安全页展示现有会话入口及尚未开放的密码修改、第三方绑定、授权管理说明，不代表这些后端能力已完成。正面立绘暂不用于页面，保留角色配色。前后端分工和新对话接手说明见 [交接文档](workstreams.md)。

2026-10-19 新增用户站自身的 Go OAuth RP：可从正常页面登录入口或 [直接发起 OAuth 登录](http://account.localhost:3000/api/v1/portal/login?return_to=/developer/applications) 试用真实启动、授权与回调，身份会话和用户站会话分开。正常入口/守卫已由前端接入并通过真实浏览器回归。迁移 5 和受控客户端初始化已在专属开发库执行，完整契约与其他工作区命令见 [接入说明](run-first-party-oidc.md)。资料/恢复和开发者新增接口见 [账号管理](run-account-management.md) 与 [开发者应用](run-developer-applications.md)，新表单仍需前端接入。

跨应用地址来自 `NUXT_PUBLIC_ACCOUNT_URL`、`NUXT_PUBLIC_ADMIN_URL`。开发者入口使用用户站下的 `/developer`，不再读取独立开发者地址；旧 `.env` 中该变量不再生效，无需更改密钥或数据库。admin 为静态构建，应在构建环境配置公共用户站地址，未配置时跨应用链接禁用。这些公共地址不替代后端 Origin、回调或权限检查。

脚本只检查选中服务的端口：前端 3000/3001、Storybook 6006，API 从 MOETOPIA_LISTEN 读取（默认 8080）；每次启动/重启再次检查。不会抢占、杀死已有进程或让 Nuxt 悄悄换端口。提示占用时先检查是否已在另一终端运行了本项目；只在归属明确的原终端停止。一个工作区默认保持一份 `pnpm dev` 即可；不同终端按服务拆开时，不能重复启动相同服务。

停止自己的开发实例：TUI 在服务列表按 q，或按 Ctrl-C；纯文本模式按 Ctrl-C。仅停止该实例启动的进程树，Ubuntu 依赖继续运行。不要重复初始化 `.env`、密钥或删除数据卷。安装了新依赖的其他工作区先执行 pnpm install。

## 试用注册和登录

没有预置的固定测试账号或默认管理员；此前自行注册的开发账号可以继续登录。自动化测试账号属于临时数据库，测试结束即清理，不能作为日常账号使用。邀请码与邮箱验证码不同，分别按下面步骤获取。

1. 在另一个 PowerShell 终端进入同一目录，运行 `pnpm dev:invite`。命令创建一个默认 24 小时有效、单次使用的开发邀请码，并只向该终端输出一次原文；数据库仅保存哈希。不要把邀请码贴入 issue、日志或提交。
2. 在账号页选择“创建账号”，填写邮箱、昵称、邀请码和至少 15 字符的密码。可使用例如 `reader@moetopia.test` 的测试邮箱。
3. 点击“发送验证码”，在 Mailpit 找到该收件人的注册邮件，将 8 位验证码填回页面。当前邮件只进入开发 SMTP，不会投递至真实外部邮箱。
4. 创建成功后用邮箱和密码登录。刷新应保持登录；可在会话区结束会话，或点击退出登录。

注册默认邀请制。当前开放/邀请模式由后端持久策略控制，管理页面切换尚未完成，不需要为了看页面直接改数据库。GitHub、Google、找回密码、资料编辑和管理 MFA 仍未开放；现有登录不等于完整身份产品或管理台 SSO。

`dev:invite` 仅允许 development 环境，不提供默认管理员或固定账号密码。开发依赖文件和应用加密密钥均不提交。

## 单独查看组件

在另一个终端运行：

```powershell
pnpm storybook
```

打开 `http://localhost:6006`，在 Overview 查看组件与品牌工作台，在 Layouts 查看页框、认证页布局和工作区组合。Storybook 不依赖真实身份账号。自动浏览器测试使用独立的 53000/53001/56006 端口，不占用日常开发或 Storybook 端口。

## 数据变更与故障

专属开发库已按顺序应用迁移 1–7，最近的 6/7 分别新增账号管理及开发者应用数据，保留账号、既有会话和签名密钥；详见相应运行说明。前端布局合并本身没有迁移。其他工作区有新迁移时，确认目标库再执行 `pnpm api:migrate`；旧授权主体显式回填用 `pnpm api:grants:backfill`。API 因缺迁移退出时，迁移后在 TUI 选 api 按 r。不能用生成 Ent 代码代替迁移。

账号服务不可用时先看启动终端，再运行 `pnpm dev:doctor:remote` 检查远程端口。该检查不验证密码或数据库结构；`/readyz` 和集成测试才会进行进一步检查。Ubuntu 重启后如何启动依赖见工程指南，不在 Windows 执行远程 `sudo` 或混用本机 `dev:up`。

反馈时提供页面地址、操作步骤、主题/语言、期望效果和实际表现即可。截图可包含界面，但应遮挡密码、验证码、邀请码和个人资料。前端修改通常即时可见；服务端迁移和启动脚本变化会明确说明是否需要操作。
