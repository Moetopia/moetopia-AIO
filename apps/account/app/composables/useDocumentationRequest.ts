import type { DocDetail, DocSchema } from '../../shared/documentation'
export const useDocumentationRequest = (detail: ComputedRef<DocDetail | null | undefined>) => {
  const { docs } = useDocumentation()
  const values = ref<Record<string, string>>({})
  const json = ref('{}')
  const bearer = ref('')
  const verifier = ref('')
  const response = ref<string>()
  const status = ref(0)
  const elapsed = ref<number>()
  const error = ref('')
  const busy = ref(false)
  let controller: AbortController | undefined
  let generation = 0
  const entry = computed(() => detail.value?.entry)
  const blocked = computed(() => entry.value?.path.startsWith('/api/v1/admin/') ? docs.value.labels.adminOnly : entry.value?.path === '/oauth/introspect' ? docs.value.labels.serverOnly : '')
  const navigation = computed(() => ['/oauth/authorize', '/oauth/logout'].includes(entry.value?.path ?? ''))
  const publicToken = computed(() => ['/oauth/token', '/oauth/revoke'].includes(entry.value?.path ?? ''))
  const bodyType = computed(() => Object.keys(detail.value?.operation.requestBody?.content ?? {})[0])
  const bodySchema = computed(() => {
    const schema = detail.value?.operation.requestBody?.content?.[bodyType.value ?? '']?.schema
    return schema?.$ref ? detail.value?.schemas[schema.$ref.split('/').at(-1) ?? ''] : schema
  })
  const fields = computed(() => [
    ...(detail.value?.operation.parameters ?? []).filter(field => field.in === 'path' || field.in === 'query').map(field => ({ key: `${field.in}:${field.name}`, name: field.name, location: field.in!, required: !!field.required, schema: field.schema ?? {} as DocSchema })),
    ...(bodyType.value === 'application/x-www-form-urlencoded' ? Object.entries(bodySchema.value?.properties ?? {}).filter(([name]) => name !== 'client_secret').map(([name, schema]) => ({ key: `body:${name}`, name, location: 'body', required: !!bodySchema.value?.required?.includes(name), schema })) : []),
  ])
  const clear = () => {
    generation++; controller?.abort(); controller = undefined
    values.value = {}; json.value = '{}'; bearer.value = ''; verifier.value = ''; response.value = undefined; status.value = 0; elapsed.value = undefined; error.value = ''; busy.value = false
    if (entry.value?.path === '/oauth/token') values.value['body:grant_type'] = 'authorization_code'
    if (entry.value?.path === '/oauth/authorize') { values.value['query:response_type'] = 'code'; values.value['query:scope'] = 'openid profile'; values.value['query:code_challenge_method'] = 'S256' }
  }
  const random = () => {
    const bytes = crypto.getRandomValues(new Uint8Array(32))
    return btoa(String.fromCharCode(...bytes)).replaceAll('+', '-').replaceAll('/', '_').replaceAll('=', '')
  }
  const pkce = async () => {
    const current = generation
    try {
      const secret = random()
      const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(secret))
      if (current !== generation) return
      verifier.value = secret
      Object.assign(values.value, { 'query:state': random(), 'query:nonce': random(), 'query:code_challenge': btoa(String.fromCharCode(...new Uint8Array(digest))).replaceAll('+', '-').replaceAll('/', '_').replaceAll('=', ''), 'query:code_challenge_method': 'S256' })
    } catch { if (current === generation) error.value = docs.value.labels.testFailure }
  }
  const run = async () => {
    if (!detail.value || busy.value || blocked.value) return
    error.value = ''; response.value = undefined; status.value = 0; elapsed.value = undefined
    const operation = detail.value
    let path = operation.entry.path
    const query = new URLSearchParams()
    const form = new URLSearchParams()
    for (const field of fields.value) {
      const value = values.value[field.key] ?? ''
      if (field.required && !value || field.location === 'path' && (!value || value === '.' || value === '..' || /[\/\\]/.test(value))) { error.value = docs.value.labels.invalidTest; return }
      if (field.location === 'path') path = path.replace(`{${field.name}}`, encodeURIComponent(value))
      else if (value) (field.location === 'body' ? form : query).set(field.name, value)
    }
    if (publicToken.value && !form.get('client_id')) { error.value = docs.value.labels.invalidTest; return }
    let body: string | undefined
    if (!['GET', 'HEAD'].includes(operation.entry.method)) {
      if (bodyType.value === 'application/json') { try { body = JSON.stringify(JSON.parse(json.value)) } catch { error.value = docs.value.labels.invalidTest; return } }
      else if (bodyType.value === 'application/x-www-form-urlencoded') body = form.toString()
    }
    const target = path + (query.size ? `?${query}` : '')
    if (navigation.value) {
      if (operation.entry.method === 'POST') {
        const element = document.createElement('form')
        element.method = 'POST'; element.action = target; element.target = '_blank'; element.rel = 'noopener noreferrer'
        for (const [name, value] of form) { const input = document.createElement('input'); input.type = 'hidden'; input.name = name; input.value = value; element.append(input) }
        document.body.append(element)
        try { element.submit() } finally { element.remove() }
      } else window.open(target, '_blank', 'noopener,noreferrer')
      error.value = docs.value.labels.redirectTest; return
    }
    const request = new AbortController()
    controller = request; busy.value = true
    const current = ++generation
    const started = performance.now()
    const timer = setTimeout(() => request.abort(), 15000)
    try {
      const headers: Record<string, string> = {}
      const business = path.startsWith('/api/v1/')
      if (body !== undefined && bodyType.value) headers['Content-Type'] = bodyType.value
      if (path === '/oauth/userinfo' && bearer.value) headers.Authorization = `Bearer ${bearer.value}`
      if (business && !['GET', 'HEAD'].includes(operation.entry.method)) {
        const segment = path.split('/')[3]
        const prefix = segment === 'developer' ? 'portal' : segment
        if (prefix === 'portal' || prefix === 'account') {
          const context = await $fetch<{ csrf_token: string }>(`/api/v1/${prefix}/context`, { retry: 0, signal: request.signal })
          headers['X-Moetopia-CSRF'] = context.csrf_token
        }
      }
      const result = await fetch(target, { method: operation.entry.method, headers, body, signal: request.signal, credentials: business ? 'same-origin' : 'omit', redirect: 'manual', cache: 'no-store' })
      if (result.type === 'opaqueredirect') throw new Error('Navigation required')
      const reader = result.body?.getReader()
      const decoder = new TextDecoder()
      let text = '', bytes = 0
      if (reader) {
        try { while (true) { const chunk = await reader.read(); if (chunk.done) break; bytes += chunk.value.byteLength; if (bytes > 262144) { await reader.cancel(); throw new Error('Response limit exceeded') } text += decoder.decode(chunk.value, { stream: true }) } text += decoder.decode() } finally { reader.releaseLock() }
      }
      if (generation !== current) return
      status.value = result.status; elapsed.value = Math.round(performance.now() - started)
      try { response.value = text ? JSON.stringify(JSON.parse(text), null, 2) : '' } catch { response.value = text }
    } catch { if (generation === current) error.value = docs.value.labels.testFailure }
    finally { clearTimeout(timer); if (generation === current) { busy.value = false; controller = undefined } }
  }
  watch(() => entry.value?.id, clear, { immediate: true })
  const restored = (event: PageTransitionEvent) => { if (event.persisted) clear() }
  onMounted(() => { window.addEventListener('pagehide', clear); window.addEventListener('pageshow', restored) })
  onBeforeUnmount(() => { window.removeEventListener('pagehide', clear); window.removeEventListener('pageshow', restored); clear() })
  return { values, json, bearer, verifier, response, status, elapsed, error, busy, fields, blocked, navigation, bodyType, run, clear, pkce, cancel: () => controller?.abort() }
}
