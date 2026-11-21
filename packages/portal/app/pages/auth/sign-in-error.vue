<script setup lang="ts">
import { MoeBrandMark, MoeLink, MoeStatusPage } from '@moetopia/ui'

const messages = {
  'zh-CN': { title: '登录未完成', expired: '这次登录请求已失效。请从这里重新登录。', denied: '登录授权已取消。你可以重新发起登录。', unavailable: '登录服务暂时不可用，请稍后重试。', failed: '无法确认这次登录。请重新登录。', retry: '重新登录' },
  'zh-TW': { title: '登入未完成', expired: '這次登入請求已失效。請從這裡重新登入。', denied: '登入授權已取消。你可以重新發起登入。', unavailable: '登入服務暫時無法使用，請稍後再試。', failed: '無法確認這次登入。請重新登入。', retry: '重新登入' },
  en: { title: 'Sign-in could not be completed', expired: 'This sign-in request is no longer valid. Start a new sign-in.', denied: 'Sign-in authorization was cancelled. You can start again.', unavailable: 'Sign-in is temporarily unavailable. Try again in a moment.', failed: 'We could not verify this sign-in. Start a new sign-in.', retry: 'Sign in again' },
  ja: { title: 'ログインを完了できませんでした', expired: 'このログインリクエストは無効です。もう一度ログインしてください。', denied: 'ログインの許可が取り消されました。もう一度ログインできます。', unavailable: '現在ログインできません。しばらくしてからお試しください。', failed: 'ログインを確認できませんでした。もう一度ログインしてください。', retry: 'もう一度ログイン' },
}
const route = useRoute()
const { locale } = usePortalPreferences()
const text = computed(() => messages[locale.value])
const reason = computed(() => {
  const value = route.query.reason
  return value === 'expired' || value === 'denied' || value === 'unavailable' ? value : 'failed'
})
const retry = computed(() => route.query.site === 'admin' ? '/api/v1/admin/login' : '/api/v1/portal/login')
useHead(() => ({ title: text.value.title }))
</script>

<template>
  <MoeStatusPage :code="reason === 'unavailable' ? '503' : '400'" :title="text.title" :description="text[reason]" :illustration="reason === 'unavailable' ? 'service-unavailable' : 'access-restricted'">
    <template #header><MoeLink href="/" class="moe-brand-lockup"><MoeBrandMark />Moetopia</MoeLink><PortalPreferences /></template>
    <a :href="retry">{{ text.retry }}</a>
  </MoeStatusPage>
</template>
