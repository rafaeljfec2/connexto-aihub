import { ApiError } from '../../../shared/api/client'

export function mutationErrorMessage(error: unknown): string | undefined {
  if (error instanceof ApiError) {
    return error.message
  }
  if (error instanceof Error) {
    return error.message
  }
  return undefined
}

export function firstMutationError(errors: unknown[]): string | undefined {
  for (const error of errors) {
    const message = mutationErrorMessage(error)
    if (message) {
      return message
    }
  }
  return undefined
}
