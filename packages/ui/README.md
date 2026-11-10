# UI 工作台

## 当前组件

完整组件契约与统一参数已随源码迁至共享 UI 仓的 `docs/component-reference.md`，也包含在固定来源包中；Storybook 从包内故事生成。本工作台不另维护组件清单。

共享组件源码已迁至独立 moetopia-ui 仓。本目录只保留 Storybook 和类型检查工作台，从固定来源包加载故事，不维护组件副本。

当前来源提交和包 SHA-256 见 [共享包记录](../../vendor/ui-source.json)。使用根目录 pnpm storybook、pnpm storybook:build 预览和构建；修改组件应在共享 UI 仓完成 typecheck、故事和构建，再更新两端的包及来源记录。

消费者通过 @moetopia/ui 公开入口导入，主题和品牌变量由共享包拥有。Tailwind 应用扫描由 packages/portal/app/assets/shared-ui.css 定义。远程仓未发布期间使用固定源码提交生成的归档；没有相邻目录运行依赖。
