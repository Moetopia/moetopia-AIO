import { documentationMessages, portalMessages } from '@moetopia/locales'
export const useDocumentation = () => {
  const { locale } = usePortalPreferences()
  return { locale, docs: computed(() => documentationMessages[locale.value]), common: computed(() => portalMessages[locale.value]) }
}
