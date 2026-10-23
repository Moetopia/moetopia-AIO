import type { DocumentationMessages } from './types'
export default {
  "labels": {
    "test": "在线测试",
    "run": "发送请求",
    "openTest": "打开测试窗口",
    "clearTest": "清空测试数据",
    "testHint": "请求发送到当前站点。写操作会修改实际数据；请使用测试账号和测试应用。输入与响应只保留在当前页面，离开即清空。",
    "serverOnly": "此接口需要应用后端密钥，请在自己的后端或终端测试，不在文档页面输入 client_secret。",
    "adminOnly": "此接口使用独立管理员会话，请在管理台执行对应操作。用户站不会借用管理员 Cookie。",
    "requestBody": "JSON 请求体",
    "bearer": "Access token",
    "result": "实际响应",
    "elapsed": "耗时",
    "cancelTest": "停止等待",
    "testFailure": "请求未完成，请检查服务、参数或超时。写操作的结果可能尚未确定，不会自动重试。",
    "invalidTest": "请填写必填参数，并检查 JSON 格式。",
    "redirectTest": "授权与退出通过浏览器跳转，请在新窗口完成。",
    "pkce": "生成 state / nonce / PKCE",
    "verifier": "本次 code_verifier",
    "testOperation": "选择测试接口",
    "testSecret": "仅用于当前测试，切换接口或离开页面会清除。",
    "terminal": "终端测试模板（替换占位值后执行）",
    "title": "开发者文档",
    "description": "从第一个登录请求，到可以上线的身份接入。",
    "guides": "接入指南",
    "reference": "API 参考",
    "outline": "本页内容",
    "search": "搜索接口",
    "searchPlaceholder": "路径、接口名称或 operationId",
    "group": "接口分组",
    "method": "HTTP 方法",
    "all": "全部",
    "empty": "没有找到匹配的接口，请调整筛选条件。",
    "download": "下载 OpenAPI",
    "source": "接口结构来自本版本生成的 OpenAPI，部署能力以目标 issuer 的 Discovery 为准。",
    "languageNote": "接口标题与字段说明保留后端契约原文；指南和界面支持四种语言。",
    "parameters": "请求参数",
    "body": "请求体",
    "responses": "响应",
    "authentication": "鉴权与调用边界",
    "required": "必填",
    "optional": "可选",
    "none": "无字段",
    "raw": "原始接口定义",
    "schema": "字段结构",
    "requestExample": "请求示例",
    "exampleNote": "示例使用占位值，仅演示请求格式。在线测试请使用测试账号与 Public 客户端；应用后端密钥不输入此页面。",
    "protocol": "第三方 OAuth / OIDC",
    "internal": "站点业务接口",
    "internalNote": "仅供本站业务流程：使用对应的 HttpOnly Cookie、Origin 和 CSRF。管理员业务另需角色与有效 MFA；OAuth Bearer 不能替代这些条件。",
    "protocolNote": "按已注册的客户端类型与流程调用；Basic 仅用于应用后端，Bearer 应传 access token。条件必填规则见说明和 oneOf 原始定义。",
    "back": "返回 API 列表",
    "related": "相关资料",
    "previous": "上一篇",
    "next": "下一篇",
    "retry": "重新加载",
    "loadError": "文档暂时无法加载，请重试。",
    "fields": "字段",
    "status": "状态码",
    "media": "内容类型",
    "headers": "响应头",
    "noBody": "无响应体",
    "deprecated": "已弃用",
    "contract": "查看接口契约",
    "count": "接口",
    "sdk": "推荐使用维护中的标准 OIDC 客户端库。下方 HTTP 示例用于理解协议，不替代库的回调与令牌校验。"
  },
  "guides": {
    "overview": {
      "title": "快速开始",
      "description": "先跑通一个授权码登录，再逐步接入所需能力。",
      "sections": [
        {
          "id": "prepare",
          "title": "创建测试应用",
          "paragraphs": [
            "在“我的应用”创建应用，填写名称和回调地址。应用 UUID 用于控制台管理，client_id 才用于 OAuth 请求；不要混用。Public 适合无法保密的浏览器或原生应用，Confidential 适合能在服务端保管密钥的应用。",
            "测试应用仅供所有者和显式添加的测试成员使用。注册、草稿保存或提交审核不代表已经发布。"
          ],
          "code": "discovery",
          "operations": [
            "oidc-discovery"
          ]
        },
        {
          "id": "connect",
          "title": "完成登录闭环",
          "paragraphs": [
            "从目标服务的 issuer 获取 Discovery，使用返回的 authorization_endpoint、token_endpoint、jwks_uri 等地址，不从本站文档域名猜测生产 issuer。",
            "为每次登录生成新的 state、nonce、PKCE verifier 与 S256 challenge，将事务绑定在应用自己的服务端会话中。浏览器跳转授权；回调后校验 state、兑换授权码并验证 ID Token，再建立应用自己的登录会话。"
          ],
          "code": "authorize",
          "operations": [
            "oauth-authorize",
            "oauth-token",
            "oauth-userinfo"
          ]
        },
        {
          "id": "publish",
          "title": "测试与正式发布",
          "paragraphs": [
            "验证拒绝授权、回调错误、重放和令牌失效等失败场景。需要其他用户接入时再提交审核；审核前仍按测试范围限制。",
            "后续敏感修改形成新的审核版本，获批前旧批准版本继续服务。上线检查、刷新轮换和退出规则见后续章节。"
          ]
        }
      ]
    },
    "authentication": {
      "title": "授权码登录",
      "description": "使用 Authorization Code、PKCE S256 与标准 OIDC 验证完成登录。",
      "sections": [
        {
          "id": "transaction",
          "title": "建立一次性登录事务",
          "paragraphs": [
            "每次请求使用密码学安全随机的 state、nonce 和 code_verifier。state 至少八个字符，OIDC 的 nonce 至少八个字符；实际使用高熵随机值，并绑定同一浏览器的登录事务。code_challenge 是 verifier 的 SHA-256 摘要经 base64url 无填充编码。",
            "只支持 response_type=code、response_mode=query 和 S256。openid 请求必须有 nonce；授权请求使用单一允许的 audience，只有一个注册 audience 时可省略。回调地址按注册规则精确匹配。"
          ],
          "code": "authorize",
          "operations": [
            "oauth-authorize"
          ]
        },
        {
          "id": "callback",
          "title": "处理回调与令牌验证",
          "paragraphs": [
            "先处理 error 并验证 state 和事务时效，再在服务端发送 form 编码的 token 请求。使用原始 verifier 和相同 redirect_uri；授权码仅能兑换一次，不自动重试已消费的授权码。",
            "使用 OIDC 库校验签名、固定 issuer、audience/client_id、有效期、nonce 以及适用的 azp、auth_time、at_hash。不要只解析 JWT 内容就视为已登录。成功后销毁登录事务，并使用 (iss, sub) 识别用户。"
          ],
          "code": "token",
          "operations": [
            "oauth-token",
            "oauth-jwks"
          ]
        },
        {
          "id": "claims",
          "title": "获取用户资料",
          "paragraphs": [
            "UserInfo 使用 access token，而不是 ID Token。email/profile 资料只有获得对应 scope 才返回，邮箱和昵称都可能变化，不能作为永久账号主键。",
            "prompt=login 强制重新认证；prompt=consent 要求同意；prompt=none 当前返回 interaction_required。max_age=0 要求本次请求后的重新登录。"
          ],
          "code": "userinfo",
          "operations": [
            "oauth-userinfo"
          ]
        }
      ]
    },
    "credentials": {
      "title": "应用与凭据",
      "description": "选择正确的客户端类型，理解测试、审核和生效版本。",
      "sections": [
        {
          "id": "types",
          "title": "Public 与 Confidential",
          "paragraphs": [
            "不能保密的客户端选择 Public，不能用打包在 JavaScript、手机安装包或仓库里的 secret 假装 Confidential。Public 使用 client_id 与 PKCE；Confidential 将 secret 留在自己的后端并使用注册的客户端认证方式。",
            "Basic 的 client_id/secret 编码应由标准 OAuth 库处理，不混用 Basic 与表单凭据。Introspection 只支持 Confidential 的 HTTP Basic；机器凭据流程也只适用于获准的 Confidential 客户端。"
          ]
        },
        {
          "id": "versions",
          "title": "测试、审核与版本",
          "paragraphs": [
            "应用所有者可以添加显式测试成员，移除后会在线撤销相关授权资格。提交审核后待审版本不可编辑；拒绝后可修改并再次提交。",
            "当前启用状态、平台暂停状态和批准版本共同决定可用性。所有者重新启用不能解除平台暂停，也不会恢复已撤销的授权。"
          ],
          "operations": [
            "get-api-v1-developer-applications",
            "get-api-v1-developer-applications-by-id-versions"
          ]
        },
        {
          "id": "secrets",
          "title": "密钥生命周期",
          "paragraphs": [
            "Confidential secret 仅创建或轮换成功时显示一次，应交给应用后端的秘密管理系统。不要放入前端、URL、日志或截图。",
            "轮换会使旧 secret 立即失效；准备好切换流程，使用当前 revision 与本人验证，不自动重试轮换或假设网络失败代表轮换没有生效。"
          ],
          "operations": [
            "post-api-v1-developer-applications-by-id-secret"
          ]
        }
      ]
    },
    "tokens": {
      "title": "令牌、刷新与撤销",
      "description": "分别处理 access token、ID Token 和 refresh token。",
      "sections": [
        {
          "id": "types",
          "title": "三种令牌的职责",
          "paragraphs": [
            "ID Token 用来验证本次登录身份；access token 用于获准的资源 API；refresh token 用于续期。三者不能互换，机器令牌也不能当作用户登录。",
            "在应用后端保管凭据，浏览器使用应用自己的安全会话。离线 JWT 验签必须核对 issuer、audience、签名算法、时效和所需 scope；离线验签存在撤销传播窗口。"
          ]
        },
        {
          "id": "refresh",
          "title": "严格刷新轮换",
          "paragraphs": [
            "只有客户端配置允许且用户明确同意 offline_access 的授权码流程才会签发 refresh token。刷新必须串行，成功后原子替换为新的 refresh token；旧值重用会撤销整族，没有并发宽限。",
            "当前刷新只接受省略 scope/audience 以沿用原授权，或传入与原授权相同的值；不能更换 audience，也不要依赖缩减 scope 的行为。网络结果不明时不要盲目重试旧刷新令牌，应按客户端安全策略重新认证。"
          ],
          "code": "refresh",
          "operations": [
            "oauth-token"
          ]
        },
        {
          "id": "revoke",
          "title": "撤销与在线检查",
          "paragraphs": [
            "POST /oauth/revoke 对未知、已失效或不属于当前客户端的 token 也返回空 200，不能据此判断令牌是否存在。撤销用户授权族不等于退出全部站点会话。",
            "POST /oauth/introspect 仅供 Confidential 后端查询自己的 access/refresh；非活跃只含 active=false。需立即检查状态时使用在线接口，且不要缓存 active=true 来冒充实时性。"
          ],
          "code": "introspect",
          "operations": [
            "oauth-revoke",
            "oauth-introspect"
          ]
        }
      ]
    },
    "scopes": {
      "title": "权限与用户资料",
      "description": "按最小权限申请，并区分用户身份与资源访问。",
      "sections": [
        {
          "id": "scopes",
          "title": "请求与批准",
          "paragraphs": [
            "openid 进入 OIDC 流程；profile 允许公开资料；email 允许邮箱资料；offline_access 用于获准的离线刷新。应用只能申请注册/审核允许的 scope，并以实际 token 响应中的 scope 为准。",
            "不要把其他生态模块的权限照搬到身份服务中。需要业务 API 时，先确认目标资源服务器、audience 及它定义的权限，而不是把 UserInfo 当成通用资料数据库。"
          ]
        },
        {
          "id": "identity",
          "title": "用户映射",
          "paragraphs": [
            "使用 (iss, sub) 作为外部身份键，不能使用可变邮箱合并账号。UserInfo 的 sub 必须与本次 ID Token 的 sub 一致，缺少可选字段时保留未知状态。",
            "测试成员不等于管理员，用户授权不等于应用获得平台管理权限。本站 account/portal/admin 业务 API 使用各自 Cookie 会话，不开放给任意第三方 Bearer 调用。"
          ],
          "code": "userinfo",
          "operations": [
            "oauth-userinfo"
          ]
        }
      ]
    },
    "logout": {
      "title": "退出登录",
      "description": "区分应用本地退出、中央会话退出和授权撤销。",
      "sections": [
        {
          "id": "local",
          "title": "先结束应用自己的会话",
          "paragraphs": [
            "应用本地退出由你的服务端清除自己的会话与 Cookie。Moetopia 用户站退出只影响其 RP 会话，不能用清理 localStorage 代替服务端会话撤销。",
            "结束登录不会自动撤销此前的 OAuth 授权；如需撤销长期访问，应另外调用撤销端点。"
          ]
        },
        {
          "id": "central",
          "title": "请求中央退出",
          "paragraphs": [
            "从 Discovery 读取 end_session_endpoint。可携带本 issuer 有效的 id_token_hint，或使用 client_id；退出回调必须精确匹配已批准的 post_logout_redirect_uri，并使用 state 验证回跳。",
            "请求会进入显式确认页面，由用户决定是否结束中央身份会话。不要自动确认，也不要宣称所有 RP、所有设备会立即同步退出。"
          ],
          "code": "logout",
          "operations": [
            "oauth-logout-get",
            "oauth-logout-post"
          ]
        }
      ]
    },
    "errors": {
      "title": "错误与排查",
      "description": "先识别协议错误、业务错误和网络结果不明，再决定如何恢复。",
      "sections": [
        {
          "id": "protocol",
          "title": "OAuth/OIDC 错误",
          "paragraphs": [
            "协议端点使用 error/error_description，授权错误可能通过已经验证的回调重定向返回；没有可信回调时可直接返回错误。回调处理必须同时验证 state，不能只看 HTTP 200。",
            "invalid_client 检查客户端类型与认证方式；invalid_grant 检查授权码/刷新是否过期、已消费、verifier 和 redirect_uri 是否一致；access_denied 表示用户拒绝；interaction_required 应发起交互登录。"
          ]
        },
        {
          "id": "business",
          "title": "本站业务 API",
          "paragraphs": [
            "业务 API 使用 Problem Details，可附 code 和字段错误；401 重新认证，403 检查权限/MFA/Origin/CSRF，409 重读最新 revision 后让用户重新决定，422 检查输入，429 遵循 Retry-After，503 保留输入并稍后重试。",
            "可选查询参数为空时省略，不能发送 ?cursor 或 ?after 作为首页占位。分页从响应的 next_cursor/next_after 取得；空字符串代表末页，不把它再作为下一页参数。"
          ]
        },
        {
          "id": "diagnose",
          "title": "安全地定位失败",
          "paragraphs": [
            "记录请求方法、端点、状态码和非敏感 request_id，并说明使用的客户端类型、流程与复现步骤。不要记录或分享 Cookie、Authorization、密码、验证码、授权码、verifier、密钥或完整请求正文。",
            "先从 Discovery 确认 issuer，检查本机时间、回调白名单、注册范围和应用测试/发布/暂停状态。不要通过关闭 PKCE、放宽重定向或跳过验签来排错。"
          ]
        }
      ]
    },
    "production": {
      "title": "上线检查",
      "description": "在真实部署地址与应用后端上完成验收。",
      "sections": [
        {
          "id": "configuration",
          "title": "环境与配置",
          "paragraphs": [
            "确定正式 issuer 和 HTTPS 回调，使用准确的 Discovery 与客户端配置。开发 loopback 回调规则不应直接作为生产配置；不要复制参考项目的域名、端口或凭据。",
            "确认客户端已获审核、启用且未被平台暂停。将 secret/令牌放入受控秘密存储，配置时钟同步、依赖超时与受控重试，禁止凭据进入日志。"
          ]
        },
        {
          "id": "acceptance",
          "title": "必须覆盖的场景",
          "paragraphs": [
            "完成登录成功/拒绝、错误 state/nonce/verifier、重复回调、授权码重放、过期、并发刷新与旧值重用、密钥轮换、应用暂停和账号停用的检查。协议测试与实际生产互操作分别记录，不将单元测试通过视为上线完成。",
            "验证本地退出、中央确认、撤销与下游离线 JWT 窗口。应用自身的监控、备份、恢复和密钥切换应有负责人及可执行步骤。"
          ]
        },
        {
          "id": "limits",
          "title": "当前能力边界",
          "paragraphs": [
            "当前不提供动态客户端注册、隐式流、密码授权或通用多租户。不要从协议库支持范围推断本站已经开放的能力。",
            "以目标环境 Discovery、正式审核配置与本版本 API 参考为准；容量、生产令牌时效和实际外部供应商接入需独立验收。"
          ]
        }
      ]
    }
  }
} satisfies DocumentationMessages
