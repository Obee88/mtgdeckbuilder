import { useEffect, useState } from 'react'
import api from '../api'
import { useAuth } from '../AuthContext'
import toast from 'react-hot-toast'
import { Recycle } from 'lucide-react'
import CardImage from '../components/CardImage'

interface UserCard {
  id: string
  card_name: string
  rarity: string
  image_uri: string
  mana_cost: string
  type_line: string
  set_name: string
  quantity: number
}

const rarityColor: Record<string, string> = {
  common: 'border-gray-600',
  uncommon: 'border-blue-400',
  rare: 'border-yellow-400',
  'mythic rare': 'border-orange-500',
}

export default function Cards() {
  const { refreshUser } = useAuth()
  const [cards, setCards] = useState<UserCard[]>([])
  const [filters, setFilters] = useState({ name: '', rarity: '', set: '' })
  const [groupBy, setGroupBy] = useState<'none' | 'rarity' | 'set'>('none')

  async function fetchCards() {
    const params: Record<string, string> = {}
    if (filters.name) params.name = filters.name
    if (filters.rarity) params.rarity = filters.rarity
    if (filters.set) params.set = filters.set
    const res = await api.get('/cards', { params })
    setCards(res.data)
  }

  useEffect(() => { fetchCards() }, [filters])

  async function recycle(card: UserCard) {
    try {
      await api.post(`/cards/${card.id}/recycle`)
      toast.success(`Recycled ${card.card_name} for 1 JAD`)
      fetchCards()
      refreshUser()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to recycle')
    }
  }

  function groupCards(): { label: string; items: UserCard[] }[] {
    if (groupBy === 'none') return [{ label: '', items: cards }]
    const groups: Record<string, UserCard[]> = {}
    for (const c of cards) {
      const key = groupBy === 'rarity' ? c.rarity : c.set_name
      groups[key] = [...(groups[key] ?? []), c]
    }
    return Object.entries(groups)
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([label, items]) => ({ label, items }))
  }

  return (
    <div>
      <div className="flex flex-wrap gap-3 mb-6">
        <h2 className="text-2xl font-bold flex-1">My Cards</h2>
        <input
          type="text"
          placeholder="Search name…"
          value={filters.name}
          onChange={(e) => setFilters((f) => ({ ...f, name: e.target.value }))}
          className="bg-gray-800 border border-gray-600 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-purple-500 w-44"
        />
        <select
          value={filters.rarity}
          onChange={(e) => setFilters((f) => ({ ...f, rarity: e.target.value }))}
          className="bg-gray-800 border border-gray-600 rounded-lg px-3 py-2 text-sm focus:outline-none"
        >
          <option value="">All rarities</option>
          <option value="common">Common</option>
          <option value="uncommon">Uncommon</option>
          <option value="rare">Rare</option>
          <option value="mythic rare">Mythic Rare</option>
        </select>
        <select
          value={groupBy}
          onChange={(e) => setGroupBy(e.target.value as typeof groupBy)}
          className="bg-gray-800 border border-gray-600 rounded-lg px-3 py-2 text-sm focus:outline-none"
        >
          <option value="none">No grouping</option>
          <option value="rarity">Group by rarity</option>
          <option value="set">Group by set</option>
        </select>
      </div>

      {cards.length === 0 && (
        <p className="text-gray-500 text-center py-16">No cards yet. Open some boosters!</p>
      )}

      {groupCards().map(({ label, items }) => (
        <div key={label} className="mb-8">
          {label && <h3 className="text-sm font-semibold uppercase text-gray-400 mb-3 capitalize">{label}</h3>}
          <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-3">
            {items.map((card) => (
              <div
                key={card.id}
                className={`rounded-xl overflow-hidden border-2 ${rarityColor[card.rarity] ?? 'border-gray-700'} bg-gray-900 group relative`}
              >
                {card.image_uri ? (
                  <CardImage src={card.image_uri} alt={card.card_name} className="w-full" />
                ) : (
                  <div className="aspect-[2/3] bg-gray-800 flex items-center justify-center p-2 text-xs text-center text-gray-400">
                    {card.card_name}
                  </div>
                )}
                <div className="p-2 flex items-center justify-between">
                  <div className="min-w-0">
                    <p className="text-xs font-medium truncate">{card.card_name}</p>
                    <p className="text-xs text-gray-500">×{card.quantity}</p>
                  </div>
                  <button
                    onClick={() => recycle(card)}
                    title="Recycle for 1 JAD"
                    className="ml-1 p-1.5 rounded-lg bg-gray-700 hover:bg-green-700 text-gray-400 hover:text-white transition-colors"
                  >
                    <Recycle size={14} />
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
