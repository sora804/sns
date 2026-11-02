import { Link, Navigate, useParams } from 'react-router-dom'
import { useHomeTimeline, useMe, usePost, useProfile, useToggleFollow, useUserPosts } from '../api/hooks'
import PostCard from '../components/PostCard'
import PostForm from '../components/PostForm'
import Timeline from '../components/Timeline'

export function HomePage() {
  const { data: me, isPending } = useMe()
  const timeline = useHomeTimeline()

  if (isPending) return <p className="p-4 text-gray-500">読み込み中…</p>
  if (!me) return <Navigate to="/login" replace />

  return (
    <div className="space-y-4">
      <PostForm />
      <Timeline query={timeline} empty="まだ投稿がありません。最初の投稿をしてみましょう。他のユーザーをフォローするとここに表示されます。" />
    </div>
  )
}

export function ProfilePage() {
  const { handle = '' } = useParams()
  const { data: me } = useMe()
  const profile = useProfile(handle)
  const posts = useUserPosts(handle)
  const follow = useToggleFollow()

  if (profile.isPending) return <p className="p-4 text-gray-500">読み込み中…</p>
  if (profile.isError) return <NotFoundPage />

  const p = profile.data
  const isMe = me?.handle === p.handle

  return (
    <div className="space-y-4">
      <section className="rounded-lg border border-gray-200 bg-white p-4">
        <div className="flex items-start gap-3">
          <div className="min-w-0">
            <h1 className="truncate text-xl font-bold">{p.displayName}</h1>
            <p className="text-gray-500">@{p.handle}</p>
          </div>
          {me && !isMe && (
            <button
              type="button"
              disabled={follow.isPending}
              onClick={() => follow.mutate({ handle: p.handle, following: p.isFollowing })}
              className={
                p.isFollowing
                  ? 'ml-auto rounded-full border border-gray-300 px-4 py-1 hover:border-red-300 hover:text-red-600'
                  : 'ml-auto rounded-full bg-gray-900 px-4 py-1 text-white hover:bg-gray-700'
              }
            >
              {p.isFollowing ? 'フォロー中' : 'フォローする'}
            </button>
          )}
        </div>
        {p.bio && <p className="mt-2 whitespace-pre-wrap">{p.bio}</p>}
        <dl className="mt-3 flex gap-4 text-sm text-gray-600">
          <div><dt className="inline">投稿 </dt><dd className="inline font-semibold">{p.postCount}</dd></div>
          <div><dt className="inline">フォロー中 </dt><dd className="inline font-semibold">{p.followingCount}</dd></div>
          <div><dt className="inline">フォロワー </dt><dd className="inline font-semibold">{p.followerCount}</dd></div>
        </dl>
      </section>
      <Timeline query={posts} empty="まだ投稿がありません。" />
    </div>
  )
}

export function PostPage() {
  const id = Number(useParams().id)
  const post = usePost(id)

  if (!Number.isInteger(id) || id <= 0) return <NotFoundPage />
  if (post.isPending) return <p className="p-4 text-gray-500">読み込み中…</p>
  if (post.isError) return <NotFoundPage />
  return <PostCard post={post.data} />
}

export function NotFoundPage() {
  return (
    <div className="p-8 text-center">
      <p className="mb-4 text-lg">ページが見つかりません</p>
      <Link to="/" className="text-sky-600 hover:underline">
        ホームへ戻る
      </Link>
    </div>
  )
}
