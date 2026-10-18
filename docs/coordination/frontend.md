# 前端进度与后端接入反馈

更新时间：2026-10-18。所有者：前端/UI 对话。已读取 [同步规则](README.md)、[后端状态](backend.md) 和 [授权接口说明](../run-authorization.md)。本文件由前端维护，不代表另一对话已收到通知。

## 本轮接入

- account 首页识别 `authorization_request`，在原 URL 中跨登录保留；重复、空值或非 43 位 base64url 句柄显示无效请求，不调用授权接口。
- 按生成 `components` 类型调用授权 GET/POST，不修改生成包。展示真实 client_name/client_id、完整 redirect_uri、scope、audience、到期时间和账号；未知 scope 原文展示，不编造权限含义。
- `login_required`（包括已有账号的强制认证）显示原有密码登录表单；成功后重读详情，POST 使用登录返回的新 CSRF 和详情 user.id。不自动批准；拒绝显式发送 deny。
- 401 进入重新登录；409 重读 context 和详情，提示账号变化并要求再次核对；403/404/422/429/503 各有恢复或重新发起说明。过期隐藏确认；写入不自动重试。成功完整导航至后端返回的 redirect_to，不拼接 code/state、不 fetch 回调。
- 固定 Go 上游 Discovery 代理已加入共享 Nuxt 层，保留 `/oauth/**`。account 页面设置 no-store、no-referrer、CSP frame-ancestors 'none' 和 X-Frame-Options DENY。
- 页面仅组合既有共享 UI，不新增应用内重复控件或角色图片；支持中英文、明暗主题和手机布局。

## 验证状态

全项目类型检查、三应用构建已通过；32 项页面浏览器检查通过（含 18 项新增授权检查），覆盖真实字段展示、允许/拒绝完整导航、强制认证、新 CSRF、账号变化与切换、过期、GET/POST 错误、英文深色窄屏及响应头。桌面与手机截图复核无横向溢出或页面脚本错误。模拟页面检查不等于独立 RP/真实协议互操作。

Discovery 的固定代理配置已通过构建，但实际请求 Go `127.0.0.1:8080` 时连接失败；未启动或停止后端，真实 Discovery 内容一致性尚未验收。admin/developer 当前为静态产物，静态托管不会执行 Nitro 代理；协议入口以 account 服务为准。

共享翻译入口 `packages/locales/src/index.ts` 的新增授权导出在本轮被再次移除，导致后续构建失败；原因未确认。前端已恢复该导出，请保持本轮文件所有权，避免同时改写此入口。恢复后最终 account 构建、全项目类型检查和 32 项页面检查均重新通过；文档检查通过（42 个 Markdown、208 个本地链接）。私人页面响应头仅作用于 `/`、`/sessions`、`/security`，不将身份页禁缓存策略铺到所有静态资源。

## 协作边界与后续

本轮只改 account 应用、共享 portal 配置、locales、页面测试与本状态文件；未改 Go、迁移、生成契约、根依赖或另一侧状态。当前 Git 已存在，main 尚无提交且有大量原有暂存/未暂存文件；保留现状，不自行初始化、提交、清理或重置。早期文档“没有 Git”的描述已过时，后续维护公共文档时请更新。

前端接入完成后，需要后端侧继续安排真实 RP 浏览器互操作，包括 Discovery 经 Nuxt 的 issuer/端点一致性、授权 Cookie/登录轮换、真实 PKCE/nonce/ID Token 与回调、拒绝和重放。本轮不创建默认 OAuth 客户端，不更改共享开发库或结束任何已有开发进程。
