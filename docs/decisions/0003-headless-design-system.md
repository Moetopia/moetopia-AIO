# ADR-0003：Headless 设计系统的候选组合

- 状态：建议；Headless + Storybook 已确认。
- 记录日期：2026-10-17。

## 建议决定与依据

采用 Reka UI 提供交互原语，Tailwind CSS 表达样式，建立 `@moetopia/ui`；组件故事使用 Storybook 的 Vue + Vite 框架。Reka 提供无样式 Vue 原语，Storybook 提供 Vue/Vite 集成，分别见 [Reka 文档](https://www.reka-ui.com/docs/overview/introduction) 和 [Storybook 文档](https://storybook.js.org/docs/get-started/frameworks/vue3-vite)。资料核对日期：2026-10-17。

## 理由与代价

登录、表单、弹窗和后台能共享品牌与交互。原语负责基础交互，不代表项目自动满足可访问性要求；标签、错误、对比度、焦点与组合行为仍需项目测试。

自有封装要维护 API、主题和故事。不能将每个原语机械包一遍，也不能让业务页面各自复制控件。

## 替代方案

成品库主题化可降低组件维护，但与已选择的 Headless 路线不同，需要重新讨论。Headless UI for Vue 等其他原语库可作为验证失败后的替代，不能在同类控件中混用多套交互底座。

## 待验证与重新评估

验证 Nuxt/Vue/Reka/Storybook 的稳定版本组合、弹层传送、SSR、水合、键盘及主题切换。不直接复制参考仓库版本号。若缺少关键原语、上游维护或兼容不满足，再更新 ADR。

长期与性能复核建议继续采用该候选组合，按需导入和封装，检查共享包产物、Tailwind 源码扫描及弹层/长列表成本。Storybook 只用于开发与 CI，不是生产运行依赖；详见 [全栈复核](../stack-review.md)。

品牌和组件规范见 [design-system.md](../design-system.md)。
