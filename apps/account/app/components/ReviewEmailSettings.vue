<script setup lang="ts">
import type { ProfileResponse, paths } from '@moetopia/api-client'
import { MoeAlert, MoeButton, MoeCard, MoeCheckbox } from '@moetopia/ui'

type Preference = paths['/api/v1/portal/notification-preferences']['put']['requestBody']['content']['application/json']

const messages = {
  'zh-CN': { title: '邮件通知', label: '应用审核进度', hint: '提交审核、通过或拒绝时发送邮件。验证码和安全提醒持续发送。', save: '保存' },
  'zh-TW': { title: '郵件通知', label: '應用審核進度', hint: '送出審核、通過或拒絕時寄送郵件。驗證碼及安全通知會繼續寄送。', save: '儲存' },
  en: { title: 'Email notifications', label: 'Application review updates', hint: 'Get an email when a review is submitted, approved, or rejected. Verification and security emails stay enabled.', save: 'Save' },
  ja: { title: 'メール通知', label: 'アプリの審査状況', hint: '審査の申請・承認・却下をメールで通知します。確認コードとセキュリティ通知は引き続き送信されます。', save: '保存' },
}
const { locale } = usePortalPreferences()
const copy = computed(() => messages[locale.value])
const { context } = usePortalContext()
const { busy, error, success, headers, run } = useManagementRequest()
const saved = computed(() => !context.value?.user?.mute_application_review_email)
const enabled = ref(saved.value)
watch(saved, value => { enabled.value = value })
const save = () => run(async () => {
  const user = context.value?.user
  if (!user) return
  const body: Preference = { muted: !enabled.value, revision: user.revision ?? 0 }
  const updated = await $fetch<ProfileResponse>('/api/v1/portal/notification-preferences', { method: 'PUT', headers: headers.value, body })
  if (context.value?.user?.id === user.id) context.value = { ...context.value, user: updated }
})
</script>

<template>
  <MoeCard layout="form" :title="copy.title">
    <form class="grid gap-4" @submit.prevent="save">
      <MoeCheckbox v-model="enabled" :label="copy.label" :description="copy.hint" :disabled="busy" />
      <MoeAlert v-if="error" :title="error" color="danger" announce />
      <MoeAlert v-else-if="success" :title="{ 'zh-CN': '已保存', 'zh-TW': '已儲存', en: 'Saved', ja: '保存しました' }[locale]" color="success" announce />
      <div><MoeButton type="submit" :loading="busy" :disabled="enabled === saved">{{ copy.save }}</MoeButton></div>
    </form>
  </MoeCard>
</template>
