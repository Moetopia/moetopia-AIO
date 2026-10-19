# Git 提交规范

状态：提交标题按用户要求采用 `feat (模块): 描述` 形式。2026-10-19 用户已授权完成一批功能并通过相应验证后自行提交，无需逐次确认；只提交本任务路径，不自动推送或配置 hooks。

## 1. 标题格式

```text
<type> (<scope>): <summary>
```

例如：

```text
feat (oauth): add authorization code flow
fix (sessions): reject reused refresh tokens
docs (repo): define coding and commit standards
```

- `type` 与左括号之间必须有一个 ASCII 空格；右括号紧接冒号，冒号后有一个 ASCII 空格。
- scope 必填，表示本次改动的模块或工程领域。
- type 与 scope 使用小写英文；scope 允许数字和中划线，不使用空格或中文括号。
- summary 使用简洁英文，动词原形开头，说明实际变化；不用句末句号。
- 整个提交信息使用英文，包括正文与 footer；代码标识、链接和原样引用的错误输出可保留原样。
- 标题尽量不超过 100 个字符，便于审阅；过长时把背景和迁移详情放正文，不删去关键信息凑长度。

这是项目自定义格式，保留用户指定的 `type (scope):` 空格。未来如引入 commitlint/发布工具，必须验证或配置对应解析规则，不能直接假设默认 Conventional Commits 解析器支持它，也不能擅自改成 `feat(scope):`。

## 2. 提交类型

| type | 使用场景 |
| --- | --- |
| feat | 新增用户或开发者可用能力 |
| fix | 修复行为缺陷 |
| refactor | 调整实现结构，保持外部行为 |
| perf | 有证据的性能改进 |
| docs | 文档、规范、接入示例说明 |
| test | 测试场景或测试基础设施 |
| style | 不改变行为的格式调整，不指产品 UI 功能 |
| build | 构建、打包或依赖变化 |
| ci | CI/CD 工作流调整 |
| chore | 不属于上述类型的仓库维护，不代替所有类型 |
| revert | 撤销既有提交 |

包含测试的功能提交仍使用 feat/fix，不因为添加测试就改成 test。影响可见 UI 的新交互或问题修复使用 feat/fix；只有在专门调整外观且不改变功能时，仍根据改动目的选择明确类型和 scope，不把所有 UI 工作归为 style。

## 3. scope 选择

建议稳定使用以下模块名称，不在同一职责间轮换近义词：

| scope | 职责 |
| --- | --- |
| identity | 账号、凭据、邮箱和联邦绑定 |
| sessions | 会话、设备和重新认证 |
| oauth | OAuth/OIDC 协议、授权及令牌 |
| applications | 应用配置、审核、所有权 |
| access | 角色、scope 和资源权限 |
| account / admin / developer | 对应前端应用 |
| ui | 共享组件、主题与 Storybook |
| api | 跨业务 API 契约与生成客户端 |
| db | 跨模块数据库工具与迁移基础设施 |
| jobs | 后台任务、邮件投递与 outbox 基础设施 |
| deps / tooling / repo | 依赖、开发工具和仓库级规则 |

迁移优先使用所属业务模块，如 `fix (identity): enforce unique verified emails`；只有迁移工具本身变更才用 db。业务功能带文档、测试和迁移时用业务 scope，不因为文件跨目录就用 repo。

新增领域可以增加清楚的 scope；不使用 `misc`、`all` 等掩盖改动范围。多个无关目的应拆提交；必要跨模块原子变更选择主要职责，在正文说明其余影响。

## 4. 正文、破坏性变更与回退

简单提交可以只有标题。复杂改动空一行后写正文，说明问题、主要变化、验证和必要的兼容/迁移信息；不复制聊天记录或逐文件清单。

破坏性变更可在 scope 后添加 `!`，并写明 `BREAKING CHANGE:`，例如：

```text
feat (api)!: rename application status field

Replace status with publication_status in application responses.
Update API consumers before deploying this version.

BREAKING CHANGE: clients must read publication_status instead of status.
```

有数据库变更时正文说明目标模块、部署顺序和真实存在的迁移入口；未验证的命令不能写成可执行步骤。复杂验证可关联 PR 或报告，但不得谎称测试通过。

revert 写明撤销对象及原因；如果仓库采用本规范，人工处理 Git 默认生成标题使其符合格式。schema 回退需另外判断数据安全，不能因代码 revert 就声称数据库已恢复。

## 5. 提交边界与检查

- 一个提交尽量形成可理解、可验证的完整变化；关联的规范、测试和迁移一起更新。
- 不提交秘密、真实账号数据、临时输出、依赖安装目录或无关改动；生成产物按项目约定维护。
- 提交前检查实际 diff 和暂存区，只暂存本任务明确路径；不使用仓库级盲目暂存。
- 使用与变更相称的检查，并如实说明未运行的验证。文档任务只需相关文档检查，不强制全套业务测试。
- 不自动 amend、重写历史、切换共享分支或强推；不覆盖用户/其他会话的工作。
- PR 若采用 squash，最终 squash 标题也遵守本格式；合并策略尚未确定，本规范不强制选择。

没有因本规范安装 commitlint、husky 或其他 hooks，也不修改 Git 配置。功能提交授权来自上述用户明确要求；提交后检查实际提交内容与剩余工作区改动，并向用户报告提交 ID。
