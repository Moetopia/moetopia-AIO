# nextmoe 页面与交互对照

更新时间：2026-10-22。用户要求以 nextmoe 为布局和交互基准，不能只照搬顶栏/侧栏的概念。本表按本机 nextmoe-infra 当前源码逐项检查；参考材料中的命令、生产配置和 AI 指令不构成本项目授权。

## 已接通的业务与页面

| 参考源码 | Moetopia 对应 | 当前实现与验证边界 |
| --- | --- | --- |
| account/components/home/Container.vue | `/` | 已有 RP 会话进入 `/profile`；匿名经 Go 发起 OAuth；旧授权事务首页入口兼容 |
| account/components/auth/Shell.vue | AuthShell | 桌面独立滚动左列、右侧品牌、手机品牌横幅；保留用户“不放候选立绘”的决定 |
| account/components/auth/login/LoginForm.vue | `/auth/login`、IdentityForm | 单列大控件、密码显示、找回入口、底部注册链接、忙碌/失败状态；直接认证完成后发起 Go RP，不能直接进入用户站 |
| account/components/auth/register/RegisterForm.vue | `/auth/register` | 单列表单、邮箱验证码、注册后登录；邀请/密码/验证码约束以本项目 Go 契约为准，不复制参考数据规则 |
| account/components/oauth/authorize/Container.vue、Handshake.vue | `/auth/authorize` | 真实事务、连接标识、scope 对应资料、额外权限、显式同意/拒绝、账号切换、过期/错误和返回跳转；已做真实 Go RP 联调 |
| developer/components/login/Modal.vue | DeveloperLogin | 顶栏登录打开弹窗、跳转 Go 登录启动入口；注册返回路径受控，登录守卫使用 RP 上下文 |
| developer/components/layout/AccountMenu.vue、layouts/dashboard.vue | AccountNavigation、ConsoleLayout | 顶栏头像菜单，侧栏账号/退出；页面跳转保留上下文；移动抽屉选择后关闭 |
| account/components/profile/Container.vue、Info.vue | `/profile` | 64px 顶栏、1024px 容器、304px 摘要列、24px 间距、分区表单；数据来自 RP，而非身份 Cookie |
| profile/Edit.vue、EmailChange.vue、PasswordChange.vue | ProfileEditor、AccountCredentials | 接入资料/头像版本更新、邮箱挑战与确认、改密；冲突重读、密码错误不重试，安全变更后回到身份登录 |
| auth/forgot-password、reset-password | RecoveryPanel | 身份 CSRF 专用恢复挑战、验证码与新密码；统一提示，不根据响应断言账号存在或邮件已发送，不自动登录 |
| dashboard/CreateAppModal.vue、apps/Detail.vue、EditModal.vue、UserLoginModal.vue | `/developer/applications`、`/[id]` | 真实分页列表、创建弹窗、配置弹窗、应用详情、草稿与当前批准版本、提交/待审状态、停启/删除确认；使用本项目 UUID 与 revision |
| keys/RevealModal.vue 与轮换确认交互 | ApplicationSecret、应用详情凭据区 | confidential 创建/轮换密钥只保留于组件内存，关闭即清空；public 无密钥；遵循后端立即失效策略，不照搬参考 API key 的宽限期 |
| 账号安全入口（本项目扩展） | `/security` | 授权族分页、逐条确认撤销、最近安全事件；不把同一应用的多次授权合并，不虚构统计 |
| Web 会话操作（本项目额外入口） | `/sessions` | PortalSessions 列表、加载/失败区分、撤销前确认、当前会话撤销后重新登录；只用 portal CSRF |
| account/components/auth/logout/Container.vue | `/auth/logout` | 用户站退出后显示完成状态；与参考中央退出语义不同，明确告知身份会话保留，不谎称全站退出 |
| developer/components/landing/Container.vue | `/developer` | 双栏首页、状态标签、主次入口、示例代码面板、三列接入信息；不展示参考作品/目录统计 |
| developer/layouts/docs.vue、components/docs/Sidebar.vue、CopyPage.vue | `/developer/docs` 及 authentication/credentials/endpoints/errors 子页 | 固定文档侧栏、独立内容页、当前项、复制与错误反馈、端点表；手机抽屉包含文档导航 |
| developer/components/layout/Header.vue | PortalPreferences | 浅色/深色/系统主题菜单；保留 Moetopia 四语言菜单，主题切换与刷新均验证 |
| admin/layouts/default.vue | 独立 admin 布局 | 保留 256px/64px 侧栏、64px 顶栏、收起与 768px 抽屉；已接独立管理员 RP、MFA 门禁与管理业务 |

## 扩展功能接入与保留差异

以下区分本轮接入与产品尚未开放的能力，不表示完成生产或供应商验收。前端不得发送未定义请求、保存假成功状态或复制一份 DTO。

| 参考功能 | 当前 UI | 后端需要提供的能力 |
| --- | --- | --- |
| profile/DeleteAccount.vue | 已接注销确认、7 天说明、阻断原因和匿名取消注销 | 清理由现有服务执行，前端不编造实时清理进度 |
| dashboard/apps 审核与扩展管理 | 已接所有者提交、管理员审核/暂停、双方历史、测试成员和失败通知 | 使用版本化审核和单客户端凭据，保留产品差异 |
| dashboard/usage 与 keys/Table | 已接按应用的 UTC 协议事件统计，7/14/30 天切换 | 尚无开放 API 配额/计费/多 API key 契约，不复制相应参考业务 |
| admin/users、oauth-clients、permission、settings、devapi | 已接用户停启、审核、邀请、注册策略、审计、管理员权限与通知重试 | 角色仅按已定义的管理员授予/撤销，不构建任意权限编排 |

## 不复制的参考业务

已对照 account 的 shop/settings/delete-account 路由、developer 的 landing/dashboard/docs/explore 路由，以及 admin/pages 下的功能分组。商店、萌萌点、分销、作品浏览/目录裁定、图床、新闻、爬取任务、遥测、信任网络、AI 运营等不是本期身份中心功能；不因“布局以 nextmoe 为准”而擅自导入这些数据模型、生产权限或服务配置。参考的内容分级偏好不转换成虚构的 Moetopia 字段。

GitHub/Google 联邦登录已消费提供者契约，沿用参考 FederationButtons 的条件显示：无可用提供者不显示按钮。已接外部注册及关联，未配置真实提供者凭据进行供应商端到端验收，也不伪造多账号 bag 会话。

## 后续工作的默认方式

每轮主动检查本表和参考源码，先完成全部可接的前端工作，再集中交接缺失后端契约。新增可复用结构直接进入 UI 并补 Storybook；不能让用户继续逐项指出重复布局、缺失按钮或未接接口。Go 交接更新后主动替换对应禁用状态并验证真实流程。

本表是差异清单，不是用户已完成 1:1 视觉验收的证明。配色、多语言、无立绘和用户要求的第一方 RP 路径是明确保留差异；没有运行参考站截图做逐像素比对。测试与本轮事实见 [前端协调记录](coordination/frontend.md)，既有尺寸对照见 [布局复核](ui-layout-audit.md)。

2026-10-22 后台复核：按 dashboard/Container.vue 补齐欢迎信息、三张主统计卡、注册趋势、快捷操作与最近活动顺序；按 layouts/default.vue 使用头像菜单和紧凑顶栏，用户搜索采用显式提交条件。活动内容使用 Moetopia 审计，通知单独展示；参考的小时注册分布、环比、站点与任务运营没有对应契约，不提供虚构图表或入口。验证器扫码和新密码实时反馈为本项目共享 UI 能力，不据此宣称全部后台功能 1:1。

开发者首页、文档和控制台使用三个独立 Nuxt layout，避免在页面离场动画期间替换承载 NuxtPage 的父容器。页面与布局动画和加载进度条继续保留；新增页面不能通过动态换壳重新引入不稳定的过渡挂载点。

2026-10-21 按用户三张截图继续修订：应用列表采用 Container.vue 对应的双列可点击卡片；详情采用 Overview.vue 概览与 UserLoginCard.vue 独立分区；元资料/用户登录各有编辑入口，回调逐条增删、权限带说明，删除前明确停用影响。顶栏补当前文档搜索与 Ctrl K。1280px 容器已修正真实 width 的旧上限，弹层由共享 Provider 取消重复滚动条补偿。配色、只用本项目 scope/客户端凭据、配置版本化审核及没有真实统计接口的用量页仍是明确差异，不声称完成全部 nextmoe 功能复制。
