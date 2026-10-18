import type zhCN from './lang/zh-CN'

export type Locale = 'zh-CN' | 'zh-TW' | 'en' | 'ja'
export type Portal = 'account' | 'admin' | 'developer'
export type LocaleMessages = { [Group in keyof typeof zhCN]: { [Key in keyof typeof zhCN[Group]]: string } }
