import { useState, type FormEvent, type ReactNode } from 'react'
import { Link, Navigate, useNavigate } from 'react-router-dom'
import { useLogin, useMe, useRegister } from '../api/hooks'

function Field({
  label,
  hint,
  ...props
}: { label: string; hint?: string } & React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <label className="block">
      <span className="text-sm font-medium">{label}</span>
      <input {...props} className="mt-1 w-full rounded border border-gray-300 p-2 focus:border-sky-500 focus:outline-none" />
      {hint && <span className="text-xs text-gray-500">{hint}</span>}
    </label>
  )
}

function Card({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-6">
      <h1 className="mb-4 text-xl font-bold">{title}</h1>
      {children}
    </div>
  )
}

export function LoginPage() {
  const { data: me } = useMe()
  const navigate = useNavigate()
  const login = useLogin()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')

  if (me) return <Navigate to="/" replace />

  function submit(e: FormEvent) {
    e.preventDefault()
    login.mutate({ email, password }, { onSuccess: () => navigate('/') })
  }

  return (
    <Card title="ログイン">
      <form onSubmit={submit} className="space-y-4">
        <Field label="メールアドレス" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required autoComplete="email" />
        <Field label="パスワード" type="password" value={password} onChange={(e) => setPassword(e.target.value)} required autoComplete="current-password" />
        {login.error && <p role="alert" className="text-sm text-red-600">{login.error.message}</p>}
        <button type="submit" disabled={login.isPending} className="w-full rounded-full bg-sky-500 py-2 font-semibold text-white hover:bg-sky-600 disabled:opacity-50">
          ログイン
        </button>
      </form>
      <p className="mt-4 text-sm">
        アカウントがない場合は{' '}
        <Link to="/register" className="text-sky-600 hover:underline">
          新規登録
        </Link>
      </p>
    </Card>
  )
}

export function RegisterPage() {
  const { data: me } = useMe()
  const navigate = useNavigate()
  const register = useRegister()
  const login = useLogin()
  const [form, setForm] = useState({ handle: '', displayName: '', email: '', password: '' })

  if (me) return <Navigate to="/" replace />

  const set = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [k]: e.target.value })
  const error = register.error ?? login.error

  // 登録に成功したら、そのままログインしてホームへ進む
  function submit(e: FormEvent) {
    e.preventDefault()
    register.mutate(form, {
      onSuccess: () =>
        login.mutate({ email: form.email, password: form.password }, { onSuccess: () => navigate('/') }),
    })
  }

  return (
    <Card title="新規登録">
      <form onSubmit={submit} className="space-y-4">
        <Field label="ユーザーID" hint="3〜15文字の英数字とアンダースコア" value={form.handle} onChange={set('handle')} required pattern="[A-Za-z0-9_]{3,15}" autoComplete="username" />
        <Field label="表示名" value={form.displayName} onChange={set('displayName')} required maxLength={50} />
        <Field label="メールアドレス" type="email" value={form.email} onChange={set('email')} required autoComplete="email" />
        <Field label="パスワード" hint="8文字以上" type="password" value={form.password} onChange={set('password')} required minLength={8} maxLength={128} autoComplete="new-password" />
        {error && <p role="alert" className="text-sm text-red-600">{error.message}</p>}
        <button type="submit" disabled={register.isPending || login.isPending} className="w-full rounded-full bg-sky-500 py-2 font-semibold text-white hover:bg-sky-600 disabled:opacity-50">
          登録する
        </button>
      </form>
      <p className="mt-4 text-sm">
        すでにアカウントがある場合は{' '}
        <Link to="/login" className="text-sky-600 hover:underline">
          ログイン
        </Link>
      </p>
    </Card>
  )
}
