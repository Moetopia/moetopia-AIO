import source from '@moetopia/api-client/openapi.json'
import type { DocEntry, DocOperation, DocSchema, DocDetail } from '../../shared/documentation'

export const documentation = source
const methods = new Set(['get', 'post', 'put', 'patch', 'delete', 'head', 'options'])
const operations = new Map<string, { entry: DocEntry; operation: DocOperation }>()
for (const [path, item] of Object.entries(source.paths)) {
  for (const [method, value] of Object.entries(item)) {
    if (!methods.has(method)) continue
    const operation = value as DocOperation
    const entry = { id: operation.operationId, method: method.toUpperCase(), path, title: operation.summary ?? operation.operationId, group: operation.tags?.[0] ?? 'API' }
    if (operations.has(entry.id)) throw new Error('Duplicate documentation operation identifier')
    operations.set(entry.id, { entry, operation })
  }
}
export const documentationCatalog = [...operations.values()].map(value => value.entry)
export const documentationDetail = (id: string): DocDetail | undefined => {
  const item = operations.get(id)
  if (!item) return
  const schemas: Record<string, DocSchema> = {}
  const available: Record<string, DocSchema> = source.components.schemas
  const visit = (value: unknown) => {
    if (!value || typeof value !== 'object') return
    if ('$ref' in value && typeof value.$ref === 'string' && value.$ref.startsWith('#/components/schemas/')) {
      const name = value.$ref.slice('#/components/schemas/'.length)
      if (!schemas[name] && available[name]) { schemas[name] = available[name]; visit(available[name]) }
    }
    Object.values(value).forEach(visit)
  }
  visit(item.operation)
  return { ...item, schemas, securitySchemes: source.components.securitySchemes }
}
