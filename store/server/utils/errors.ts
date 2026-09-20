import { createError } from 'h3'
import { toWireError, type AppError } from '../../shared/errors'

export function sendAppError(event: any, error: AppError, statusCode: number): never {
  throw createError({
    statusCode,
    statusMessage: error.message,
    data: toWireError(error),
  })
}
