<script setup lang="ts">
import { MoeAlert, MoeButton, MoeDialog } from '@moetopia/ui'
import { managementMessages } from '@moetopia/locales'

const props = defineProps<{ file: File }>()
const emit = defineEmits<{ cancel: []; confirm: [base64: string] }>()
const { locale } = usePortalPreferences()
const text = computed(() => managementMessages[locale.value])
const canvas = ref<HTMLCanvasElement>()
const image = shallowRef<HTMLImageElement>()
const failed = ref(false)
const zoom = ref(1)
const horizontal = ref(50)
const vertical = ref(50)
const size = 256

onMounted(async () => {
  const source = URL.createObjectURL(props.file)
  try {
    const decoded = new Image()
    decoded.src = source
    await decoded.decode()
    image.value = decoded
  } catch {
    failed.value = true
  } finally {
    URL.revokeObjectURL(source)
  }
})

watchEffect(() => {
  const picture = image.value
  const context = canvas.value?.getContext('2d')
  if (!picture || !context) return
  const side = Math.min(picture.naturalWidth, picture.naturalHeight) / zoom.value
  const x = (picture.naturalWidth - side) * horizontal.value / 100
  const y = (picture.naturalHeight - side) * vertical.value / 100
  context.clearRect(0, 0, size, size)
  context.drawImage(picture, x, y, side, side, 0, 0, size, size)
}, { flush: 'post' })

function confirm() {
  if (!image.value || !canvas.value) return
  try {
    const data = canvas.value.toDataURL('image/png').split(',')[1]
    if (!data || !canvas.value.getContext('2d')) throw new Error()
    emit('confirm', data)
  } catch {
    failed.value = true
  }
}
</script>

<template>
  <MoeDialog :open="true" :title="text.avatarCrop" :close-label="text.close" :show-close-action="false" @update:open="open => { if (!open) emit('cancel') }">
    <div class="grid gap-4">
      <canvas v-show="image && !failed" ref="canvas" :width="size" :height="size" class="mx-auto max-w-full rounded-xl bg-surface-raised" role="img" :aria-label="text.avatarCrop" />
      <MoeAlert v-if="failed" :title="text.invalid" color="danger" announce />
      <fieldset v-else class="grid gap-3" :disabled="!image">
        <label class="grid gap-1">{{ text.avatarZoom }}<input v-model.number="zoom" class="w-full accent-primary" type="range" min="1" max="4" step="0.01" /></label>
        <label class="grid gap-1">{{ text.avatarHorizontal }}<input v-model.number="horizontal" class="w-full accent-primary" type="range" min="0" max="100" :disabled="zoom === 1 && !!image && image.naturalWidth <= image.naturalHeight" /></label>
        <label class="grid gap-1">{{ text.avatarVertical }}<input v-model.number="vertical" class="w-full accent-primary" type="range" min="0" max="100" :disabled="zoom === 1 && !!image && image.naturalHeight <= image.naturalWidth" /></label>
      </fieldset>
      <div class="flex justify-end gap-3">
        <MoeButton variant="light" @click="emit('cancel')">{{ text.close }}</MoeButton>
        <MoeButton :disabled="!image || failed" @click="confirm">{{ text.avatarUpload }}</MoeButton>
      </div>
    </div>
  </MoeDialog>
</template>
