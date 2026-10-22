import type { DocumentationMessages } from './types'
export default {
  "labels": {
    "title": "開発者ドキュメント",
    "description": "最初のログインリクエストから、本番公開できる認証連携まで。",
    "guides": "連携ガイド",
    "reference": "API リファレンス",
    "outline": "このページの内容",
    "search": "エンドポイントを検索",
    "searchPlaceholder": "パス、操作名、operationId",
    "group": "API グループ",
    "method": "HTTP メソッド",
    "all": "すべて",
    "empty": "一致するエンドポイントがありません。絞り込み条件を変更してください。",
    "download": "OpenAPI をダウンロード",
    "source": "API 構造はこのビルドの生成済み OpenAPI に基づきます。実際の対応機能は対象 issuer の Discovery で確認してください。",
    "languageNote": "操作名とフィールド説明はバックエンド契約の原文です。ガイドと操作 UI は四言語に対応しています。",
    "parameters": "リクエストパラメーター",
    "body": "リクエストボディ",
    "responses": "レスポンス",
    "authentication": "認証と利用範囲",
    "required": "必須",
    "optional": "任意",
    "none": "フィールドなし",
    "raw": "操作の元定義",
    "schema": "フィールド構造",
    "requestExample": "リクエスト例",
    "exampleNote": "例はプレースホルダーを使った形式の説明です。自分の issuer、クライアント、コールバックに置き換えてください。このページに実際の認証情報を貼り付けないでください。",
    "protocol": "外部 OAuth / OIDC",
    "internal": "サイト業務 API",
    "internalNote": "対応する HttpOnly Cookie、Origin、CSRF を使うサイト用 API です。管理操作には権限と有効な MFA も必要です。OAuth Bearer では代用できません。",
    "protocolNote": "登録済みクライアント種別とフローに従って呼び出してください。Basic はアプリのサーバー専用、Bearer は access token 用です。条件付き必須項目は説明と oneOf の元定義を確認してください。",
    "back": "API 一覧へ戻る",
    "related": "関連資料",
    "previous": "前へ",
    "next": "次へ",
    "retry": "再読み込み",
    "loadError": "ドキュメントを読み込めませんでした。再試行してください。",
    "fields": "フィールド",
    "status": "ステータス",
    "media": "コンテンツタイプ",
    "headers": "レスポンスヘッダー",
    "noBody": "レスポンスボディなし",
    "deprecated": "非推奨",
    "contract": "API 契約を見る",
    "count": "操作",
    "sdk": "保守されている標準 OIDC クライアントライブラリを使用してください。HTTP 例はプロトコルの説明用であり、ライブラリのコールバック検証やトークン検証の代わりにはなりません。"
  },
  "guides": {
    "overview": {
      "title": "クイックスタート",
      "description": "まず認可コードによるログインを一巡させます。",
      "sections": [
        {
          "id": "prepare",
          "title": "テストアプリを作る",
          "paragraphs": [
            "「自分のアプリ」で名前とコールバックを登録します。管理用のアプリ UUID と OAuth 用 client_id は別です。秘密を守れない環境は Public、サーバーで秘密を管理できる場合は Confidential を選びます。",
            "テスト段階では所有者と明示的なテストメンバーだけが利用できます。作成や審査申請だけでは公開されません。"
          ],
          "code": "discovery",
          "operations": [
            "oidc-discovery"
          ]
        },
        {
          "id": "connect",
          "title": "ログインを完成させる",
          "paragraphs": [
            "対象 issuer の Discovery から authorization_endpoint、token_endpoint、jwks_uri を取得します。このドキュメントのドメインを本番 issuer と推測しないでください。",
            "ログインごとに state、nonce、PKCE verifier と S256 challenge を生成して自分のセッションに結び付けます。認可へ遷移し、コールバックと ID Token を検証した後、自分のアプリのセッションを作成します。"
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
          "title": "テストして公開する",
          "paragraphs": [
            "同意拒否、無効なコールバック、リプレイ、有効期限切れを確認してから審査を申請します。",
            "重要な設定変更は新しい審査バージョンとなります。承認までは以前の承認済み設定が有効です。刷新とログアウトのガイドも確認してください。"
          ]
        }
      ]
    },
    "authentication": {
      "title": "認可コードログイン",
      "description": "Authorization Code、PKCE S256 と標準 OIDC 検証を使います。",
      "sections": [
        {
          "id": "transaction",
          "title": "一回限りの認証処理を作る",
          "paragraphs": [
            "暗号学的に安全な state、nonce、code_verifier を毎回生成します。state と OIDC nonce は最低八文字ですが、高エントロピーの乱数を使いブラウザーの認証処理に結び付けます。challenge は SHA-256 とパディングなし base64url で作ります。",
            "code、query、S256 のみ対応しています。openid には nonce が必要です。登録済み audience を一つ指定し、一つしか登録されていない場合は省略できます。コールバックは登録規則と一致させます。"
          ],
          "code": "authorize",
          "operations": [
            "oauth-authorize"
          ]
        },
        {
          "id": "callback",
          "title": "コールバックとトークンを検証する",
          "paragraphs": [
            "error、state、有効期限を確認してから form 形式で認可コードを交換します。元の verifier と同じ redirect_uri が必要です。使用済みコードを自動再試行しないでください。",
            "OIDC ライブラリで署名、issuer、audience/client_id、期限、nonce、必要な azp、auth_time、at_hash を検証します。JWT のデコードだけではログインできません。処理を消費し (iss, sub) でユーザーを識別します。"
          ],
          "code": "token",
          "operations": [
            "oauth-token",
            "oauth-jwks"
          ]
        },
        {
          "id": "claims",
          "title": "許可された情報を取得する",
          "paragraphs": [
            "UserInfo には ID Token ではなく access token を渡します。email/profile は許可された scope に従い、変更され得るため永続キーには使えません。",
            "prompt=login は再認証、consent は同意を要求します。none は現在 interaction_required を返します。max_age=0 は今回の要求開始後の認証を要求します。"
          ],
          "code": "userinfo",
          "operations": [
            "oauth-userinfo"
          ]
        }
      ]
    },
    "credentials": {
      "title": "アプリと認証情報",
      "description": "クライアント種別、テスト、審査と有効バージョンを理解します。",
      "sections": [
        {
          "id": "types",
          "title": "Public と Confidential",
          "paragraphs": [
            "秘密を保持できなければ Public を選びます。JavaScript や配布パッケージ内の secret は秘密ではありません。Public は client_id/PKCE、Confidential はサーバーの秘密管理を使います。",
            "Basic のエンコードは OAuth ライブラリに任せ、フォーム認証と混在させません。内省は Confidential の HTTP Basic のみ、client credentials も許可済み Confidential 専用です。"
          ]
        },
        {
          "id": "versions",
          "title": "テストと審査",
          "paragraphs": [
            "所有者はテストメンバーを追加できます。削除時には関連する認可資格をオンラインで失効させます。審査中は編集できず、却下後に修正して再申請できます。",
            "所有者の有効設定、プラットフォーム停止、承認バージョンが利用可否を決めます。再有効化しても停止や失効済み認可は解除されません。"
          ],
          "operations": [
            "get-api-v1-developer-applications",
            "get-api-v1-developer-applications-by-id-versions"
          ]
        },
        {
          "id": "secrets",
          "title": "秘密のライフサイクル",
          "paragraphs": [
            "Confidential secret は作成・更新成功時の一度だけ表示されます。バックエンドの秘密管理に保存し、フロントエンド、URL、ログや画像に残しません。",
            "更新すると古い secret は直ちに無効になります。最新 revision と本人確認で計画的に切り替え、結果不明な更新を無条件に再試行しないでください。"
          ],
          "operations": [
            "post-api-v1-developer-applications-by-id-secret"
          ]
        }
      ]
    },
    "tokens": {
      "title": "トークン、刷新と失効",
      "description": "access token、ID Token、refresh token を区別します。",
      "sections": [
        {
          "id": "types",
          "title": "三種類の役割",
          "paragraphs": [
            "ID Token はログインの確認、access token は許可済みリソースへのアクセス、refresh token は更新に使います。相互に代用できず、マシントークンはユーザーログインではありません。",
            "秘密はバックエンドに置き、ブラウザーは自分の安全なセッションを使います。JWT のオフライン検証には issuer、audience、アルゴリズム、期限、scope の確認が必要で、失効反映には時間差があります。"
          ]
        },
        {
          "id": "refresh",
          "title": "厳格なローテーション",
          "paragraphs": [
            "refresh token は対応 grant と明示的な offline_access 同意がある認可コードフローで発行されます。更新を直列化し成功後に新値へ原子的に置換します。旧値再利用は猶予なく認可全体を失効させます。",
            "現在 scope/audience は省略するか元と同一値を指定します。audience 変更や scope 縮小に依存しないでください。通信結果が不明な場合、古い refresh token を無条件に再送しません。"
          ],
          "code": "refresh",
          "operations": [
            "oauth-token"
          ]
        },
        {
          "id": "revoke",
          "title": "失効とオンライン検査",
          "paragraphs": [
            "失効 API は未知・期限切れ・他クライアントの token にも空の 200 を返します。存在確認や全サイトのログアウトを意味しません。",
            "内省は Confidential のバックエンドが自分の access/refresh を確認するためのものです。無効時は active=false のみです。active=true のキャッシュで即時確認を保証できません。"
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
      "title": "スコープとユーザー情報",
      "description": "最小限の権限を要求し、本人情報とリソース操作を分けます。",
      "sections": [
        {
          "id": "scopes",
          "title": "要求と許可",
          "paragraphs": [
            "openid は OIDC、profile/email は対応情報、offline_access は許可済み刷新を有効にします。登録・審査済みの権限だけを要求し、token レスポンスの実際の scope を確認します。",
            "別のエコシステムの権限を転用しません。業務 API を使う前にリソースサーバー、audience、権限定義を確認します。UserInfo は汎用データベースではありません。"
          ]
        },
        {
          "id": "identity",
          "title": "アカウントの対応付け",
          "paragraphs": [
            "外部本人キーには (iss, sub) を使います。変更可能なメールでアカウントを統合しません。UserInfo sub は ID Token と一致させ、省略された情報は不明として扱います。",
            "テストメンバーは管理者ではありません。account/portal/admin API はそれぞれの Cookie セッションを使い、任意の外部 Bearer では呼び出せません。"
          ],
          "code": "userinfo",
          "operations": [
            "oauth-userinfo"
          ]
        }
      ]
    },
    "logout": {
      "title": "ログアウト",
      "description": "アプリ内ログアウト、中央ログアウトと認可失効を区別します。",
      "sections": [
        {
          "id": "local",
          "title": "自分のセッションを終了する",
          "paragraphs": [
            "アプリのサーバーでセッションと Cookie を無効化します。localStorage の消去はサーバー側失効の代わりになりません。Moetopia ユーザーサイトのログアウトは自身の RP セッションに作用します。",
            "ログアウトだけでは OAuth 認可は失効しません。長期アクセスを終了する場合は別途 token を失効させます。"
          ]
        },
        {
          "id": "central",
          "title": "中央ログアウトを要求する",
          "paragraphs": [
            "Discovery の end_session_endpoint を使用します。この issuer の有効な id_token_hint または client_id を渡し、承認済み post_logout_redirect_uri と state を確認します。",
            "中央セッションの終了にはユーザーの明示確認が必要です。自動確認せず、全 RP・全端末の即時終了を保証しないでください。"
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
      "title": "エラーと調査",
      "description": "プロトコル、業務 API、結果不明の通信エラーを区別します。",
      "sections": [
        {
          "id": "protocol",
          "title": "OAuth/OIDC エラー",
          "paragraphs": [
            "error/error_description を使い、検証済みコールバックへリダイレクトされる場合があります。信頼できる回呼がない場合は直接エラーになります。state は必ず検証し、HTTP 200 だけで成功と判断しません。",
            "invalid_client は認証方式、invalid_grant は期限・消費状態・verifier・redirect_uri を確認します。access_denied は拒否、interaction_required は対話ログインが必要です。"
          ]
        },
        {
          "id": "business",
          "title": "サイト業務 API",
          "paragraphs": [
            "Problem Details に code やフィールドエラーを付加します。401 は再認証、403 は権限/MFA/Origin/CSRF、409 は revision 再読込と再確認、422 は入力修正、429 は Retry-After、503 は入力保持と再試行です。",
            "任意クエリーが空なら省略します。裸の ?cursor/?after は先頭ページ指定ではありません。応答の next_cursor/next_after が空なら最終ページです。"
          ]
        },
        {
          "id": "diagnose",
          "title": "安全な診断情報",
          "paragraphs": [
            "メソッド、エンドポイント、ステータス、機密でない request_id、クライアント種別と再現手順を記録します。Cookie、Authorization、パスワード、コード、verifier、秘密や本文全体を共有しません。",
            "Discovery、issuer、時計、コールバック、scope、アプリの審査・停止状態を確認します。PKCE や署名検証を無効にして回避しないでください。"
          ]
        }
      ]
    },
    "production": {
      "title": "本番公開チェック",
      "description": "実際の環境とアプリのバックエンドで受け入れ確認します。",
      "sections": [
        {
          "id": "configuration",
          "title": "環境と秘密",
          "paragraphs": [
            "本番 issuer、HTTPS コールバック、正しい Discovery を設定します。開発用 loopback、参考プロジェクトのドメインや認証情報をコピーしません。",
            "承認、有効化、停止状態を確認します。秘密を保護し、時計同期、依存先タイムアウト、制御された再試行を設定してログから認証情報を除外します。"
          ]
        },
        {
          "id": "acceptance",
          "title": "確認するシナリオ",
          "paragraphs": [
            "同意成功/拒否、不正 state/nonce/verifier、コールバック・コード再利用、期限切れ、並行刷新・旧値再利用、鍵更新、アプリ停止、アカウント無効化を確認します。単体テストは本番相互運用の検証ではありません。",
            "アプリ内終了、中央確認、失効とオフライン JWT の時間差を確認します。監視、バックアップ、復元、鍵切替の担当と手順を定めます。"
          ]
        },
        {
          "id": "limits",
          "title": "対応範囲",
          "paragraphs": [
            "動的クライアント登録、implicit、password grant、汎用マルチテナントは提供しません。ライブラリの機能と公開機能は別です。",
            "対象 Discovery、承認済み設定と本版 API を参照します。容量、本番 token 期限、実際の外部認証事業者は別途確認が必要です。"
          ]
        }
      ]
    }
  }
} satisfies DocumentationMessages
