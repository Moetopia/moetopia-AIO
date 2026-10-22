import { documentationDetail } from '../../../../utils/documentation'
export default defineEventHandler(event => {
  const detail = documentationDetail(getRouterParam(event, 'id') ?? '')
  if (!detail) throw createError({ statusCode: 404, statusMessage: 'API operation not found' })
  return detail
})
