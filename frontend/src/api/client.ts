import createClient from 'openapi-fetch'
import type { components, paths } from './schema'

export type Post = components['schemas']['Post']
export type User = components['schemas']['User']
export type Profile = components['schemas']['Profile']
type ErrorBody = components['schemas']['Error']

// schema.d.ts は `npm run gen:api` で api/openapi.yml から自動生成する（手で編集しない）。
// X-Requested-With は、バックエンドの CSRF 対策で状態を変更するリクエストに必須のヘッダー。
export const api = createClient<paths>({
  baseUrl: '/api/v1',
  headers: { 'X-Requested-With': 'sns-web' },
})

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    public field?: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

interface Result<T> {
  data?: T
  error?: unknown
  response: Response
}

/** openapi-fetch の結果を取り出す。エラー応答なら ApiError を投げる（204 など本文なしは undefined を返す）。 */
export function unwrap<T>(r: Result<T>): T {
  if (r.error !== undefined || !r.response.ok) {
    const e = (r.error ?? {}) as Partial<ErrorBody>
    throw new ApiError(r.response.status, e.code ?? 'error', e.message ?? 'エラーが発生しました', e.field)
  }
  return r.data as T
}
