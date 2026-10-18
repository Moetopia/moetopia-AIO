# 前端语言资源

当前语言：简体中文 `zh-CN`、繁體中文 `zh-TW`、English `en`、日本語 `ja`。

每个语言独立放在 `src/lang/<locale>.ts`，内部使用 common、account、portal、authorization 分组。`src/types.ts` 定义消息形状，`src/index.ts` 只注册语言、导出已有的分组访问入口与适配函数，不维护翻译正文。UI 通过包入口消费，不能直接依赖其他语言文件或在页面复制字典。

新增语言时填写完整语言文件、扩展 Locale 与 supportedLocales，并通过类型检查、菜单键盘选择、SSR/刷新、长文本和窄屏验证。新增消息须同时更新所有语言；不要展开英文对象掩盖缺失翻译，不把国家旗帜用作语言标识。默认语言与无效 Cookie 的回退由 resolveLocale 统一处理。

邮件与前端语言独立：Go 的验证码接口目前只接受 zh-CN/en。emailLocale 将繁中映射为 zh-CN、日文映射为 en；提供完整繁中/日文邮件需后端另外实现，不能将前端语言码直接扩大为 API 合同。界面切换不清空表单，也不保存密码或验证码到 Cookie。

验证使用根目录的 `pnpm typecheck`、页面和组件测试；完整约束见 [前端规范](../../docs/frontend.md#国际化与视觉)。
