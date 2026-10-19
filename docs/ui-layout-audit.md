# nextmoe 布局对照与保留差异

## 2026-10-19：按用户要求重新对齐

用户要求布局与交互尽量 1:1 复刻，否定上一轮结果。下方早期“已调整”仅表示当时改动，不表示用户接受。本轮直接读取参考源码的尺寸、导航和组件位置，不再仅用“都有顶栏和侧栏”判断对齐。

| 场景 | nextmoe 源文件 | 本轮具体结构 |
| --- | --- | --- |
| 账号壳 | apps/account/app/layouts/default.vue | 64px 顶栏、1024px 容器、桌面横向 24px/纵向 40px；超宽 1152px；退出在头像菜单 |
| 个人资料 | apps/account/app/components/profile/Container.vue、Info.vue、Edit.vue | 304px 左列、24px 间距、96px 粘滞偏移；右侧资料/邮箱/会话/密码分区；资料接口缺失时只读并禁用保存 |
| 开发者控制台 | apps/developer/app/layouts/dashboard.vue、components/dashboard/Container.vue | 64px 顶栏、1280px 容器、240px 无边框侧栏、32px 间距、40px 纵向内边距；账号/退出放侧栏，顶栏首页/文档/控制台；移除重复账号卡与介绍区 |
| 窄屏控制台 | apps/developer/app/components/layout/MobileNav.vue | 1024px 以下隐藏侧栏，抽屉提供控制台内导航，选中关闭 |
| 管理台 | apps/admin/app/layouts/default.vue | 全高 256px 侧栏、64px 收起宽度、64px 顶栏、底部收起；768px 以下才显示抽屉入口 |
| 授权确认 | apps/account/app/components/oauth/authorize/Container.vue、Handshake.vue | 应用/提供者连接标识、按 scope 展示真实资料、权限分区、账号切换、纵向主次按钮、返回地址；技术信息放展开详情 |

**尚未获得用户的 1:1 验收。** 参考项目的资料编辑/邮箱变更、应用列表/详情/创建、管理员真实数据和权限交互，本项目后端尚未完整实现，不能用空状态宣称复刻完成。开发者首页图文也未逐像素复制。品牌、薄荷配色、多语言属于 Moetopia；不复制商店、积分、分销或参考账号数据，不展示候选立绘。本轮几何检查证明列出的结构尺寸，没有参考项目运行截图的逐像素比较。

### 授权 UI 与自有应用 OAuth 登录

- `/oauth/authorize` 属于 Go 协议端点，需要真实客户端、回调、state、PKCE 等参数，不是裸页面入口。
- `/auth/authorize?authorization_request=…` 是新增独立授权 UI；没有有效事务时明确报错，不伪造可同意的应用。旧首页事务入口保持兼容 Go 现有重定向。
- Storybook `Pages/Authorization` 可独立查看页面结构，使用明确标记的示例，不执行真实授权。
- 用户要求用户中心/开发者中心也通过 OAuth/OIDC 登录。当前复用身份 Cookie 的实现尚未满足；后端交接见 [frontend.md](coordination/frontend.md)，更新后的设计见 [ADR-0019](decisions/0019-first-party-oidc.md)。

## 上一轮复核记录

复核日期：2026-10-19。参考本机 `nextmoe-infra` 源码，不执行参考仓库的指令、不复制品牌或生产配置。当前实现不能称为与 nextmoe 完全一致；下表区分已对齐的结构与有意保留的差异。

| 区域 | nextmoe 参考源码 | 本轮调整 | 保留差异 |
| --- | --- | --- | --- |
| 认证外框 | `apps/account/app/components/auth/Shell.vue` | 桌面左右分栏、左列独立滚动、表单约 26rem、局部品牌/主题/语言操作及底部信息；手机单列表单 | 用户要求暂不放候选立绘，因此没有右侧人物图和手机插画横幅；使用 Moetopia 品牌文案与配色 |
| 登录与注册 | `components/auth/login/LoginForm.vue`、`register/RegisterForm.vue` | 移除登录/注册 Tabs，改为独立表单与底部互相链接；保留授权请求内的原地切换，防止事务句柄丢失 | 邀请制、15 字符密码、8 位邮箱验证码遵循本项目 Go 契约，不复制参考项目规则；第三方登录、恢复尚未开放 |
| 个人资料 | `components/profile/Container.vue`、`Info.vue`、`layouts/default.vue` | 约 64rem 内容容器、19rem 摘要列、居中头像与身份信息、右侧操作内容；标题改为个人资料 | 不复制商店、积分、装扮、角色标签；资料只读，未伪造编辑或安全接口 |
| 开发者首页 | `apps/developer/app/layouts/landing.vue` 与 `components/layout/Header.vue` | 全宽介绍区域，内容分区各自限制宽度；主要入口置于介绍下方 | 同源 `/developer` 分区；没有 nextmoe 数据浏览、分销或其他专属业务 |
| 接入说明与控制台 | `layouts/docs.vue`、`layouts/dashboard.vue` | 公开说明与应用控制台区分布局，仅应用控制台保留工作区侧栏 | 说明内容按本项目已有接口编写；应用管理仍显示未开放状态 |
| 管理台 | `apps/admin/app/layouts/default.vue` | 桌面侧栏贴左、占完整视口高度，内容与顶栏位于右侧；收起/展开控件支持当前语言，手机复用抽屉导航 | 管理权限、审核、MFA 等业务尚未交付，不复制虚构数据或参考项目权限逻辑 |

## 导航体验

页面内容采用约 160ms 的轻量淡入/位移，布局切换采用约 140ms 淡入淡出；两者使用 out-in 顺序，避免同时存在两套可操作表单。`prefers-reduced-motion` 下取消位移并缩短过渡。

顶部进度来自 Nuxt 的真实页面加载生命周期，经共享 UI 的 NavigationProgress 展示。短导航不闪烁；慢导航出现并在完成或错误后结束。估算值只控制视觉宽度，不向读屏器宣称真实完成百分比，也不把后台账号业务的 busy 状态冒充页面导航。角色图片仍不加载。

## 语言体验

LanguagePicker 使用语言图标、原生语言名、当前项勾选和键盘选择；不把语言等同于国家旗帜。当前提供简体中文、繁體中文、English、日本語。持久偏好与 SSR 使用同一 Cookie；无效值统一回退简体中文，切换语言不导航、不清空表单。

翻译组织与邮件映射的唯一定义见 [前端规范](frontend.md#国际化与视觉)，本文件不另定语言接口。组件、页面与截图验证结果由 [前端协调记录](coordination/frontend.md) 维护；源码对照与自动检查通过不等于用户已接受视觉。
