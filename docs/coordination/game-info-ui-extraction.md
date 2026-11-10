# Game Info 与共享 UI 提取交接

2026-11-10 用户授权创建独立 `moetopia-game-info` 和共享 `moetopia-ui`。本仓继续拥有身份及账号业务，没有新增游戏数据库表、游戏接口或共享身份 Cookie。

共享 UI 从 AIO 提交 c9b5f7f7eb92149fc382ddd121c1ed52149627e0 提取到独立仓；源码、样式、品牌素材和故事仅在那里维护。AIO 的 packages/ui 保留工作台，从固定归档加载。当前包和来源提交见 [记录](../../vendor/ui-source.json)。远程仓尚不可访问，未推送；因此使用附 SHA-256 的归档，后续远程建立后改为固定 Git 提交依赖。

已验证：account/admin 生产构建、全前端类型检查、迁移后 Storybook 构建、41 项组件浏览器回归和文档链接。原有 packages/ui/src 已在源码提取后移入本仓忽略的 .cache/ui-extraction-backup，保留可逆副本；消费者不依赖该目录。

规范入口与前端/设计/架构说明已更新。维护插画时必须显式提供 MOETOPIA_UI_SOURCE 指向共享源码仓，避免脚本重建本仓副本。素材原图及其他会话改动保持原样。

后续补齐：新增仅 integration 构建标签启用的 `cmd/gameinfofixture`，用于 Game Info 的真实 AIO 授权服务器协议联调。夹具严格要求 loopback 回调、显式测试标记、`mgi_test_aio_<随机值>` 空数据库，并复用真实授权/签名/持久化实现；合成用户和同意页只用于隔离验证，不编入正常 API。调用入口为 Game Info 的 `pnpm test:aio <AIO checkout>`。

共享 UI 0.1.4 同步修复 Dialog/Drawer 的描述 ID 绑定以及无描述导航抽屉的 ARIA 声明，版本及摘要由来源记录维护。提供描述时不向 Reka 显式传 undefined，以保留默认描述关联；无描述时传空值。读屏描述、焦点和弹窗警告检查以及 41 项组件回归通过。

正式 confidential RP 注册、真实账号/MFA 与完整授权页面浏览器联调尚未完成，不把隔离协议测试当作生产验收。不得自动变更 AIO 生产应用或主机资源。
