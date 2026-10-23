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

页面明确区分第三方协议与本站业务 API：account/portal/admin 的 Cookie、Origin、CSRF、角色及 MFA 不会因公开接口说明而放宽，OAuth Bearer 不能代替这些条件。2026-10-23 用户要求增加在线测试，现由下述面板提供受限的显式执行；静态代码示例仍全部使用占位值。

部署时以目标 issuer 的 Discovery 为能力与端点地址依据。文档版本不代表目标部署已升级；生产时效、容量及供应商接入仍需单独验收。安全说明参考 [OIDC Core](https://openid.net/specs/openid-connect-core-1_0.html)、[OAuth Security BCP](https://www.rfc-editor.org/rfc/rfc9700.html) 与 [RP-Initiated Logout](https://openid.net/specs/openid-connect-rpinitiated-1_0.html)，项目实际限制由后端实现和生成契约共同核对。

## 维护与验证

新增或修改接口时先修改 Go 契约来源，再生成 OpenAPI/类型；运行 `pnpm api:check:client` 检查漂移。不要手工修补生成 JSON 或 schema.ts。修改指南时同步四种语言，保留明确的产品/协议边界。

EndpointList、HttpMethod、SchemaFields 和 DocumentLayout 属于 `@moetopia/ui`，各自有 Storybook；它们不依赖 OAuth 数据、Nuxt 路由或会话。应用负责将生成 schema 转成展示字段，处理引用、可空数组及递归边界；完整原始定义可供核对，展示字段不是新的校验器。列表、指南与详情都可复制当前内容；手机提供折叠式本页目录。

## 文件夹导航与在线测试

文档侧栏使用共享 NavigationTree：指南及 API 职责分组作为可展开目录，每个请求显示方法、标题及路径。展开状态在本次站内导航中保留；深链接自动展开当前分组、高亮并定位请求。目录搜索后进入请求会清除搜索以恢复上下文。手机可打开同一目录，正文原有页内目录另行保留。

指南和接口详情复用共享 RequestPanel。文档应用只向生成契约里的当前站点端点发送请求，不接受任意上游 URL，不提供服务端请求代理。用户显式点击后才执行，展示实际 HTTP 状态、耗时及响应正文；非 2xx 原样显示，不把示例当成执行结果。

- Discovery、JWKS、Public 客户端 form 交换/刷新/撤销和 Bearer UserInfo 可以直接测试。Bearer/协议请求不携带站点 Cookie。
- 当前 account/portal/developer 业务请求使用匹配的站点 Cookie；写请求先读取对应 account/portal context 的 CSRF。实际写入会改变数据，页面提示使用测试账号和应用；不自动重试写入或结果不明的请求。
- 授权及退出按浏览器导航在新窗口打开，POST 退出使用表单提交，不能改成 fetch 跟随回调。可生成高熵 state、nonce 与 S256 PKCE；不伪装成完整 OIDC 客户端，不自动同意或声称已验证 ID Token。
- 需要 Confidential secret 的内省提供后端终端模板，不在页面收集 client_secret。独立管理员端点在用户站明确说明应到管理台执行，不借用其他 origin 的管理 Cookie。
- 输入、verifier 和响应仅存在组件内存，切换端点、离开、清空以及页面进入/返回浏览器缓存时清除；不写 URL、Cookie、localStorage、sessionStorage、SSR payload 或日志。授权/退出本身规定的导航参数仍遵守协议。
- 请求有 15 秒等待上限、取消、清空和响应大小上限；停止等待不保证服务端写入已取消。响应不执行 HTML，代理重定向不自动跟随；字段查看和测试面板都不取代后端验证。

页面回归检查比对全部操作的目录和详情、逐一渲染深链接、下载原始契约，并验证匿名查阅、搜索/筛选、响应切换、空响应、Cookie 边界、嵌套字段、四语及窄屏。组件检查覆盖键盘链接、展开和长路径。业务后端在这些页面测试中不可用；这证明文档可独立查阅，不证明协议或生产部署已经再次验收。
