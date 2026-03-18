import { useEffect, useState } from 'react'
import api from '../api'
import toast from 'react-hot-toast'
import { Shield, ShieldOff, Ban, Trash2, Plus, AlertTriangle } from 'lucide-react'

interface MTGSet { id: string; code: string; set_name: string; enabled: boolean }
interface BannedCard { id: string; card_name: string; reason: string; banned_at: string }
interface AdminUser { id: string; username: string; is_admin: boolean }
interface SeedStatus { done: boolean; message: string }

export default function Settings() {
  const [sets, setSets] = useState<MTGSet[]>([])
  const [bans, setBans] = useState<BannedCard[]>([])
  const [users, setUsers] = useState<AdminUser[]>([])
  const [banInput, setBanInput] = useState('')
  const [tab, setTab] = useState<'sets' | 'banlist' | 'users'>('sets')
  const [seed, setSeed] = useState<SeedStatus | null>(null)
  const [resetting, setResetting] = useState(false)
  const [resetConfirm, setResetConfirm] = useState(false)

  async function fetchAll() {
    const [setsRes, bansRes, usersRes, seedRes] = await Promise.all([
      api.get('/admin/sets'),
      api.get('/admin/banlist'),
      api.get('/admin/users'),
      api.get('/seed-status'),
    ])
    setSets(setsRes.data)
    setBans(bansRes.data)
    setUsers(usersRes.data)
    setSeed(seedRes.data)
  }

  useEffect(() => {
    fetchAll()
    const id = setInterval(async () => {
      const res = await api.get('/seed-status')
      setSeed(res.data)
      if (res.data.done) {
        clearInterval(id)
        fetchAll()
      }
    }, 5000)
    return () => clearInterval(id)
  }, [])

  async function applyFormat(format: 'standard' | 'modern') {
    try {
      const res = await api.post(`/admin/sets/format/${format}`)
      toast.success(`${format === 'standard' ? 'Standard' : 'Modern'} sets applied (${res.data.enabled_sets} sets enabled)`)
      fetchAll()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to apply format')
    }
  }

  async function toggleSet(set: MTGSet) {
    await api.put(`/admin/sets/${set.id}/toggle`, { enabled: !set.enabled })
    setSets((prev) => prev.map((s) => s.id === set.id ? { ...s, enabled: !s.enabled } : s))
  }

  async function banCard() {
    if (!banInput.trim()) return
    try {
      const res = await api.post('/admin/banlist', { card_name: banInput.trim() })
      setBans((prev) => [...prev, res.data])
      setBanInput('')
      toast.success('Card banned')
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to ban')
    }
  }

  async function unban(id: string) {
    await api.delete(`/admin/banlist/${id}`)
    setBans((prev) => prev.filter((b) => b.id !== id))
    toast.success('Card unbanned')
  }

  async function resetDB() {
    if (!resetConfirm) {
      setResetConfirm(true)
      return
    }
    setResetting(true)
    try {
      await api.post('/admin/reset-db')
      toast.success('Database reset — all cards, decks, boosters and lists cleared')
      setResetConfirm(false)
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Reset failed')
    } finally {
      setResetting(false)
    }
  }

  async function toggleAdmin(u: AdminUser) {
    await api.put(`/admin/users/${u.id}/admin`, { is_admin: !u.is_admin })
    setUsers((prev) => prev.map((x) => x.id === u.id ? { ...x, is_admin: !x.is_admin } : x))
  }

  const tabs = [
    { key: 'sets', label: `Sets (${sets.filter((s) => s.enabled).length}/${sets.length})` },
    { key: 'banlist', label: `Banlist (${bans.length})` },
    { key: 'users', label: `Users (${users.length})` },
  ] as const

  return (
    <div>
      <h2 className="text-2xl font-bold mb-4">Admin Settings</h2>

      {/* Danger Zone */}
      <div className="mb-6 border border-red-800 rounded-xl p-4 bg-red-950/20 max-w-xl">
        <div className="flex items-center gap-2 mb-2">
          <AlertTriangle size={16} className="text-red-400" />
          <span className="text-sm font-semibold text-red-400">Danger Zone</span>
        </div>
        <p className="text-xs text-gray-400 mb-3">
          Resets all opened boosters, cards, decks, and lists for every user. JAD balances and trades are also cleared. User accounts are preserved.
        </p>
        <button
          onClick={resetDB}
          disabled={resetting}
          className={`flex items-center gap-2 px-4 py-2 rounded-lg text-sm font-medium transition-colors disabled:opacity-50 ${
            resetConfirm
              ? 'bg-red-600 hover:bg-red-500 text-white animate-pulse'
              : 'bg-gray-800 hover:bg-red-900 border border-red-800 text-red-400'
          }`}
        >
          <AlertTriangle size={14} />
          {resetting ? 'Resetting…' : resetConfirm ? 'Click again to confirm reset' : 'Reset Database'}
        </button>
        {resetConfirm && !resetting && (
          <button onClick={() => setResetConfirm(false)} className="ml-2 text-xs text-gray-500 hover:text-gray-300 underline">
            Cancel
          </button>
        )}
      </div>

      {seed && !seed.done && (
        <div className="bg-yellow-900/30 border border-yellow-700 rounded-xl px-4 py-3 mb-6 text-sm text-yellow-300 max-w-xl">
          ⏳ {seed.message}
        </div>
      )}
      {seed?.done && (
        <div className="bg-green-900/20 border border-green-800 rounded-xl px-4 py-3 mb-6 text-sm text-green-400 max-w-xl">
          ✓ {seed.message}
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 bg-gray-800 p-1 rounded-xl mb-6 w-fit">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
              tab === t.key ? 'bg-purple-600 text-white' : 'text-gray-400 hover:text-white'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      {/* Sets */}
      {tab === 'sets' && (
        <div>
          <div className="flex flex-wrap items-center gap-3 mb-5">
            <p className="text-sm text-gray-400">Quick select:</p>
            <button onClick={() => applyFormat('standard')}
              className="px-3 py-1.5 rounded-lg bg-blue-700 hover:bg-blue-600 text-sm font-medium">
              Standard legal
            </button>
            <button onClick={() => applyFormat('modern')}
              className="px-3 py-1.5 rounded-lg bg-purple-700 hover:bg-purple-600 text-sm font-medium">
              Modern legal
            </button>
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-2">
            {sets.map((set) => (
              <button
                key={set.id}
                onClick={() => toggleSet(set)}
                className={`flex items-center gap-2 px-3 py-2.5 rounded-xl border text-sm font-medium text-left transition-colors ${
                  set.enabled
                    ? 'bg-green-900/30 border-green-600 text-green-300'
                    : 'bg-gray-800 border-gray-700 text-gray-500'
                }`}
              >
                <span className="font-mono text-xs uppercase w-8">{set.code}</span>
                <span className="truncate flex-1">{set.set_name}</span>
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Banlist */}
      {tab === 'banlist' && (
        <div>
          <div className="flex gap-2 mb-4">
            <input
              type="text"
              placeholder="Card name to ban…"
              value={banInput}
              onChange={(e) => setBanInput(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && banCard()}
              className="bg-gray-800 border border-gray-600 rounded-lg px-4 py-2 text-sm focus:outline-none focus:border-red-500 w-72"
            />
            <button onClick={banCard}
              className="flex items-center gap-2 bg-red-700 hover:bg-red-600 px-4 py-2 rounded-lg text-sm font-medium">
              <Plus size={14} /> Ban Card
            </button>
          </div>

          {bans.length === 0 && <p className="text-gray-500 text-sm">No cards banned.</p>}

          <div className="space-y-2">
            {bans.map((ban) => (
              <div key={ban.id} className="flex items-center gap-3 bg-gray-900 border border-red-900/40 rounded-xl px-4 py-3">
                <Ban size={15} className="text-red-400 flex-shrink-0" />
                <span className="font-semibold text-sm flex-1">{ban.card_name}</span>
                <span className="text-xs text-gray-500 capitalize">{ban.reason}</span>
                <span className="text-xs text-gray-600">{new Date(ban.banned_at).toLocaleDateString()}</span>
                <button onClick={() => unban(ban.id)}
                  className="p-1.5 rounded-lg hover:bg-gray-700 text-gray-400 hover:text-white">
                  <Trash2 size={13} />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Users */}
      {tab === 'users' && (
        <div className="space-y-2">
          {users.map((u) => (
            <div key={u.id} className="flex items-center gap-4 bg-gray-900 border border-gray-700 rounded-xl px-4 py-3">
              <span className="font-semibold text-sm flex-1">{u.username}</span>
              {u.is_admin && <span className="text-xs bg-purple-900 text-purple-300 px-2 py-0.5 rounded-full">Admin</span>}
              <button
                onClick={() => toggleAdmin(u)}
                className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
                  u.is_admin
                    ? 'bg-gray-700 hover:bg-red-900 text-gray-300 hover:text-red-300'
                    : 'bg-gray-700 hover:bg-purple-700 text-gray-300 hover:text-white'
                }`}
              >
                {u.is_admin ? <><ShieldOff size={12} /> Revoke Admin</> : <><Shield size={12} /> Make Admin</>}
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
