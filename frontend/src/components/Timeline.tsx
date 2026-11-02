import type { UseInfiniteQueryResult, InfiniteData } from '@tanstack/react-query'
import type { components } from '../api/schema'
import PostCard from './PostCard'

type Page = components['schemas']['PostPage']

/** 投稿一覧（「もっと見る」で追加取得）。ホームとプロフィールで共通に使う。 */
export default function Timeline({
  query,
  empty,
}: {
  query: UseInfiniteQueryResult<InfiniteData<Page>, Error>
  empty: string
}) {
  if (query.isPending) return <p className="p-4 text-gray-500">読み込み中…</p>
  if (query.isError) return <p role="alert" className="p-4 text-red-600">{query.error.message}</p>

  const posts = query.data.pages.flatMap((p) => p.items)
  if (posts.length === 0) return <p className="p-4 text-gray-500">{empty}</p>

  return (
    <div className="space-y-3">
      {posts.map((p) => (
        <PostCard key={p.id} post={p} />
      ))}
      {query.hasNextPage && (
        <button
          type="button"
          onClick={() => query.fetchNextPage()}
          disabled={query.isFetchingNextPage}
          className="w-full rounded-lg border border-gray-300 bg-white py-2 text-sky-600 hover:bg-gray-50 disabled:opacity-50"
        >
          {query.isFetchingNextPage ? '読み込み中…' : 'もっと見る'}
        </button>
      )}
    </div>
  )
}
