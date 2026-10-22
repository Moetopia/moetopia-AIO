import { documentation } from '../../../utils/documentation'
export default defineEventHandler(event => {
  setHeader(event, 'content-type', 'application/json; charset=utf-8')
  return documentation
})
