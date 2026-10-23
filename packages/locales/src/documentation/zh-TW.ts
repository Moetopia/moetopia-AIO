import type { DocumentationMessages } from './types'
export default {
  "labels": {
    "test": "線上測試",
    "run": "傳送請求",
    "openTest": "開啟測試視窗",
    "clearTest": "清除測試資料",
    "testHint": "請求傳送至目前站點。寫入操作會修改實際資料，請使用測試帳號與應用。輸入及回應只留在目前頁面，離開即清除。",
    "serverOnly": "此介面需要應用後端密鑰，請在自己的後端或終端測試，不在文件頁面輸入 client_secret。",
    "adminOnly": "此介面使用獨立管理員會話，請在管理台執行對應操作。使用者站不會借用管理員 Cookie。",
    "requestBody": "JSON 請求主體",
    "bearer": "Access token",
    "result": "實際回應",
    "elapsed": "耗時",
    "cancelTest": "停止等待",
    "testFailure": "請求未完成，請檢查服務、參數或逾時。寫入結果可能尚未確定，不會自動重試。",
    "invalidTest": "請填寫必填參數並檢查 JSON 格式。",
    "redirectTest": "授權與登出使用瀏覽器跳轉，請在新視窗完成。",
    "pkce": "產生 state / nonce / PKCE",
    "verifier": "本次 code_verifier",
    "testOperation": "選擇測試介面",
    "testSecret": "僅用於本次測試，切換介面或離開頁面即清除。",
    "terminal": "終端測試範本（替換預留值後執行）",
    "title": "開發者文件",
    "description": "從第一個登入請求，到可以上線的身分串接。",
    "guides": "串接指南",
    "reference": "API 參考",
    "outline": "本頁內容",
    "search": "搜尋介面",
    "searchPlaceholder": "路徑、介面名稱或 operationId",
    "group": "介面分組",
    "method": "HTTP 方法",
    "all": "全部",
    "empty": "沒有符合的介面，請調整篩選條件。",
    "download": "下載 OpenAPI",
    "source": "介面結構來自此版本產生的 OpenAPI，部署能力以目標 issuer 的 Discovery 為準。",
    "languageNote": "介面標題與欄位說明保留後端契約原文；指南與介面支援四種語言。",
    "parameters": "請求參數",
    "body": "請求主體",
    "responses": "回應",
    "authentication": "驗證與呼叫邊界",
    "required": "必填",
    "optional": "選填",
    "none": "無欄位",
    "raw": "原始介面定義",
    "schema": "欄位結構",
    "requestExample": "請求範例",
    "exampleNote": "範例使用預留值，僅展示格式。線上測試請使用測試帳號與 Public 用戶端，不在此頁輸入後端密鑰。",
    "protocol": "第三方 OAuth / OIDC",
    "internal": "站點業務介面",
    "internalNote": "僅供本站業務流程：使用對應的 HttpOnly Cookie、Origin 與 CSRF。管理業務另需角色及有效 MFA；OAuth Bearer 不能取代這些條件。",
    "protocolNote": "依已註冊的用戶端類型與流程呼叫；Basic 僅供應用後端使用，Bearer 應傳 access token。條件必填規則請見說明與 oneOf 原始定義。",
    "back": "返回 API 清單",
    "related": "相關資料",
    "previous": "上一篇",
    "next": "下一篇",
    "retry": "重新載入",
    "loadError": "暫時無法載入文件，請重試。",
    "fields": "欄位",
    "status": "狀態碼",
    "media": "內容類型",
    "headers": "回應標頭",
    "noBody": "無回應主體",
    "deprecated": "已棄用",
    "contract": "查看介面契約",
    "count": "介面",
    "sdk": "建議使用持續維護的標準 OIDC 用戶端函式庫。下方 HTTP 範例用於理解協定，不取代函式庫的回呼與權杖驗證。"
  },
  "guides": {
    "overview": {
      "title": "快速開始",
      "description": "先完成一個授權碼登入，再接入所需功能。",
      "sections": [
        {
          "id": "prepare",
          "title": "建立測試應用",
          "paragraphs": [
            "在「我的應用」填寫名稱與回呼。應用 UUID 用於控制台，client_id 才用於 OAuth。無法保密選 Public，能在後端保管密鑰選 Confidential。",
            "測試應用只供擁有者與明確加入的測試成員使用。建立草稿或送審不代表已發布。"
          ],
          "code": "discovery",
          "operations": [
            "oidc-discovery"
          ]
        },
        {
          "id": "connect",
          "title": "完成登入流程",
          "paragraphs": [
            "從目標 issuer 取得 Discovery，使用其中的 authorization_endpoint、token_endpoint、jwks_uri，不從文件站域名推測正式 issuer。",
            "每次登入產生新的 state、nonce、PKCE verifier/S256 challenge 並綁定應用會話。瀏覽器跳轉授權，回呼時驗證 state、兌換授權碼與驗證 ID Token，再建立自己的登入會話。"
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
          "title": "測試與正式發布",
          "paragraphs": [
            "測試拒絕授權、錯誤回呼、重放與過期。需要測試群組以外的使用者時再送審。",
            "敏感設定變更形成新審核版本；核准前舊核准版本繼續服務。請一併閱讀刷新與登出章節。"
          ]
        }
      ]
    },
    "authentication": {
      "title": "授權碼登入",
      "description": "使用 Authorization Code、PKCE S256 與標準 OIDC 驗證。",
      "sections": [
        {
          "id": "transaction",
          "title": "建立單次登入交易",
          "paragraphs": [
            "每次使用密碼學安全的 state、nonce 與 code_verifier。state 與 OIDC nonce 至少八個字元，實際應使用高熵亂數並綁定瀏覽器交易。challenge 為 SHA-256 後的無填充 base64url。",
            "只支援 code、query 與 S256。openid 必須帶 nonce；一次使用單一允許的 audience，只有一個註冊 audience 時可省略。回呼依註冊規則精確匹配。"
          ],
          "code": "authorize",
          "operations": [
            "oauth-authorize"
          ]
        },
        {
          "id": "callback",
          "title": "處理回呼與驗證權杖",
          "paragraphs": [
            "先處理 error 並驗證 state、交易時效，再以 form 兌換授權碼。使用原 verifier 與相同 redirect_uri；授權碼單次使用，不自動重試已消費的碼。",
            "由 OIDC 函式庫驗證簽章、issuer、audience/client_id、時效、nonce 及適用的 azp、auth_time、at_hash。只解析 JWT 不足以登入。成功後消費交易，以 (iss, sub) 識別使用者。"
          ],
          "code": "token",
          "operations": [
            "oauth-token",
            "oauth-jwks"
          ]
        },
        {
          "id": "claims",
          "title": "取得核准資料",
          "paragraphs": [
            "UserInfo 使用 access token，不能傳 ID Token。email/profile 依核准 scope 回傳，可變資料不能當永久主鍵。",
            "prompt=login 強制重新認證，consent 要求同意，none 目前回傳 interaction_required；max_age=0 要求本次請求之後重新登入。"
          ],
          "code": "userinfo",
          "operations": [
            "oauth-userinfo"
          ]
        }
      ]
    },
    "credentials": {
      "title": "應用與憑證",
      "description": "選擇用戶端類型並理解測試、審核與生效版本。",
      "sections": [
        {
          "id": "types",
          "title": "Public 與 Confidential",
          "paragraphs": [
            "無法保密選 Public；放進 JavaScript 或安裝包的 secret 不能當保密憑證。Public 使用 client_id/PKCE，Confidential 在後端保管密鑰。",
            "使用標準 OAuth 函式庫處理 Basic 編碼，不混用表單認證。內省僅接受 Confidential HTTP Basic；機器憑證流程也需要獲准的 Confidential。"
          ]
        },
        {
          "id": "versions",
          "title": "測試與審核版本",
          "paragraphs": [
            "擁有者可加入測試成員；移除後線上撤銷相關授權資格。待審版本不可編輯，拒絕後可修改再提交。",
            "啟用狀態、平台暫停與核准版本共同決定可用性。重新啟用不能解除平台暫停，也不恢復已撤銷授權。"
          ],
          "operations": [
            "get-api-v1-developer-applications",
            "get-api-v1-developer-applications-by-id-versions"
          ]
        },
        {
          "id": "secrets",
          "title": "密鑰生命週期",
          "paragraphs": [
            "Confidential secret 僅在建立或輪換後顯示一次，交給後端秘密管理系統，不放前端、URL、日誌或截圖。",
            "輪換立即廢止舊密鑰；準備同步切換，使用最新 revision 與本人驗證。網路結果不明時不能盲目重試輪換。"
          ],
          "operations": [
            "post-api-v1-developer-applications-by-id-secret"
          ]
        }
      ]
    },
    "tokens": {
      "title": "權杖、刷新與撤銷",
      "description": "分別處理 access token、ID Token 與 refresh token。",
      "sections": [
        {
          "id": "types",
          "title": "三種權杖職責",
          "paragraphs": [
            "ID Token 驗證登入身分，access token 存取核准資源，refresh token 續期。三者不能互換，機器權杖不能當使用者登入。",
            "憑證留在後端，瀏覽器使用自己的安全會話。離線 JWT 必須驗證 issuer、audience、演算法、時效與 scope，且存在撤銷傳播窗口。"
          ]
        },
        {
          "id": "refresh",
          "title": "嚴格刷新輪換",
          "paragraphs": [
            "授權碼流程須允許對應 grant 且明確同意 offline_access 才會取得刷新權杖。刷新必須序列化，成功後原子替換新值；重用舊值撤銷整族，沒有並行寬限。",
            "目前 scope/audience 應省略沿用原授權，或傳完全相同值；不能更換 audience，也不要依賴縮減 scope。結果不明時不要盲目重試舊刷新權杖。"
          ],
          "code": "refresh",
          "operations": [
            "oauth-token"
          ]
        },
        {
          "id": "revoke",
          "title": "撤銷與線上檢查",
          "paragraphs": [
            "撤銷對未知、失效或其他用戶端權杖也回傳空 200，不表示權杖存在，也不等於全部站點登出。",
            "內省只供 Confidential 後端檢查自己的 access/refresh；非活躍只含 active=false。需要即時狀態時不能快取 active=true 冒充線上檢查。"
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
      "title": "權限與使用者資料",
      "description": "申請最小權限，區分身分與資源存取。",
      "sections": [
        {
          "id": "scopes",
          "title": "申請與核准 scope",
          "paragraphs": [
            "openid 啟用 OIDC，profile/email 允許對應資料，offline_access 用於核准的離線刷新。只能申請註冊與審核允許的權限，並讀取 token 回應的實際 scope。",
            "不要搬用其他生態模組權限。使用業務 API 前先確認資源伺服器、audience 與權限定義；UserInfo 不是通用資料庫。"
          ]
        },
        {
          "id": "identity",
          "title": "安全映射帳號",
          "paragraphs": [
            "使用 (iss, sub) 作為外部身分鍵，不能按可變郵箱合併帳號。UserInfo sub 必須與 ID Token 相同，可選欄位缺失時保留未知。",
            "測試成員不是管理員。account/portal/admin 業務介面用各自 Cookie 會話，不允許任意第三方 Bearer 呼叫。"
          ],
          "code": "userinfo",
          "operations": [
            "oauth-userinfo"
          ]
        }
      ]
    },
    "logout": {
      "title": "登出",
      "description": "區分應用本地登出、中央會話登出與授權撤銷。",
      "sections": [
        {
          "id": "local",
          "title": "先結束應用會話",
          "paragraphs": [
            "自己的後端必須清除會話與 Cookie，清理 localStorage 不等於伺服器撤銷。Moetopia 使用者站登出只影響它自己的 RP。",
            "登出不自動撤銷 OAuth 授權；需要結束長期存取時另行撤銷權杖。"
          ]
        },
        {
          "id": "central",
          "title": "請求中央登出",
          "paragraphs": [
            "從 Discovery 讀取 end_session_endpoint。提供此 issuer 有效的 id_token_hint 或 client_id；退出回呼精確匹配核准名單，並驗證 state。",
            "先進入明確確認頁，由使用者決定結束中央會話。不要自動確認，不能保證所有 RP 與裝置即時退出。"
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
      "title": "錯誤與排查",
      "description": "辨識協定錯誤、業務錯誤及網路結果不明。",
      "sections": [
        {
          "id": "protocol",
          "title": "OAuth/OIDC 錯誤",
          "paragraphs": [
            "協定使用 error/error_description，可能透過可信回呼跳轉回傳；無可信回呼時直接報錯。回呼仍要驗證 state，HTTP 200 不足以證明成功。",
            "invalid_client 查認證方式；invalid_grant 查時效、消費狀態、verifier 與 redirect_uri；access_denied 是拒絕，interaction_required 需互動登入。"
          ]
        },
        {
          "id": "business",
          "title": "本站業務 API",
          "paragraphs": [
            "業務使用 Problem Details，可含 code/欄位錯誤。401 重認證，403 查權限/MFA/Origin/CSRF，409 重讀 revision 再確認，422 修正輸入，429 遵循 Retry-After，503 保留輸入。",
            "空的可選查詢參數應省略，不能用裸 ?cursor/?after 代表首頁。next_cursor/next_after 為空代表末頁。"
          ]
        },
        {
          "id": "diagnose",
          "title": "安全收集證據",
          "paragraphs": [
            "記錄方法、端點、狀態、非敏感 request_id、用戶端類型與復現步驟。不要記錄 Cookie、Authorization、密碼、驗證碼、授權碼、verifier 或完整正文。",
            "檢查 Discovery、issuer、時間、回呼名單、scope 與應用審核/暫停狀態，不透過關閉 PKCE 或驗簽排錯。"
          ]
        }
      ]
    },
    "production": {
      "title": "上線檢查",
      "description": "在真實部署與自己的應用後端完成驗收。",
      "sections": [
        {
          "id": "configuration",
          "title": "正式設定與秘密",
          "paragraphs": [
            "確認正式 issuer、HTTPS 回呼與正確 Discovery，不直接複製開發 loopback 或參考專案域名與憑證。",
            "確認應用已核准、啟用且未暫停。保護秘密、同步時鐘、設定依賴逾時與受控重試，憑證不得進日誌。"
          ]
        },
        {
          "id": "acceptance",
          "title": "驗收場景",
          "paragraphs": [
            "測試登入成功/拒絕、錯誤 state/nonce/verifier、回呼與授權碼重放、過期、並行刷新、舊值重用、密鑰輪換、應用暫停與帳號停用。單元測試不是正式互通驗收。",
            "確認本地登出、中央確認、撤銷與離線 JWT 窗口；監控、備份、恢復、密鑰切換要有負責人與操作步驟。"
          ]
        },
        {
          "id": "limits",
          "title": "能力邊界",
          "paragraphs": [
            "目前不提供動態用戶端註冊、隱式流程、密碼授權或通用多租戶，函式庫支援不等於本站開放。",
            "以目標 Discovery、核准設定與本版 API 為準，容量、正式權杖時效及外部身分供應商需另行驗收。"
          ]
        }
      ]
    }
  }
} satisfies DocumentationMessages
