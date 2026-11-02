import { useState, type FormEvent } from 'react'
import { useCreatePost } from '../api/hooks'

export const MAX_POST_LENGTH = 140

/** サーバーと同じく、文字数は「見た目の文字」（サロゲートペアを1文字）で数える。 */
export function countChars(s: string): number {
  return [...s].length
}

export default function PostForm() {
  const [body, setBody] = useState('')
  const create = useCreatePost()
  const remaining = MAX_POST_LENGTH - countChars(body)
  const disabled = body.trim() === '' || remaining < 0 || create.isPending

  function submit(e: FormEvent) {
    e.preventDefault()
    create.mutate(body, { onSuccess: () => setBody('') })
  }

  return (
    <form onSubmit={submit} className="rounded-lg border border-gray-200 bg-white p-4">
      <textarea
        value={body}
        onChange={(e) => setBody(e.target.value)}
        placeholder="いまどうしてる？"
        rows={3}
        aria-label="投稿本文"
        className="w-full resize-none rounded border border-gray-300 p-2 focus:border-sky-500 focus:outline-none"
      />
      <div className="mt-2 flex items-center justify-between">
        <span
          data-testid="remaining"
          className={remaining < 0 ? 'text-sm font-semibold text-red-600' : 'text-sm text-gray-500'}
        >
          {remaining}
        </span>
        <button
          type="submit"
          disabled={disabled}
          className="rounded-full bg-sky-500 px-5 py-1.5 font-semibold text-white hover:bg-sky-600 disabled:opacity-40"
        >
          投稿する
        </button>
      </div>
      {create.error && <p role="alert" className="mt-2 text-sm text-red-600">{create.error.message}</p>}
    </form>
  )
}
