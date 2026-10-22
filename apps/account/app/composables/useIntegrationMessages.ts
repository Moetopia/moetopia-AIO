import { integrationMessages, adminMessages } from '@moetopia/locales'
export const useIntegrationMessages = () => {
  const { locale } = usePortalPreferences()
  return { text: computed(() => integrationMessages[locale.value]), common: computed(() => adminMessages[locale.value]) }
}
