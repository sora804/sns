import { Link, Outlet, useNavigate } from 'react-router-dom'
import { useLogout, useMe } from '../api/hooks'

export default function Layout() {
  const { data: me } = useMe()
  const logout = useLogout()
  const navigate = useNavigate()

  return (
    <div className="mx-auto min-h-screen max-w-xl px-4 pb-12">
      <header className="flex items-center gap-3 py-4">
        <Link to="/" className="text-xl font-bold text-sky-600">
          SNS
        </Link>
        <nav className="ml-auto flex items-center gap-3 text-sm">
          {me ? (
            <>
              <Link to={`/users/${me.handle}`} className="hover:underline">
                @{me.handle}
              </Link>
              <button
                type="button"
                disabled={logout.isPending}
                onClick={() => logout.mutate(undefined, { onSettled: () => navigate('/login') })}
                className="text-gray-500 hover:text-gray-800"
              >
                ログアウト
              </button>
            </>
          ) : (
            <>
              <Link to="/login" className="hover:underline">
                ログイン
              </Link>
              <Link to="/register" className="rounded-full bg-sky-500 px-3 py-1 text-white hover:bg-sky-600">
                新規登録
              </Link>
            </>
          )}
        </nav>
      </header>
      <main>
        <Outlet />
      </main>
    </div>
  )
}
