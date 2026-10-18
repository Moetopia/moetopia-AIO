export const authorizationMessages = {
  'zh-CN': {
    title: '确认应用访问', description: '检查应用请求的权限，再决定是否继续。',
    loading: '正在读取授权请求', application: '申请访问的应用', client: '应用标识', callback: '返回地址', audience: '访问的服务', expires: '请求到期时间',
    permissions: '请求的权限', openid: '确认你的账号身份', profile: '读取昵称等基本资料', email: '读取邮箱地址', otherScope: '应用请求的其他权限',
    account: '使用此账号继续', approve: '允许访问', deny: '拒绝', switchAccount: '换个账号', login: '请登录后继续', reauthenticate: '请重新验证账号',
    loginDescription: '登录后会再次展示授权详情；登录本身不会批准应用访问。',
    unavailable: '授权服务暂不可用，请稍后重试。', expired: '此请求已失效。请回到发起登录的应用，重新开始。',
    forbidden: '无法验证此次操作。请重新读取请求；仍失败时从原应用重新开始。', changed: '账号已变化，请重新核对下方账号与权限，再作决定。',
    invalid: '授权请求格式无效。请从原应用重新发起登录。', limited: '操作过于频繁，请稍后重试。', retry: '重新读取请求',
    submitting: '正在处理你的选择', redirecting: '正在返回应用', noPermissions: '此请求没有额外权限。',
  },
  en: {
    title: 'Review application access', description: 'Check the requested permissions before continuing.',
    loading: 'Loading authorization request', application: 'Application requesting access', client: 'Application ID', callback: 'Return address', audience: 'Target services', expires: 'Request expires',
    permissions: 'Requested permissions', openid: 'Confirm your account identity', profile: 'Read your name and basic profile', email: 'Read your email address', otherScope: 'Additional permission requested by this application',
    account: 'Continue with this account', approve: 'Allow access', deny: 'Deny', switchAccount: 'Use another account', login: 'Sign in to continue', reauthenticate: 'Verify your account again',
    loginDescription: 'You will review this request again after signing in. Signing in does not grant access.',
    unavailable: 'Authorization is temporarily unavailable. Please try again later.', expired: 'This request is no longer available. Return to the application and start sign-in again.',
    forbidden: 'This action could not be verified. Reload the request, or restart sign-in from the application if it still fails.', changed: 'The account has changed. Review the account and permissions again before deciding.',
    invalid: 'This authorization request is invalid. Restart sign-in from the application.', limited: 'Too many requests. Please try again later.', retry: 'Reload request',
    submitting: 'Processing your choice', redirecting: 'Returning to the application', noPermissions: 'No additional permissions requested.',
  },
} as const
