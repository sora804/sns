import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { describe, expect, it, vi } from 'vitest'
import type { Post } from '../api/client'
import { ApiError, unwrap } from '../api/client'
import PostCard from './PostCard'
import PostForm, { countChars } from './PostForm'

// ネットワークに出ないよう、ログイン状態だけ差し替える
let currentUser: { handle: string } | null = { handle: 'alice' }
vi.mock('../api/hooks', async (orig) => ({
  ...(await orig<typeof import('../api/hooks')>()),
  useMe: () => ({ data: currentUser }),
}))

function wrap(ui: React.ReactNode) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <MemoryRouter>{ui}</MemoryRouter>
    </QueryClientProvider>,
  )
}

const post: Post = {
  id: 1,
  body: '<script>alert(1)</script>こんにちは',
  createdAt: '2026-11-02T10:00:00Z',
  authorHandle: 'alice',
  authorDisplayName: 'アリス',
  likeCount: 2,
  likedByMe: false,
}

describe('countChars', () => {
  it('絵文字などのサロゲートペアを1文字として数える', () => {
    expect(countChars('あ😀')).toBe(2)
    expect(countChars('')).toBe(0)
  })
})

describe('unwrap', () => {
  it('エラー応答は ApiError にする', () => {
    const r = {
      error: { code: 'validation_error', message: '入力が正しくありません', field: 'body' },
      response: new Response(null, { status: 400 }),
    }
    expect(() => unwrap(r)).toThrow(ApiError)
    try {
      unwrap(r)
    } catch (e) {
      expect((e as ApiError).status).toBe(400)
      expect((e as ApiError).field).toBe('body')
    }
  })

  it('204 など本文が無い成功は undefined を返す', () => {
    expect(unwrap({ response: new Response(null, { status: 204 }) })).toBeUndefined()
  })
})

describe('PostForm', () => {
  it('空のときは投稿できない', () => {
    wrap(<PostForm />)
    expect(screen.getByRole('button', { name: '投稿する' })).toBeDisabled()
    expect(screen.getByTestId('remaining')).toHaveTextContent('140')
  })

  it('140文字を超えると残りがマイナスになり投稿できない', async () => {
    wrap(<PostForm />)
    await userEvent.click(screen.getByLabelText('投稿本文'))
    await userEvent.paste('あ'.repeat(141))
    expect(screen.getByTestId('remaining')).toHaveTextContent('-1')
    expect(screen.getByRole('button', { name: '投稿する' })).toBeDisabled()
  })

  it('140文字ちょうどなら投稿できる', async () => {
    wrap(<PostForm />)
    await userEvent.click(screen.getByLabelText('投稿本文'))
    await userEvent.paste('あ'.repeat(140))
    expect(screen.getByRole('button', { name: '投稿する' })).toBeEnabled()
  })
})

describe('PostCard', () => {
  it('本文のHTMLは実行されず、文字としてそのまま表示される', () => {
    const { container } = wrap(<PostCard post={post} />)
    expect(screen.getByText(/<script>alert\(1\)<\/script>/)).toBeInTheDocument()
    expect(container.querySelector('script')).toBeNull()
  })

  it('自分の投稿にだけ削除ボタンが出る', () => {
    currentUser = { handle: 'alice' }
    const { unmount } = wrap(<PostCard post={post} />)
    expect(screen.getByRole('button', { name: '削除' })).toBeInTheDocument()
    unmount()

    currentUser = { handle: 'bob' }
    wrap(<PostCard post={post} />)
    expect(screen.queryByRole('button', { name: '削除' })).toBeNull()
  })

  it('未ログインではいいねできない', () => {
    currentUser = null
    wrap(<PostCard post={post} />)
    expect(screen.getByRole('button', { name: 'いいねする' })).toBeDisabled()
  })
})
