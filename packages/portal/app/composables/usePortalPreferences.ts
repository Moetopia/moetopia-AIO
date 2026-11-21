import { isLocale, messages, resolveLocale, type Locale } from '@moetopia/locales'

export const usePortalPreferences = (syncTheme = false) => {
  const localeCookie = useCookie<Locale>('moetopia_locale', { default: () => 'zh-CN', sameSite: 'lax', path: '/' })
  const themeCookie = useCookie<'light' | 'dark' | 'system'>('moetopia_theme', { default: () => 'light', sameSite: 'lax', path: '/' })
  const systemDark = useState('moetopia-system-dark', () => import.meta.client ? window.matchMedia('(prefers-color-scheme: dark)').matches : false)
  const locale = computed<Locale>(() => resolveLocale(localeCookie.value))
  const themeChoice = computed(() => themeCookie.value === 'dark' || themeCookie.value === 'system' ? themeCookie.value : 'light')
  const theme = computed(() => themeChoice.value === 'system' ? systemDark.value ? 'dark' : 'light' : themeChoice.value)
  const text = computed(() => messages[locale.value])
  const setLocale = (value: string | undefined) => { if (isLocale(value)) localeCookie.value = value }
  const toggleTheme = () => { themeCookie.value = theme.value === 'light' ? 'dark' : 'light' }
  const setTheme = (value: 'light' | 'dark' | 'system') => { themeCookie.value = value }
  if (syncTheme && import.meta.client) {
    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const syncSystem = () => { systemDark.value = media.matches }
    syncSystem()
    watch(theme, value => { document.documentElement.setAttribute('data-theme', value) }, { immediate: true })
    onMounted(() => {
      syncSystem()
      media.addEventListener('change', syncSystem)
    })
    onBeforeUnmount(() => media.removeEventListener('change', syncSystem))
  }
  return { locale, theme, themeChoice, text, setLocale, toggleTheme, setTheme }
}
