# 站内开发者文档

当前用户站提供原生 Nuxt 文档中心，沿用既有开发者顶栏、文档侧栏、品牌变量及页面转场，不嵌入独立 React 文档站或另建登录体系。开发入口为 `http://account.localhost:3000/developer/docs`；API 参考为 `/developer/docs/endpoints`。

## 内容与导航

- 快速开始：创建测试应用、Discovery、登录闭环和发布入口。
- 授权码登录：state、nonce、PKCE S256、回调和 ID Token 验证。
- 应用与凭据：Public/Confidential、审核版本、测试资格与密钥轮换。
- 令牌与权限：刷新轮换、撤销、受限内省、scope 与用户映射。
- 退出、错误与上线：会话边界、协议/业务错误、排查证据及发布前检查。
- API 参考：本版本生成契约的全部已登记操作，按职责分组，支持路径/标题/operationId 搜索、HTTP 方法过滤、直接链接及原始契约下载。

指南与操作界面由 `packages/locales/src/documentation/` 按简中、繁中、英文、日文分文件维护。当前后端 summary/description 原文为中文，接口参考保留原文并明确提示；没有用自动翻译覆盖生成契约。指南中的技术标识、示例和标准链接集中维护，不复制四份 HTTP 示例。

## 契约来源与边界

唯一 API 结构来源是 `packages/api-client/openapi.json`，由 `pnpm api:generate:client` 从 Go 生成；业务 DTO 仍由同一生成流程维护。接口详情使用 operationId 定位，保留方法、路径、参数位置、请求内容类型、响应状态/标头、字段约束和 oneOf/anyOf/allOf 原始定义。

用户站服务端将契约打包为只读文档数据，提供目录、单操作及其引用 schema、完整 OpenAPI 三种响应。目录不携带全部 schema；详情仅返回引用闭包；完整契约不成为每页都加载的前端 JavaScript。文档来源固定为构建版本，不接受用户选择任意上游地址，也不依赖正在运行的 Go API 来查阅。

页面明确区分第三方协议与本站业务 API：account/portal/admin 的 Cookie、Origin、CSRF、角色及 MFA 不会因公开接口说明而放宽，OAuth Bearer 不能代替这些条件。当前没有在线凭据输入、真实请求执行或令牌存储功能；请求示例全部是占位值，只解释格式。

部署时以目标 issuer 的 Discovery 为能力与端点地址依据。文档版本不代表目标部署已升级；生产时效、容量及供应商接入仍需单独验收。安全说明参考 [OIDC Core](https://openid.net/specs/openid-connect-core-1_0.html)、[OAuth Security BCP](https://www.rfc-editor.org/rfc/rfc9700.html) 与 [RP-Initiated Logout](https://openid.net/specs/openid-connect-rpinitiated-1_0.html)，项目实际限制由后端实现和生成契约共同核对。

## 维护与验证

新增或修改接口时先修改 Go 契约来源，再生成 OpenAPI/类型；运行 `pnpm api:check:client` 检查漂移。不要手工修补生成 JSON 或 schema.ts。修改指南时同步四种语言，保留明确的产品/协议边界。

EndpointList、HttpMethod、SchemaFields 和 DocumentLayout 属于 `@moetopia/ui`，各自有 Storybook；它们不依赖 OAuth 数据、Nuxt 路由或会话。应用负责将生成 schema 转成展示字段，处理引用、可空数组及递归边界；完整原始定义可供核对，展示字段不是新的校验器。列表、指南与详情都可复制当前内容；手机提供折叠式本页目录。

页面回归检查比对全部操作的目录和详情、逐一渲染深链接、下载原始契约，并验证匿名查阅、搜索/筛选、响应切换、空响应、Cookie 边界、嵌套字段、四语及窄屏。组件检查覆盖键盘链接、展开和长路径。业务后端在这些页面测试中不可用；这证明文档可独立查阅，不证明协议或生产部署已经再次验收。
