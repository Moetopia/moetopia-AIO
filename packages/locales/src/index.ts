export type Locale = 'zh-CN' | 'en'
export type Portal = 'account' | 'admin' | 'developer'
export { accountMessages } from './account'
export { portalMessages } from './portal'
export { authorizationMessages } from './authorization'

export const messages = {
  'zh-CN': {
    tagline: '让每一次相遇，从同一个身份开始。',
    introduction: 'Moetopia 统一身份中心',
    account: '账号中心', admin: '管理控制台', developer: '开发者门户',
    accountDescription: '统一管理你的身份、安全设置与应用授权。',
    adminDescription: '管理生态账号、应用审核与平台安全。',
    developerDescription: '连接你的应用，构建属于 Moetopia 的生态。',
    phase: '暂未开放',
    notice: '此处功能还在准备中，请稍后再来。',
    connected: '服务正常', disconnected: '连接暂不可用', checking: '正在连接',
    serviceLabel: '服务状态', themeLabel: '切换主题', languageLabel: 'Switch to English',
    light: '浅色', dark: '深色', ecosystem: '一个身份，连接整个生态',
  },
  en: {
    tagline: 'Every connection begins with one identity.',
    introduction: 'Moetopia identity platform',
    account: 'Account center', admin: 'Administration', developer: 'Developer portal',
    accountDescription: 'Manage your identity, security settings and application access.',
    adminDescription: 'Manage ecosystem accounts, application reviews and platform security.',
    developerDescription: 'Connect your applications and build the Moetopia ecosystem.',
    phase: 'Not available yet',
    notice: 'This area is still being prepared. Please check back later.',
    connected: 'Service available', disconnected: 'Connection unavailable', checking: 'Connecting',
    serviceLabel: 'Service status', themeLabel: 'Change theme', languageLabel: '切换到中文',
    light: 'Light', dark: 'Dark', ecosystem: 'One identity, an entire ecosystem',
  },
} as const
