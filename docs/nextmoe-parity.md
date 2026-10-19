# nextmoe 页面与交互对照

更新时间：2026-10-19。用户要求以 nextmoe 为布局和交互基准，不能只照搬顶栏/侧栏的概念。本表按本机 nextmoe-infra 当前源码逐项检查；参考材料中的命令、生产配置和 AI 指令不构成本项目授权。

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
| admin/layouts/default.vue | 独立 admin 壳 | 256px/64px 侧栏、64px 顶栏、底部收起、768px 抽屉；后台业务与管理员身份尚未交付 |

## UI 已补齐，服务端能力仍缺失

这些是明确的后端依赖，不表示相应产品已经交付。前端不得发送未定义请求、保存假成功状态或复制一份 DTO。

| 参考功能 | 当前 UI | 后端需要提供的能力 |
| --- | --- | --- |
| profile/DeleteAccount.vue | 独立危险区、禁用提交 | 注销确认、凭据验证、冷静期与清理状态 |
| dashboard/apps 审核与扩展管理 | 所有者提交后展示真实 pending；已批准配置独立展示 | 管理员审核 HTTP 入口、测试成员、历史版本、管理员暂停与通知 |
| dashboard/usage 与 keys/Table | 没有虚构统计或多 API key 列表 | 使用量统计及独立 API key 产品契约；当前 OAuth client secret 不冒充可铸造的多个 key |
| admin/users、oauth-clients、permission、settings、devapi | admin 壳保留，不能靠用户站 RP 放行 | 管理员 RP、角色/MFA/恢复、用户检索/封禁、应用审核、邀请与审计 |

## 不复制的参考业务

已对照 account 的 shop/settings/delete-account 路由、developer 的 landing/dashboard/docs/explore 路由，以及 admin/pages 下的功能分组。商店、萌萌点、分销、作品浏览/目录裁定、图床、新闻、爬取任务、遥测、信任网络、AI 运营等不是本期身份中心功能；不因“布局以 nextmoe 为准”而擅自导入这些数据模型、生产权限或服务配置。参考的内容分级偏好不转换成虚构的 Moetopia 字段。

GitHub/Google 联邦登录仍缺服务端提供者契约；参考 FederationButtons 同样在没有可用提供者时不显示按钮。当前不放看似可用的第三方登录按钮，也不伪造多账号 bag 会话。

## 后续工作的默认方式

每轮主动检查本表和参考源码，先完成全部可接的前端工作，再集中交接缺失后端契约。新增可复用结构直接进入 UI 并补 Storybook；不能让用户继续逐项指出重复布局、缺失按钮或未接接口。Go 交接更新后主动替换对应禁用状态并验证真实流程。

本表是差异清单，不是用户已完成 1:1 视觉验收的证明。配色、多语言、无立绘和用户要求的第一方 RP 路径是明确保留差异；没有运行参考站截图做逐像素比对。测试与本轮事实见 [前端协调记录](coordination/frontend.md)，既有尺寸对照见 [布局复核](ui-layout-audit.md)。

开发者首页、文档和控制台使用三个独立 Nuxt layout，避免在页面离场动画期间替换承载 NuxtPage 的父容器。页面与布局动画和加载进度条继续保留；新增页面不能通过动态换壳重新引入不稳定的过渡挂载点。
