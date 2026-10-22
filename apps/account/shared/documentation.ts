export interface DocSchema {
  $ref?: string; type?: string | string[]; description?: string; format?: string
  properties?: Record<string, DocSchema>; required?: string[]; items?: DocSchema
  enum?: unknown[]; const?: unknown; default?: unknown; nullable?: boolean
  minimum?: number; maximum?: number; minLength?: number; maxLength?: number
  minItems?: number; maxItems?: number; pattern?: string; readOnly?: boolean; writeOnly?: boolean
  oneOf?: DocSchema[]; anyOf?: DocSchema[]; allOf?: DocSchema[]; additionalProperties?: boolean | DocSchema
}
export interface DocParameter { name: string; in?: string; required?: boolean; description?: string; schema?: DocSchema }
export interface DocContent { schema?: DocSchema }
export interface DocOperation {
  operationId: string; summary?: string; description?: string; tags?: string[]
  parameters?: DocParameter[]; deprecated?: boolean
  requestBody?: { required?: boolean; description?: string; content?: Record<string, DocContent> }
  responses?: Record<string, { description?: string; headers?: Record<string, DocParameter>; content?: Record<string, DocContent> }>
  security?: Record<string, string[]>[]
}
export interface DocEntry { id: string; method: string; path: string; title: string; group: string }
export interface DocDetail { entry: DocEntry; operation: DocOperation; schemas: Record<string, DocSchema>; securitySchemes: Record<string, { description?: string; type?: string; scheme?: string }> }
