import { messages, type Locale } from '@moetopia/locales'

export const usePortalPreferences = () => {
  const localeCookie = useCookie<Locale>('moetopia_locale', { default: () => 'zh-CN', sameSite: 'lax' })
  const themeCookie = useCookie<'light' | 'dark'>('moetopia_theme', { default: () => 'light', sameSite: 'lax' })
  const locale = computed<Locale>(() => localeCookie.value === 'en' ? 'en' : 'zh-CN')
  const theme = computed(() => themeCookie.value === 'dark' ? 'dark' : 'light')
  const text = computed(() => messages[locale.value])
  const toggleLocale = () => { localeCookie.value = locale.value === 'en' ? 'zh-CN' : 'en' }
  const toggleTheme = () => { themeCookie.value = theme.value === 'light' ? 'dark' : 'light' }
  return { locale, theme, text, toggleLocale, toggleTheme }
}
