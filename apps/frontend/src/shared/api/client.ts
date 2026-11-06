export type ApiErrorBody = {
  type: string
  message: string
}

type gatewayErrorEnvelope = {
  error: ApiErrorBody
}

export class ApiError extends Error {
  readonly status: number
  readonly type: string

  constructor(status: number, body: ApiErrorBody) {
    super(body.message)
    this.status = status
    this.type = body.type
  }
}

export async function fetchJson<T>(input: string, init?: RequestInit): Promise<T> {
  const response = await fetch(input, {
    credentials: 'same-origin',
    ...init,
  })
  if (!response.ok) {
    let body: ApiErrorBody = { type: 'request_failed', message: response.statusText }
    try {
      const raw = (await response.json()) as ApiErrorBody | gatewayErrorEnvelope
      if ('error' in raw && raw.error !== undefined && typeof raw.error.type === 'string') {
        body = raw.error
      } else if ('type' in raw && typeof raw.type === 'string') {
        body = raw as ApiErrorBody
      }
    } catch {
      // keep default body
    }
    throw new ApiError(response.status, body)
  }
  return (await response.json()) as T
}
