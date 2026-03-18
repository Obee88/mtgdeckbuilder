import { useEffect, useState, useRef } from 'react'
import api from '../api'
import { useAuth } from '../AuthContext'
import toast from 'react-hot-toast'
import { PackageOpen, Recycle, ChevronDown } from 'lucide-react'
import CardImage from '../components/CardImage'

interface BoosterCard {
  id: string
  card_id: string
  card_name: string
  rarity: string
  image_uri: string
  mana_cost: string
  type_line: string
}

interface Deck { id: string; name: string; cards: { card_id: string }[] }
interface BoosterStatus { available: number; next_booster: string }
interface SeedStatus { done: boolean; message: string }

const rarityColor: Record<string, string> = {
  common: 'border-gray-500',
  uncommon: 'border-blue-400',
  rare: 'border-yellow-400',
  mythic: 'border-orange-500',
}
const rarityBadge: Record<string, string> = {
  common: 'bg-gray-700 text-gray-300',
  uncommon: 'bg-blue-900 text-blue-300',
  rare: 'bg-yellow-900 text-yellow-300',
  mythic: 'bg-orange-900 text-orange-300',
}

function Countdown({ target }: { target: string }) {
  const [diff, setDiff] = useState('')
  useEffect(() => {
    function calc() {
      const ms = new Date(target).getTime() - Date.now()
      if (ms <= 0) { setDiff('Available now!'); return }
      const d = Math.floor(ms / 86400000)
      const h = Math.floor((ms % 86400000) / 3600000)
      const m = Math.floor((ms % 3600000) / 60000)
      setDiff(`${d}d ${h}h ${m}m`)
    }
    calc()
    const id = setInterval(calc, 60000)
    return () => clearInterval(id)
  }, [target])
  return <span>{diff}</span>
}

function CardActions({ card, decks, onRecycle }: {
  card: BoosterCard
  decks: Deck[]
  onRecycle: (cardId: string) => void
}) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  const inDecks = decks.filter(d => d.cards.some(c => c.card_id === card.card_id))

  async function addToDeck(deck: Deck) {
    try {
      await api.post(`/decks/${deck.id}/add-card`, {
        card_id: card.card_id,
        card_name: card.card_name,
        image_uri: card.image_uri,
      })
      toast.success(`Added to "${deck.name}"`)
      setOpen(false)
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to add to deck')
    }
  }

  return (
    <div className="p-2 space-y-1">
      <p className="text-xs font-medium truncate">{card.card_name}</p>
      <span className={`text-xs px-1.5 py-0.5 rounded inline-block ${rarityBadge[card.rarity] ?? 'bg-gray-700 text-gray-300'}`}>
        {card.rarity}
      </span>

      {inDecks.length > 0 && (
        <p className="text-xs text-purple-400 truncate">
          In: {inDecks.map(d => d.name).join(', ')}
        </p>
      )}

      <div className="flex gap-1 pt-1">
        {/* Recycle */}
        <button
          onClick={() => onRecycle(card.id)}
          title="Recycle for 1 JAD"
          className="flex-1 flex items-center justify-center gap-1 py-1 rounded-lg bg-gray-700 hover:bg-green-700 text-xs text-gray-300 hover:text-white transition-colors"
        >
          <Recycle size={11} /> 1 JAD
        </button>

        {/* Add to deck */}
        <div className="relative flex-1" ref={ref}>
          <button
            onClick={() => setOpen(o => !o)}
            className="w-full flex items-center justify-center gap-1 py-1 rounded-lg bg-gray-700 hover:bg-purple-700 text-xs text-gray-300 hover:text-white transition-colors"
          >
            Deck <ChevronDown size={10} />
          </button>
          {open && (
            <div className="absolute bottom-full mb-1 left-0 z-20 bg-gray-800 border border-gray-600 rounded-xl shadow-xl min-w-36 max-h-48 overflow-y-auto">
              {decks.length === 0
                ? <p className="text-xs text-gray-500 px-3 py-2">No decks yet</p>
                : decks.map(d => (
                  <button
                    key={d.id}
                    onClick={() => addToDeck(d)}
                    className="w-full text-left px-3 py-2 text-xs hover:bg-gray-700 truncate"
                  >
                    {d.name}
                  </button>
                ))
              }
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default function Boosters() {
  const { refreshUser } = useAuth()
  const [status, setStatus] = useState<BoosterStatus | null>(null)
  const [seed, setSeed] = useState<SeedStatus | null>(null)
  const [opening, setOpening] = useState(false)
  const [cards, setCards] = useState<BoosterCard[]>([])
  const [decks, setDecks] = useState<Deck[]>([])

  async function fetchStatus() {
    const res = await api.get('/boosters')
    setStatus(res.data)
  }
  async function fetchDecks() {
    const res = await api.get('/decks')
    setDecks(res.data)
  }
  async function fetchSeedStatus() {
    const res = await api.get('/seed-status')
    setSeed(res.data)
    return res.data as SeedStatus
  }

  useEffect(() => {
    fetchStatus()
    fetchDecks()
    fetchSeedStatus().then((s) => {
      if (!s.done) {
        const id = setInterval(async () => {
          const updated = await fetchSeedStatus()
          if (updated.done) clearInterval(id)
        }, 5000)
        return () => clearInterval(id)
      }
    })
  }, [])

  async function openBooster() {
    setOpening(true)
    try {
      const res = await api.post('/boosters/open')
      setCards(res.data.cards ?? [])
      await fetchStatus()
      await refreshUser()
      toast.success('Booster opened!')
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to open booster')
    } finally {
      setOpening(false)
    }
  }

  async function recycle(userCardId: string) {
    try {
      await api.post(`/cards/${userCardId}/recycle`)
      toast.success('Recycled for 1 JAD')
      refreshUser()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to recycle')
    }
  }

  return (
    <div>
      <h2 className="text-2xl font-bold mb-6">Boosters</h2>

      {seed && !seed.done && (
        <div className="bg-yellow-900/30 border border-yellow-700 rounded-xl px-4 py-3 mb-6 max-w-sm text-sm text-yellow-300">
          ⏳ {seed.message}
        </div>
      )}

      <div className="bg-gray-900 border border-gray-700 rounded-2xl p-6 mb-8 max-w-sm">
        <div className="flex items-center gap-3 mb-4">
          <PackageOpen size={24} className="text-purple-400" />
          <span className="text-lg font-semibold">Available Boosters</span>
        </div>
        <div className="text-5xl font-bold text-purple-400 mb-2">
          {status?.available ?? '—'}
        </div>
        <p className="text-sm text-gray-400 mb-6">
          Next booster in: <span className="text-white font-medium">
            {status ? <Countdown target={status.next_booster} /> : '…'}
          </span>
        </p>
        <button
          onClick={openBooster}
          disabled={opening || !status?.available || !seed?.done}
          className="w-full bg-purple-600 hover:bg-purple-700 disabled:opacity-40 disabled:cursor-not-allowed text-white font-semibold py-3 rounded-xl transition-colors"
        >
          {opening ? 'Opening…' : !seed?.done ? 'Waiting for card data…' : 'Open Booster (30 cards)'}
        </button>
      </div>

      {cards.length > 0 && (
        <div>
          <h3 className="text-lg font-semibold mb-4">You received:</h3>
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-3">
            {cards.map((card, i) => (
              <div
                key={i}
                className={`rounded-xl overflow-hidden border-2 ${rarityColor[card.rarity] ?? 'border-gray-700'} bg-gray-900`}
              >
                {card.image_uri
                  ? <CardImage src={card.image_uri} alt={card.card_name} className="w-full" />
                  : <div className="aspect-[2/3] bg-gray-800 flex items-center justify-center p-2 text-xs text-center text-gray-400">{card.card_name}</div>
                }
                <CardActions
                  card={card}
                  decks={decks}
                  onRecycle={recycle}
                />
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}
