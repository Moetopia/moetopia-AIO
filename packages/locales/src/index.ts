import zhCN from './lang/zh-CN'
import zhTW from './lang/zh-TW'
import en from './lang/en'
import ja from './lang/ja'
import managementCN from './management/zh-CN'
import managementTW from './management/zh-TW'
import managementEN from './management/en'
import managementJA from './management/ja'
import type { Locale, LocaleMessages } from './types'

export type { Locale, LocaleMessages, Portal } from './types'
export const supportedLocales = [
  { code: 'zh-CN', label: '简体中文', shortLabel: '简中' },
  { code: 'zh-TW', label: '繁體中文', shortLabel: '繁中' },
  { code: 'en', label: 'English', shortLabel: 'EN' },
  { code: 'ja', label: '日本語', shortLabel: '日本語' },
] as const
export const isLocale = (value: unknown): value is Locale => supportedLocales.some(locale => locale.code === value)
export const resolveLocale = (value: unknown): Locale => isLocale(value) ? value : 'zh-CN'
export const emailLocale = (locale: Locale): 'zh-CN' | 'en' => locale.startsWith('zh') ? 'zh-CN' : 'en'
export const translations = { 'zh-CN': zhCN, 'zh-TW': zhTW, en, ja } as const satisfies Record<Locale, LocaleMessages>
const pick = <K extends keyof LocaleMessages>(group: K): Record<Locale, LocaleMessages[K]> => ({
  'zh-CN': zhCN[group], 'zh-TW': zhTW[group], en: en[group], ja: ja[group],
})
export const messages = pick('common')
export const accountMessages = pick('account')
export const portalMessages = pick('portal')
export const authorizationMessages = pick('authorization')
export const managementMessages = { 'zh-CN': managementCN, 'zh-TW': managementTW, en: managementEN, ja: managementJA } satisfies Record<Locale, Record<keyof typeof managementCN, string>>
