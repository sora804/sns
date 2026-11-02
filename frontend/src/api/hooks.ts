import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, unwrap } from './client'

// ---- 認証 ----

/** ログイン中のユーザー。未ログインなら null。 */
export function useMe() {
  return useQuery({
    queryKey: ['me'],
    queryFn: async () => {
      const r = await api.GET('/auth/me')
      if (r.response.status === 401) return null
      return unwrap(r)
    },
    staleTime: 60_000,
  })
}

export function useLogin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (v: { email: string; password: string }) =>
      unwrap(await api.POST('/auth/login', { body: v })),
    onSuccess: () => qc.invalidateQueries(),
  })
}

export function useRegister() {
  return useMutation({
    mutationFn: async (v: { handle: string; displayName: string; email: string; password: string }) =>
      unwrap(await api.POST('/auth/register', { body: v })),
  })
}

export function useLogout() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async () => unwrap(await api.POST('/auth/logout')),
    onSuccess: () => {
      qc.clear()
    },
  })
}

// ---- 取得 ----

export function useHomeTimeline() {
  return useInfiniteQuery({
    queryKey: ['timeline', 'home'],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) =>
      unwrap(await api.GET('/timeline/home', { params: { query: { cursor: pageParam } } })),
    getNextPageParam: (last) => last.nextCursor ?? undefined,
  })
}

export function useUserPosts(handle: string) {
  return useInfiniteQuery({
    queryKey: ['users', handle, 'posts'],
    initialPageParam: undefined as string | undefined,
    queryFn: async ({ pageParam }) =>
      unwrap(
        await api.GET('/users/{handle}/posts', {
          params: { path: { handle }, query: { cursor: pageParam } },
        }),
      ),
    getNextPageParam: (last) => last.nextCursor ?? undefined,
  })
}

export function useProfile(handle: string) {
  return useQuery({
    queryKey: ['users', handle],
    queryFn: async () => unwrap(await api.GET('/users/{handle}', { params: { path: { handle } } })),
  })
}

export function usePost(id: number) {
  return useQuery({
    queryKey: ['posts', id],
    queryFn: async () => unwrap(await api.GET('/posts/{id}', { params: { path: { id } } })),
  })
}

// ---- 更新 ----
// 更新後は表示中のデータがずれないよう、取得済みのデータをまとめて再取得する。

export function useCreatePost() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (body: string) => unwrap(await api.POST('/posts', { body: { body } })),
    onSuccess: () => qc.invalidateQueries(),
  })
}

export function useDeletePost() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (id: number) => unwrap(await api.DELETE('/posts/{id}', { params: { path: { id } } })),
    onSuccess: () => qc.invalidateQueries(),
  })
}

export function useToggleLike() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (v: { id: number; liked: boolean }) => {
      const params = { params: { path: { id: v.id } } }
      return unwrap(v.liked ? await api.DELETE('/posts/{id}/like', params) : await api.PUT('/posts/{id}/like', params))
    },
    onSuccess: () => qc.invalidateQueries(),
  })
}

export function useToggleFollow() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (v: { handle: string; following: boolean }) => {
      const params = { params: { path: { handle: v.handle } } }
      return unwrap(
        v.following ? await api.DELETE('/users/{handle}/follow', params) : await api.PUT('/users/{handle}/follow', params),
      )
    },
    onSuccess: () => qc.invalidateQueries(),
  })
}
