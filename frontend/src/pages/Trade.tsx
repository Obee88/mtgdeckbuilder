import { useEffect, useState } from 'react'
import api from '../api'
import { useAuth } from '../AuthContext'
import toast from 'react-hot-toast'
import { Plus, X, Check, Ban, ArrowLeftRight } from 'lucide-react'

interface UserCard { id: string; card_name: string; image_uri: string; rarity: string; quantity: number }
interface TradeCard { user_card_id: string; card_name: string; image_uri: string; quantity: number }
interface Trade {
  id: string
  from_user_id: string
  from_username: string
  to_user_id: string
  to_username: string
  offered_cards: TradeCard[]
  offered_jad: number
  requested_cards: TradeCard[]
  requested_jad: number
  status: string
  created_at: string
}
interface TargetUser { id: string; username: string }

const statusColor: Record<string, string> = {
  pending: 'text-yellow-400',
  accepted: 'text-green-400',
  declined: 'text-red-400',
  cancelled: 'text-gray-500',
}

export default function Trade() {
  const { user, refreshUser } = useAuth()
  const [trades, setTrades] = useState<Trade[]>([])
  const [users, setUsers] = useState<TargetUser[]>([])
  const [creating, setCreating] = useState(false)

  // Form state
  const [toUser, setToUser] = useState<TargetUser | null>(null)
  const [myCards, setMyCards] = useState<UserCard[]>([])
  const [theirCards, setTheirCards] = useState<UserCard[]>([])
  const [offeredCards, setOfferedCards] = useState<TradeCard[]>([])
  const [requestedCards, setRequestedCards] = useState<TradeCard[]>([])
  const [offeredJAD, setOfferedJAD] = useState(0)
  const [requestedJAD, setRequestedJAD] = useState(0)

  async function fetchAll() {
    const [tradesRes, usersRes] = await Promise.all([api.get('/trades'), api.get('/users')])
    setTrades(tradesRes.data)
    setUsers(usersRes.data)
  }
  useEffect(() => { fetchAll() }, [])

  async function fetchMyCards() {
    const res = await api.get('/cards')
    setMyCards(res.data)
  }

  async function fetchTheirCards(userId: string) {
    const res = await api.get(`/users/${userId}/cards`)
    setTheirCards(res.data)
  }

  function startCreating() {
    setCreating(true)
    setToUser(null)
    setOfferedCards([])
    setRequestedCards([])
    setOfferedJAD(0)
    setRequestedJAD(0)
    fetchMyCards()
  }

  function selectUser(u: TargetUser) {
    setToUser(u)
    fetchTheirCards(u.id)
    setRequestedCards([])
  }

  function toggleOffer(card: UserCard) {
    setOfferedCards((prev) => {
      const exists = prev.find((c) => c.user_card_id === card.id)
      if (exists) return prev.filter((c) => c.user_card_id !== card.id)
      if (prev.length >= 10) { toast.error('Max 10 cards per side'); return prev }
      return [...prev, { user_card_id: card.id, card_name: card.card_name, image_uri: card.image_uri, quantity: 1 }]
    })
  }

  function toggleRequest(card: UserCard) {
    setRequestedCards((prev) => {
      const exists = prev.find((c) => c.user_card_id === card.id)
      if (exists) return prev.filter((c) => c.user_card_id !== card.id)
      if (prev.length >= 10) { toast.error('Max 10 cards per side'); return prev }
      return [...prev, { user_card_id: card.id, card_name: card.card_name, image_uri: card.image_uri, quantity: 1 }]
    })
  }

  async function submitTrade() {
    if (!toUser) return
    try {
      await api.post('/trades', {
        to_user_id: toUser.id,
        offered_cards: offeredCards,
        offered_jad: offeredJAD,
        requested_cards: requestedCards,
        requested_jad: requestedJAD,
      })
      toast.success('Trade offer sent!')
      setCreating(false)
      fetchAll()
      refreshUser()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to create trade')
    }
  }

  async function accept(id: string) {
    try {
      await api.put(`/trades/${id}/accept`)
      toast.success('Trade accepted!')
      fetchAll()
      refreshUser()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to accept')
    }
  }

  async function decline(id: string) {
    await api.put(`/trades/${id}/decline`)
    fetchAll()
  }

  if (creating) {
    return (
      <div>
        <div className="flex items-center gap-3 mb-6">
          <button onClick={() => setCreating(false)} className="p-2 rounded-lg hover:bg-gray-800 text-gray-400"><X size={18} /></button>
          <h2 className="text-xl font-bold">New Trade Offer</h2>
        </div>

        {/* Step 1: pick user */}
        {!toUser ? (
          <div>
            <p className="text-gray-400 mb-3 text-sm">Select who to trade with:</p>
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-2">
              {users.map((u) => (
                <button key={u.id} onClick={() => selectUser(u)}
                  className="bg-gray-800 hover:bg-purple-700 border border-gray-700 rounded-xl px-4 py-3 text-sm font-medium text-left transition-colors">
                  {u.username}
                </button>
              ))}
            </div>
          </div>
        ) : (
          <div className="space-y-6">
            <p className="text-sm text-gray-400">Trading with <span className="text-white font-semibold">{toUser.username}</span></p>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {/* Your offer */}
              <div>
                <h3 className="font-semibold mb-3 text-purple-300">You offer</h3>
                <div className="flex items-center gap-2 mb-3">
                  <span className="text-sm text-gray-400">JAD:</span>
                  <input type="number" min={0} value={offeredJAD}
                    onChange={(e) => setOfferedJAD(Math.max(0, +e.target.value))}
                    className="w-24 bg-gray-800 border border-gray-600 rounded-lg px-3 py-1.5 text-sm focus:outline-none" />
                </div>
                <div className="grid grid-cols-3 gap-2 max-h-64 overflow-y-auto">
                  {myCards.map((c) => {
                    const sel = offeredCards.some((x) => x.user_card_id === c.id)
                    return (
                      <button key={c.id} onClick={() => toggleOffer(c)}
                        className={`rounded-lg overflow-hidden border-2 transition-colors ${sel ? 'border-purple-500' : 'border-gray-700'}`}>
                        {c.image_uri && <img src={c.image_uri} className="w-full" />}
                        <p className="text-xs p-1 truncate">{c.card_name}</p>
                      </button>
                    )
                  })}
                </div>
              </div>

              {/* Your request */}
              <div>
                <h3 className="font-semibold mb-3 text-blue-300">You want</h3>
                <div className="flex items-center gap-2 mb-3">
                  <span className="text-sm text-gray-400">JAD:</span>
                  <input type="number" min={0} value={requestedJAD}
                    onChange={(e) => setRequestedJAD(Math.max(0, +e.target.value))}
                    className="w-24 bg-gray-800 border border-gray-600 rounded-lg px-3 py-1.5 text-sm focus:outline-none" />
                </div>
                <div className="grid grid-cols-3 gap-2 max-h-64 overflow-y-auto">
                  {theirCards.map((c) => {
                    const sel = requestedCards.some((x) => x.user_card_id === c.id)
                    return (
                      <button key={c.id} onClick={() => toggleRequest(c)}
                        className={`rounded-lg overflow-hidden border-2 transition-colors ${sel ? 'border-blue-500' : 'border-gray-700'}`}>
                        {c.image_uri && <img src={c.image_uri} className="w-full" />}
                        <p className="text-xs p-1 truncate">{c.card_name}</p>
                      </button>
                    )
                  })}
                </div>
              </div>
            </div>

            <button onClick={submitTrade}
              className="bg-purple-600 hover:bg-purple-700 px-6 py-3 rounded-xl font-semibold flex items-center gap-2">
              <ArrowLeftRight size={16} /> Send Trade Offer
            </button>
          </div>
        )}
      </div>
    )
  }

  return (
    <div>
      <div className="flex items-center gap-4 mb-6">
        <h2 className="text-2xl font-bold">Trades</h2>
        <button onClick={startCreating}
          className="flex items-center gap-2 bg-purple-600 hover:bg-purple-700 px-4 py-2 rounded-xl text-sm font-medium">
          <Plus size={15} /> New Trade
        </button>
      </div>

      {trades.length === 0 && (
        <p className="text-center text-gray-500 py-16">No trades yet</p>
      )}

      <div className="space-y-3">
        {trades.map((t) => {
          const isReceiver = t.to_user_id === user?.id
          const isPending = t.status === 'pending'
          return (
            <div key={t.id} className="bg-gray-900 border border-gray-700 rounded-2xl p-4">
              <div className="flex items-start justify-between mb-3">
                <div className="text-sm">
                  <span className="font-semibold text-purple-300">{t.from_username}</span>
                  <span className="text-gray-400 mx-2">→</span>
                  <span className="font-semibold text-blue-300">{t.to_username}</span>
                </div>
                <span className={`text-xs font-semibold ${statusColor[t.status] ?? 'text-gray-400'}`}>
                  {t.status.toUpperCase()}
                </span>
              </div>

              <div className="grid grid-cols-2 gap-4 text-xs text-gray-400 mb-3">
                <div>
                  <p className="font-medium text-gray-300 mb-1">Offered</p>
                  {t.offered_jad > 0 && <p>{t.offered_jad} JAD</p>}
                  {t.offered_cards.map((c, i) => <p key={i}>• {c.card_name} ×{c.quantity}</p>)}
                </div>
                <div>
                  <p className="font-medium text-gray-300 mb-1">Requested</p>
                  {t.requested_jad > 0 && <p>{t.requested_jad} JAD</p>}
                  {t.requested_cards.map((c, i) => <p key={i}>• {c.card_name} ×{c.quantity}</p>)}
                </div>
              </div>

              {isReceiver && isPending && (
                <div className="flex gap-2">
                  <button onClick={() => accept(t.id)}
                    className="flex items-center gap-1.5 bg-green-700 hover:bg-green-600 px-3 py-1.5 rounded-lg text-xs font-medium">
                    <Check size={13} /> Accept
                  </button>
                  <button onClick={() => decline(t.id)}
                    className="flex items-center gap-1.5 bg-red-800 hover:bg-red-700 px-3 py-1.5 rounded-lg text-xs font-medium">
                    <Ban size={13} /> Decline
                  </button>
                </div>
              )}
              {!isReceiver && isPending && (
                <button onClick={() => decline(t.id)}
                  className="flex items-center gap-1.5 bg-gray-700 hover:bg-gray-600 px-3 py-1.5 rounded-lg text-xs font-medium">
                  <X size={13} /> Cancel
                </button>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
