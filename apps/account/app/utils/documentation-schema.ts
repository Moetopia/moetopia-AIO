import type { MoeSchemaField } from '@moetopia/ui'
import type { DocSchema } from '../../shared/documentation'
export const schemaFields = (input: DocSchema | undefined, schemas: Record<string, DocSchema>, ancestors: string[] = []): MoeSchemaField[] => {
  if (!input) return []
  const reference = input.$ref?.split('/').at(-1)
  if (reference && ancestors.includes(reference)) return []
  const schema = reference ? { ...schemas[reference], ...input } : input
  const visited = reference ? [...ancestors, reference] : ancestors
  if (visited.length > 8) return []
  const field = (name: string, value: DocSchema, required = false): MoeSchemaField => {
    const target = value.$ref?.split('/').at(-1)
    const resolved = target ? { ...schemas[target], ...value } : value
    const type = Array.isArray(resolved.type) ? resolved.type.join(' | ') : resolved.type ?? (resolved.properties ? 'object' : target ?? (resolved.oneOf ? 'oneOf' : resolved.anyOf ? 'anyOf' : 'unknown'))
    const constraints = Object.entries(resolved).filter(([key]) => ['format', 'enum', 'const', 'default', 'nullable', 'minimum', 'maximum', 'minLength', 'maxLength', 'minItems', 'maxItems', 'pattern', 'readOnly', 'writeOnly', 'additionalProperties'].includes(key)).map(([key, value]) => `${key}: ${JSON.stringify(value)}`)
    const nested = resolved.properties || resolved.items || resolved.allOf || resolved.oneOf || resolved.anyOf
    return { name, type, required, description: resolved.description, constraints, children: nested ? schemaFields(value, schemas, visited) : [] }
  }
  if (schema.type && !schema.properties && !schema.items && !schema.allOf && !schema.oneOf && !schema.anyOf && schema.type !== 'object') return [field('$', schema)]
  return [
    ...Object.entries(schema.properties ?? {}).map(([name, value]) => field(name, value, schema.required?.includes(name))),
    ...(schema.items ? [field('[]', schema.items)] : []),
    ...(['allOf', 'oneOf', 'anyOf'] as const).flatMap(kind => schema[kind]?.map((value, index) => ({ ...field(`${kind} ${index + 1}`, value), constraints: [...(value.required?.length ? [`required: ${value.required.join(', ')}`] : [])] })) ?? []),
  ]
}
