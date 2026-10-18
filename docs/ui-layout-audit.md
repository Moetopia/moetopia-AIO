# nextmoe 布局对照与保留差异

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
