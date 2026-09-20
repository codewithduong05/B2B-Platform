import { resolveRequestContext } from '../utils/request-context'

export default defineEventHandler((event: any) => {
  resolveRequestContext(event)
})
