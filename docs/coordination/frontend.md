# 前端进度与后端接入反馈

更新时间：2026-10-18。所有者：前端/UI 对话。开始工作已读 [同步规则](README.md) 与 [后端状态](backend.md)。本文件记录共享工作区事实，不代表另一对话已收到自动通知。

## 本轮已完成

用户已授权合并 account/developer，并要求尽量完成整轮功能、验证后提交 Git。新边界见 [ADR-0017](../decisions/0017-user-site-layouts.md)：

- account 用户站提供 `/auth/login`、`/auth/register`、`/profile`、`/sessions`、`/security` 与 `/developer`、`/developer/applications`、`/developer/docs`；`/` 兼容原入口和 authorization_request。admin 独立。
- 删除独立 developer 应用源码/workspace/build/dev 入口，默认为 api＋account＋admin，不再启动 3002。跨应用配置只需 ACCOUNT_URL 与 ADMIN_URL。旧本地开发者地址变量不再读取，无需修改秘密或数据库。
- 认证采用独立左右页壳、桌面表单列滚动及手机单列；账号采用轻量顶栏、个人摘要与操作内容双栏。开发者首页和控制台分开布局；不再让所有页面套同一种后台壳。保留角色配色、不加载候选立绘。
- AuthShell、ProfileLayout 在共享 UI 定义，含 Storybook 与键盘/响应式检查；公共组件共 57 个。基础控件继续参考 KunUI，页面组织参考 nextmoe。
- 登录先完成目标路由切换再展示已登录页面，避免紧随其后的导航被覆盖；SSR 标签与偏好按钮初始化后启用。开发者续接只接受固定 `continue=developer-applications` 标识，忽略任意 URL。
- 构建与启动配置、锁文件、规范和运行文档同步。dist 产物退出 Git 跟踪并加入忽略，本地文件保留；不提交机器生成的发布文件。

## 协议与后端兼容

沿用 [生成契约](../../packages/api-client/src/schema.ts)，未改 DTO、Go、迁移、Cookie、issuer 或 account origin。授权页继续显示真实应用、回调、scope/audience、账号与到期时间；强制登录后重读详情/CSRF，显式 approve/deny，完整导航至后端 redirect_to。不拼接 code/state、不自动批准或重试写入，处理 401/403/404/409/422/429/503。

保留后端对话对 OAuth 代理的 manual redirect 修复，让浏览器收到原始跳转与事务 Cookie；固定 Discovery 上游继续有效。私人页面保持 no-store、no-referrer 和禁止嵌入，不把身份页禁缓存策略应用到所有静态资源。管理员访问不能因同源用户站或隐藏菜单而放行。

开始时 HEAD 为 9cb17a6。后端已有独立 RP 互操作、代理选项和文档等未提交改动；保留且不纳入前端提交。公共文件出现双方改动时，只暂存本轮所属部分，不覆盖其他差异，不操作共享开发进程。

## 验证

- 两应用构建、全项目类型检查、Storybook 构建通过。
- 36 项页面检查、26 项组件检查通过；分别以 4 个 worker 顺序执行。先前多浏览器与构建并行运行出现加载超时，降低并发后完整通过，没有放宽断言或自动重试失败。
- 3 项开发启动器检查通过，默认服务数量与旧 developer 命令拒绝行为已覆盖。
- 使用后端现有 `pnpm test:oidc`，1 项真实 Chromium 互操作通过：独立 RP、Go、随机测试库、Mailpit、邀请注册、Cookie 旋转、同意/拒绝、prompt=login、JWKS/UserInfo 与异常拒绝；运行器清理自己的测试资源。未改开发库或重启已有服务。
- 已查看桌面登录/注册、账号与开发者页面，以及手机明暗页面；9 张截图无页面脚本错误或横向溢出。截图仅为合成账号，不提交预览产物。

本轮复用后端测试设施，未修改或接管其脚本。类型与构建不代表完整身份产品验收；仍没有应用创建/审核、账号资料修改、密码恢复/联邦绑定、管理员 MFA、生产发布或全面性能验收。后续接口准备好后，继续通过本目录同步前端接入。
