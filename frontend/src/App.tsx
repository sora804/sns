import { Route, Routes } from 'react-router-dom'
import Layout from './components/Layout'
import { LoginPage, RegisterPage } from './pages/AuthPages'
import { HomePage, NotFoundPage, PostPage, ProfilePage } from './pages/Pages'

export default function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<HomePage />} />
        <Route path="login" element={<LoginPage />} />
        <Route path="register" element={<RegisterPage />} />
        <Route path="users/:handle" element={<ProfilePage />} />
        <Route path="posts/:id" element={<PostPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  )
}
