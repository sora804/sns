import { Link } from 'react-router-dom'
import type { Post } from '../api/client'
import { useDeletePost, useMe, useToggleLike } from '../api/hooks'

export default function PostCard({ post }: { post: Post }) {
  const { data: me } = useMe()
  const like = useToggleLike()
  const del = useDeletePost()
  const isMine = me?.handle === post.authorHandle

  return (
    <article className="rounded-lg border border-gray-200 bg-white p-4">
      <header className="flex items-baseline gap-2 text-sm">
        <Link to={`/users/${post.authorHandle}`} className="font-semibold hover:underline">
          {post.authorDisplayName}
        </Link>
        <span className="text-gray-500">@{post.authorHandle}</span>
        <Link to={`/posts/${post.id}`} className="ml-auto text-gray-400 hover:underline">
          {new Date(post.createdAt).toLocaleString('ja-JP')}
        </Link>
      </header>

      {/* React が自動でエスケープするため、本文に HTML が含まれていても実行されない */}
      <p className="mt-2 whitespace-pre-wrap break-words">{post.body}</p>

      <footer className="mt-3 flex items-center gap-4 text-sm">
        <button
          type="button"
          disabled={!me || like.isPending}
          onClick={() => like.mutate({ id: post.id, liked: post.likedByMe })}
          aria-pressed={post.likedByMe}
          aria-label={post.likedByMe ? 'いいねを取り消す' : 'いいねする'}
          title={me ? undefined : 'ログインするといいねできます'}
          className={post.likedByMe ? 'text-pink-600' : 'text-gray-500 hover:text-pink-600 disabled:hover:text-gray-500'}
        >
          {post.likedByMe ? '♥' : '♡'} {post.likeCount}
        </button>
        {isMine && (
          <button
            type="button"
            disabled={del.isPending}
            onClick={() => {
              if (window.confirm('この投稿を削除しますか？')) del.mutate(post.id)
            }}
            className="text-gray-400 hover:text-red-600"
          >
            削除
          </button>
        )}
      </footer>
    </article>
  )
}
