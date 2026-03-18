import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Toaster } from 'react-hot-toast'
import { AuthProvider, useAuth } from './AuthContext'
import Layout from './Layout'
import Login from './pages/Login'
import Boosters from './pages/Boosters'
import Decks from './pages/Decks'
import Cards from './pages/Cards'
import Trade from './pages/Trade'
import Market from './pages/Market'
import Settings from './pages/Settings'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { user, token } = useAuth()
  if (!token) return <Navigate to="/login" replace />
  if (!user) return <div className="min-h-screen flex items-center justify-center text-gray-400">Loading…</div>
  return <>{children}</>
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const { user } = useAuth()
  if (!user?.is_admin) return <Navigate to="/boosters" replace />
  return <>{children}</>
}

function AppRoutes() {
  const { token } = useAuth()
  return (
    <Routes>
      <Route path="/login" element={token ? <Navigate to="/boosters" replace /> : <Login />} />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <Layout />
          </ProtectedRoute>
        }
      >
        <Route index element={<Navigate to="/boosters" replace />} />
        <Route path="boosters" element={<Boosters />} />
        <Route path="decks" element={<Decks />} />
        <Route path="cards" element={<Cards />} />
        <Route path="trade" element={<Trade />} />
        <Route path="market" element={<Market />} />
        <Route
          path="settings"
          element={
            <AdminRoute>
              <Settings />
            </AdminRoute>
          }
        />
      </Route>
      <Route path="*" element={<Navigate to="/boosters" replace />} />
    </Routes>
  )
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <AppRoutes />
        <Toaster
          position="top-right"
          toastOptions={{
            style: { background: '#1f2937', color: '#f9fafb', border: '1px solid #374151' },
          }}
        />
      </BrowserRouter>
    </AuthProvider>
  )
}
