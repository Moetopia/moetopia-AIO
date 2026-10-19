import type { LocaleMessages } from '../types'

export default {
  common: {
    tagline: 'ひとつのアカウントから、新しい出会いへ。', introduction: 'Moetopia アカウントサービス', account: 'アカウント', admin: '管理画面', developer: '開発者向け',
    accountDescription: 'プロフィール、セキュリティ、アプリのアクセス権を管理します。', adminDescription: 'アカウント、アプリの審査、プラットフォームの安全性を管理します。', developerDescription: 'アプリを Moetopia に接続しましょう。',
    phase: '準備中', notice: 'この機能は準備中です。公開までお待ちください。', connected: 'サービス正常', disconnected: '接続できません', checking: '接続中', serviceLabel: 'サービスの状態', themeLabel: 'テーマを切り替え', languageLabel: '言語', pageLoading: 'ページを読み込み中', light: 'ライト', dark: 'ダーク', ecosystem: 'ひとつのアカウントで、つながる世界',
  },
  account: {
    welcome: 'おかえりなさい', subtitle: 'Moetopia アカウントにログインします。', namePlaceholder: '表示名を入力', invitationPlaceholder: '招待コードを入力', passwordPlaceholder: 'パスワードを入力', codePlaceholder: '8 桁の確認コード', registerSubtitle: 'メールアドレスを確認して、Moetopia アカウントを作成します。',
    login: 'ログイン', register: 'アカウントを作成', email: 'メールアドレス', password: 'パスワード', name: '表示名', invitation: '招待コード', invitationHint: '現在、登録には招待コードが必要です。各コードは一度だけ使えます。', openHint: '登録する前にメールアドレスの確認が必要です。', passwordHint: '15 文字以上で入力してください。空白も使用できます。',
    code: 'メール確認コード', sendCode: 'コードを送信', resend: '再送信', codeHint: 'コードの有効期限は 10 分です。', showPassword: 'パスワードを表示', hidePassword: 'パスワードを隠す', continue: '確認してアカウントを作成', checking: 'アカウントの状態を確認中…', retry: '再接続', signedIn: 'アカウント', welcomeBack: 'おかえりなさい、',
    logout: 'ログアウト', sessions: 'ログインセッション', current: '現在のセッション', revoke: 'セッションを終了', sessionHint: '最近のアカウントセッションです。終了すると再ログインが必要になります。', expires: '有効期限', created: 'ログイン日時', providerHint: 'GitHub / Google ログインは今後公開予定です。',
    sent: '登録可能なメールアドレスには確認コードが送信されます。繰り返し送信しないでください。', registered: 'アカウントを作成しました。メールアドレスとパスワードでログインしてください。', invalidCredentials: 'メールアドレスまたはパスワードが正しくありません。', invalidRegistration: '登録情報が無効か期限切れです。確認コードと招待コードをご確認ください。',
    unavailable: 'アカウントサービスに接続できません。しばらくしてから再試行してください。', limited: '試行回数が多すぎます。しばらくしてから再試行してください。', forbidden: 'リクエストの確認情報が無効になりました。再接続してお試しください。', sessionExpired: 'セッションの有効期限が切れました。再ログインしてください。', sessionListFailed: 'セッションを取得できません。再試行してください。',
    noAccount: '初めてご利用ですか？', haveAccount: 'アカウントをお持ちですか？', registerLink: '新規登録', loginLink: 'ログインする',
  },
  portal: {
    "console": "コンソール", "createApplication": "アプリを作成", "createUnavailable": "アプリの作成機能は準備中です。", "emailSettings": "メールアドレス", "emailSettingsDescription": "アカウントのメールを確認します。変更機能は準備中です。", "saveChanges": "変更を保存", "profileReadOnly": "プロフィールの編集は準備中です。現在は閲覧のみ可能です。",
    collapseNavigation: "サイドバーを折りたたむ", expandNavigation: "サイドバーを展開",
    brandDetail: 'アカウント、コミュニティ、創作', yourAccount: 'あなたのアカウント', identityDescription: 'このアカウントで Moetopia と接続済みのアプリにログインできます。', developerStart: '自分のアプリを見る', developerDocs: '接続ガイドを読む', developerIntro: 'あなたのアプリを Moetopia に。', developerIntroDescription: 'Moetopia アカウントでサイトにログインできるようにしましょう。接続の流れを確認して、アプリの情報と必要な権限を準備します。',
    integrationTitle: '接続の流れ', integrationDescription: 'アプリの用途から利用者の同意まで、それぞれの役割を明確にします。', applicationReview: 'アプリを審査に提出', applicationReviewDescription: '正式な接続には審査が必要です。新しい設定の審査中も、承認済みの設定は有効です。', oauthSupport: '対応するプロトコル', oauthSupportDescription: '認可コードと PKCE S256、OIDC Discovery、UserInfo に対応しています。正式なアプリ登録と審査は準備中です。',
    docsTitle: '接続ガイド', docsDescription: 'ログイン、アプリの設定、認可の役割を確認します。', signInNeeded: 'Moetopia にログインしてください', signInNeededDescription: 'アプリ管理には現在のアカウントを使用します。作成、編集、審査への提出はまだ利用できません。', accountSignIn: 'ログインへ', backToDeveloper: '開発者ページに戻る',
    skip: 'メインコンテンツへ', navigation: 'メインナビゲーション', openNavigation: 'メニューを開く', closeNavigation: 'メニューを閉じる', workspace: 'ワークスペース', identity: '共通アカウント', overview: '概要', profile: 'プロフィール', sessions: 'セッション', security: 'セキュリティ', applications: '自分のアプリ', preparation: '接続準備', users: 'アカウント管理', review: 'アプリ審査',
    artCaption: 'MOETOPIA · ACCOUNT', artTitle: 'ここから、あなたの世界へ。', artDescription: 'ひとつのアカウントで、好きなコミュニティと創作につながる。', developerEyebrow: 'DEVELOPER WORKSPACE', adminEyebrow: 'PLATFORM WORKSPACE', developerTitle: '開発者向け', adminTitle: '管理画面', developerDescription: '初めての接続に向けて、アプリを準備しましょう。', adminDescription: 'アカウント、アプリ、プラットフォームの安全性を管理します。',
    development: '準備中', developmentDescription: '関連する機能はまだ公開されていません。概要と接続準備をご確認いただけます。', myApps: '自分のアプリ', myAppsDescription: 'アプリの設定、審査、認証情報をここで管理します。', appsEmpty: 'アプリの接続は準備中です', appsEmptyDescription: 'アプリ名、コールバック URL、必要な権限を準備してください。機能の公開後にアプリを作成できます。',
    adminArea: 'プラットフォーム管理', adminAreaDescription: '管理権限、審査、安全性の監査には専用のアクセス制御を使用します。', adminEmpty: '管理機能は準備中です', adminEmptyDescription: '公開後にアカウント、アプリ審査、安全性を管理できます。', managementTitle: '管理項目', prepareTitle: '接続前の準備', prepareDescription: 'アプリの用途、境界、権限を整理しておきましょう。',
    appIdentity: 'アプリの名前と説明', appIdentityDescription: '利用者が分かる名前、アイコン、用途の説明を準備します。', callbacks: 'コールバック URL', callbacksDescription: '開発用と本番用を分け、意図しない転送を許可しないようにします。', permissions: '必要な権限', permissionsDescription: '必要最小限の権限だけを要求し、その用途を利用者に説明します。',
    accountsDescription: 'アカウントを確認し、登録方式や招待を管理します。', reviewDescription: '重要なアプリ設定はバージョンごとに審査します。審査機能は今後公開予定です。', securityDescription: '管理者の MFA、リカバリーコード、監査は管理機能公開前の必須項目です。', planned: '公開予定', profileDescription: 'Moetopia のプロフィールとアカウントの安全性を管理します。', sessionCount: 'セッション数', sessionCountUnknown: '取得できません',
    profileTitle: '基本情報', profileCardDescription: '現在ログインしているアカウントの情報です。', sessionCardDescription: 'アカウントのセッションを確認し、使わなくなった接続を終了できます。', emailVerified: 'メール確認済み', securityLater: '追加のセキュリティ設定は準備中です', securityLaterDescription: 'パスワード変更、外部アカウント連携、アプリのアクセス管理は準備中です。現在はセッションを管理できます。',
    sessionTitle: 'アカウントセッション', emptySessions: '表示できるセッションがありません', sessionTime: '日時', providerLater: 'ログイン方法を追加予定です', viewSessions: 'ログインセッションを管理', securityOverview: '現在利用できるセキュリティ操作と今後の機能を確認します。', passwordChangesLater: 'パスワード変更は準備中', accountAccess: 'アカウントへのアクセス', passwordAccess: 'メールとパスワード', passwordAccessDescription: 'メールとパスワードでログインできます。パスワード変更やアカウント復旧は準備中です。',
    providerAccess: '外部アカウントの連携', providerAccessDescription: 'GitHub と Google のログイン・連携機能は準備中です。', accessManagement: 'アプリのアクセス権', accessManagementDescription: 'アプリのアクセス権の確認や取り消しは今後提供予定です。',
  },
  authorization: {
    "sharedInformation": "共有するアカウント情報", "displayName": "表示名", "emailAddress": "メールアドレス", "subject": "アカウント ID", "credentialNotice": "パスワードやログイン認証情報は共有されません。", "technicalDetails": "リクエストの詳細", "returnNotice": "認可後の戻り先", "requestAccess": "がアカウントへのアクセスを要求しています",
    title: 'アプリへのアクセスを確認', description: '要求されている権限を確認してから、続行するか選択してください。', loading: '認可リクエストを読み込み中', application: 'アクセスを要求しているアプリ', client: 'アプリ ID', callback: '戻り先 URL', audience: '対象サービス', expires: 'リクエストの有効期限', permissions: '要求する権限',
    openid: 'アカウントの本人確認', profile: '表示名などの基本情報を取得', email: 'メールアドレスを取得', otherScope: 'アプリが要求するその他の権限', offlineAccess: '認可の期限切れや取り消しまで、オフライン時も許可済みの情報にアクセスします。', account: 'このアカウントで続行', approve: 'アクセスを許可', deny: '拒否', switchAccount: '別のアカウントを使う', login: 'ログインして続行してください', reauthenticate: 'アカウントを再確認してください', loginDescription: 'ログイン後に認可の詳細を再表示します。ログインだけではアクセスは許可されません。',
    unavailable: '認可サービスを利用できません。しばらくしてから再試行してください。', expired: 'このリクエストは無効です。元のアプリに戻り、ログインをやり直してください。', forbidden: '操作を確認できません。再読み込みしても失敗する場合は、元のアプリからやり直してください。', changed: 'アカウントが変更されました。下のアカウントと権限を再確認してから選択してください。', invalid: '認可リクエストの形式が無効です。元のアプリからログインをやり直してください。', limited: '操作回数が多すぎます。しばらくしてから再試行してください。', retry: 'リクエストを再読み込み', submitting: '選択を処理中', redirecting: 'アプリに戻っています', noPermissions: '追加の権限は要求されていません。',
  },
} as const satisfies LocaleMessages
