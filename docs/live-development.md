# 边开发边查看效果

更新时间：2026-10-18。当前 Windows 运行 Go/Nuxt，依赖由 Ubuntu `192.168.101.41` 提供。无需在 Windows 安装 Docker。本文面向当前已有 `.env`、依赖容器、迁移和签名密钥的工作区；首次设置见 [工程指南](run-foundation.md)。

## 日常启动

在自己的 PowerShell 终端运行，并保留该终端：

```powershell
cd D:\misaka10843\Documents\Github\moetopia-AIO
pnpm dev
```

交互式终端中进入 mprocs：左侧分别显示 api、account、admin、developer 的进程状态，右侧只显示所选服务的输出。某个服务退出后保留日志和退出码，其他服务继续运行；修复原因后可以单独重启。推荐在 Windows Terminal / PowerShell 使用。

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
| `http://developer.localhost:3002` | 开发者工作区、接入准备与未开放说明 |
| `http://192.168.101.41:58025` | 开发邮件收件箱 |

账号页请使用表中的 `account.localhost`，不要改用 Nuxt 控制台显示的 `127.0.0.1:3000`：当前 Go 严格检查 `.env` 中的账号 origin，地址不同会拒绝写请求。三个应用使用不同 host，Cookie 不跨应用共享。

Vue、样式和翻译保存后由 Nuxt 自动更新。Go `.go`、迁移 `.sql`、go.mod/go.sum 变化触发 API 重新构建和重启；编译失败时保留上一版 API，错误显示在终端，修正后再保存即可。后端重启期间的请求可能失败，等待就绪后手动重试。开发启动器不会自动应用迁移。

若旧开发实例尚未识别 Nuxt 配置变化，在原启动终端 Ctrl-C 后重新运行一次 `pnpm dev`。仅查看某一个前端也可分别运行 `pnpm --filter @moetopia/account dev`、`pnpm --filter @moetopia/admin dev` 或 `pnpm --filter @moetopia/developer dev`；这不会启动 Go，账号服务不可用时页面会显示连接失败，注册与登录需要完整开发实例。

账号工作区通过侧栏或手机菜单进入 `/` 资料、`/sessions` 登录会话和 `/security` 安全设置。会话页复用现有查询与撤销接口；安全页展示现有会话入口及尚未开放的密码修改、第三方绑定、授权管理说明，不代表这些后端能力已完成。正面立绘暂不用于页面，保留角色配色。前后端分工和新对话接手说明见 [交接文档](workstreams.md)。

顶栏跨应用地址来自 `NUXT_PUBLIC_ACCOUNT_URL`、`NUXT_PUBLIC_ADMIN_URL`、`NUXT_PUBLIC_DEVELOPER_URL`。开发默认指向本表地址，示例见 [.env.example](../.env.example)；生产未配置时对应跨应用链接禁用，不猜测正式域名。admin/developer 是静态构建，应在构建环境配置这些公共地址。它们不是凭据，不用于替代后端 Origin、回调地址或权限检查。

脚本只检查选中服务的端口：前端 3000/3001/3002、Storybook 6006，API 从 MOETOPIA_LISTEN 读取（默认 8080）；每次启动/重启再次检查。不会抢占、杀死已有进程或让 Nuxt 悄悄换端口。提示占用时先检查是否已在另一终端运行了本项目；只在归属明确的原终端停止。一个工作区默认保持一份 `pnpm dev` 即可；不同终端按服务拆开时，不能重复启动相同服务。

停止自己的开发实例：TUI 在服务列表按 q，或按 Ctrl-C；纯文本模式按 Ctrl-C。仅停止该实例启动的进程树，Ubuntu 依赖继续运行。不要重复初始化 `.env`、密钥或删除数据卷。安装了新依赖的其他工作区先执行 pnpm install。

## 试用注册和登录

1. 在另一个 PowerShell 终端进入同一目录，运行 `pnpm dev:invite`。命令创建一个默认 24 小时有效、单次使用的开发邀请码，并只向该终端输出一次原文；数据库仅保存哈希。不要把邀请码贴入 issue、日志或提交。
2. 在账号页选择“创建账号”，填写邮箱、昵称、邀请码和至少 15 字符的密码。可使用例如 `reader@moetopia.test` 的测试邮箱。
3. 点击“发送验证码”，在 Mailpit 找到该收件人的注册邮件，将 8 位验证码填回页面。当前邮件只进入开发 SMTP，不会投递至真实外部邮箱。
4. 创建成功后用邮箱和密码登录。刷新应保持登录；可在会话区结束会话，或点击退出登录。

注册默认邀请制。当前开放/邀请模式由后端持久策略控制，管理页面切换尚未完成，不需要为了看页面直接改数据库。GitHub、Google、找回密码、资料编辑和管理 MFA 仍未开放；现有登录不等于完整 OIDC 登录或三应用 SSO。

`dev:invite` 仅允许 development 环境，不提供默认管理员或固定账号密码。开发依赖文件和应用加密密钥均不提交。

## 单独查看组件

在另一个终端运行：

```powershell
pnpm storybook
```

打开 `http://localhost:6006`，在 Overview 查看组件与品牌工作台，在 Layouts 查看页框、认证页布局和工作区组合。Storybook 不依赖真实身份账号。自动浏览器测试使用独立的 53000/53001/53002/56006 端口，不占用日常开发或 Storybook 端口。

## 数据变更与故障

本轮已应用迁移 2，新增 identity、sessions、notifications 三个 schema，不修改已有 authorization 表。后续有新迁移时，按该变更说明执行 `pnpm api:migrate`，必要时重新启动开发实例；不能用生成 Ent 代码代替迁移。

账号服务不可用时先看启动终端，再运行 `pnpm dev:doctor:remote` 检查远程端口。该检查不验证密码或数据库结构；`/readyz` 和集成测试才会进行进一步检查。Ubuntu 重启后如何启动依赖见工程指南，不在 Windows 执行远程 `sudo` 或混用本机 `dev:up`。

反馈时提供页面地址、操作步骤、主题/语言、期望效果和实际表现即可。截图可包含界面，但应遮挡密码、验证码、邀请码和个人资料。前端修改通常即时可见；服务端迁移和启动脚本变化会明确说明是否需要操作。
