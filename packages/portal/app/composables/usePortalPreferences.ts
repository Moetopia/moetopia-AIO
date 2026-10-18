import { isLocale, messages, resolveLocale, type Locale } from '@moetopia/locales'

export const usePortalPreferences = () => {
  const localeCookie = useCookie<Locale>('moetopia_locale', { default: () => 'zh-CN', sameSite: 'lax' })
  const themeCookie = useCookie<'light' | 'dark'>('moetopia_theme', { default: () => 'light', sameSite: 'lax' })
  const locale = computed<Locale>(() => resolveLocale(localeCookie.value))
  const theme = computed(() => themeCookie.value === 'dark' ? 'dark' : 'light')
  const text = computed(() => messages[locale.value])
  const setLocale = (value: string | undefined) => { if (isLocale(value)) localeCookie.value = value }
  const toggleTheme = () => { themeCookie.value = theme.value === 'light' ? 'dark' : 'light' }
  return { locale, theme, text, setLocale, toggleTheme }
}
