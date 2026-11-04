export type ApiErrorBody = {
  type: string
  message: string
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
      body = (await response.json()) as ApiErrorBody
    } catch {
      // keep default body
    }
    throw new ApiError(response.status, body)
  }
  return (await response.json()) as T
}
