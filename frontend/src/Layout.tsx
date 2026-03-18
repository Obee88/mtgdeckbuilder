import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from './AuthContext'
import { PackageOpen, Layers, CreditCard, ArrowLeftRight, Store, Settings, LogOut, Coins } from 'lucide-react'

const navItems = [
  { to: '/boosters', label: 'Boosters', icon: PackageOpen },
  { to: '/decks', label: 'Decks', icon: Layers },
  { to: '/cards', label: 'Cards', icon: CreditCard },
  { to: '/trade', label: 'Trade', icon: ArrowLeftRight },
  { to: '/market', label: 'Market', icon: Store },
]

export default function Layout() {
  const { user, logout } = useAuth()

  return (
    <div className="flex min-h-screen bg-gray-950">
      {/* Sidebar */}
      <aside className="w-56 bg-gray-900 border-r border-gray-800 flex flex-col py-6 px-3">
        <div className="px-3 mb-8">
          <h1 className="text-xl font-bold text-purple-400">MTG Vault</h1>
          <p className="text-xs text-gray-500 mt-1">{user?.username}</p>
        </div>

        {/* JAD balance */}
        <div className="mx-3 mb-6 bg-gray-800 rounded-lg px-3 py-2 flex items-center gap-2">
          <Coins size={16} className="text-yellow-400" />
          <span className="text-sm text-yellow-400 font-semibold">{user?.jad ?? 0} JAD</span>
          {(user?.jad_locked ?? 0) > 0 && (
            <span className="text-xs text-gray-500 ml-auto">({user?.jad_locked} locked)</span>
          )}
        </div>

        <nav className="flex-1 space-y-1">
          {navItems.map(({ to, label, icon: Icon }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-purple-700 text-white'
                    : 'text-gray-400 hover:text-white hover:bg-gray-800'
                }`
              }
            >
              <Icon size={18} />
              {label}
            </NavLink>
          ))}

          {user?.is_admin && (
            <NavLink
              to="/settings"
              className={({ isActive }) =>
                `flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-purple-700 text-white'
                    : 'text-gray-400 hover:text-white hover:bg-gray-800'
                }`
              }
            >
              <Settings size={18} />
              Settings
            </NavLink>
          )}
        </nav>

        <button
          onClick={logout}
          className="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm text-gray-400 hover:text-white hover:bg-gray-800 transition-colors mt-4"
        >
          <LogOut size={18} />
          Logout
        </button>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto p-6">
        <Outlet />
      </main>
    </div>
  )
}
