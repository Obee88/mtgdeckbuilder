import { useEffect, useState, useRef } from 'react'
import api from '../api'
import toast from 'react-hot-toast'
import { Plus, Trash2, Save, ChevronLeft } from 'lucide-react'
import CardImage from '../components/CardImage'

interface DeckCard { card_id: string; card_name: string; image_uri: string; quantity: number }
interface Deck { id: string; name: string; cards: DeckCard[]; updated_at: string }
interface CardResult { id: string; name: string; image_uri: string }

export default function Decks() {
  const [decks, setDecks] = useState<Deck[]>([])
  const [active, setActive] = useState<Deck | null>(null)
  const [ownedCounts, setOwnedCounts] = useState<Record<string, number>>({})
  const [search, setSearch] = useState('')
  const [suggestions, setSuggestions] = useState<CardResult[]>([])
  const [creating, setCreating] = useState(false)
  const [newName, setNewName] = useState('')
  const searchRef = useRef<HTMLDivElement>(null)

  async function fetchDecks() {
    const res = await api.get('/decks')
    setDecks(res.data)
  }
  async function fetchOwned() {
    const res = await api.get('/cards/owned-counts')
    setOwnedCounts(res.data)
  }

  useEffect(() => { fetchDecks(); fetchOwned() }, [])

  // Autocomplete
  useEffect(() => {
    if (search.length < 2) { setSuggestions([]); return }
    const t = setTimeout(async () => {
      const res = await api.get('/cards/search', { params: { q: search } })
      setSuggestions(res.data)
    }, 300)
    return () => clearTimeout(t)
  }, [search])

  async function createDeck() {
    if (!newName.trim()) return
    const res = await api.post('/decks', { name: newName.trim() })
    setDecks((d) => [...d, res.data])
    setActive(res.data)
    setNewName('')
    setCreating(false)
  }

  async function saveDeck() {
    if (!active) return
    await api.put(`/decks/${active.id}`, { name: active.name, cards: active.cards })
    toast.success('Deck saved')
    fetchDecks()
  }

  async function deleteDeck(id: string) {
    await api.delete(`/decks/${id}`)
    setDecks((d) => d.filter((x) => x.id !== id))
    if (active?.id === id) setActive(null)
  }

  function addCard(card: CardResult) {
    if (!active) return
    setActive((prev) => {
      if (!prev) return prev
      const existing = prev.cards.find((c) => c.card_id === card.id)
      if (existing) {
        return { ...prev, cards: prev.cards.map((c) => c.card_id === card.id ? { ...c, quantity: c.quantity + 1 } : c) }
      }
      return { ...prev, cards: [...prev.cards, { card_id: card.id, card_name: card.name, image_uri: card.image_uri, quantity: 1 }] }
    })
    setSearch('')
    setSuggestions([])
  }

  function changeQty(cardId: string, delta: number) {
    if (!active) return
    setActive((prev) => {
      if (!prev) return prev
      const cards = prev.cards.map((c) => c.card_id === cardId ? { ...c, quantity: Math.max(0, c.quantity + delta) } : c)
        .filter((c) => c.quantity > 0)
      return { ...prev, cards }
    })
  }

  const totalCards = active?.cards.reduce((s, c) => s + c.quantity, 0) ?? 0

  return (
    <div className="flex gap-6 h-full">
      {/* Deck list */}
      <div className="w-56 flex-shrink-0">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold">Decks</h2>
          <button onClick={() => setCreating(true)} className="p-1.5 rounded-lg bg-purple-700 hover:bg-purple-600 text-white">
            <Plus size={16} />
          </button>
        </div>

        {creating && (
          <div className="mb-3 flex gap-2">
            <input
              autoFocus
              type="text"
              placeholder="Deck name"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && createDeck()}
              className="flex-1 bg-gray-800 border border-gray-600 rounded-lg px-2 py-1.5 text-sm focus:outline-none"
            />
            <button onClick={createDeck} className="px-2 py-1 bg-purple-700 rounded-lg text-sm">OK</button>
          </div>
        )}

        <div className="space-y-1">
          {decks.map((deck) => (
            <div
              key={deck.id}
              onClick={() => setActive(deck)}
              className={`flex items-center justify-between px-3 py-2 rounded-lg cursor-pointer group ${
                active?.id === deck.id ? 'bg-purple-700 text-white' : 'hover:bg-gray-800 text-gray-300'
              }`}
            >
              <span className="text-sm truncate flex-1">{deck.name}</span>
              <button
                onClick={(e) => { e.stopPropagation(); deleteDeck(deck.id) }}
                className="opacity-0 group-hover:opacity-100 p-1 hover:text-red-400 transition-opacity"
              >
                <Trash2 size={13} />
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* Deck editor */}
      {active ? (
        <div className="flex-1">
          <div className="flex items-center gap-3 mb-5">
            <button onClick={() => setActive(null)} className="p-1 text-gray-400 hover:text-white">
              <ChevronLeft size={20} />
            </button>
            <input
              value={active.name}
              onChange={(e) => setActive((p) => p ? { ...p, name: e.target.value } : p)}
              className="bg-transparent text-xl font-bold border-b border-gray-600 focus:outline-none focus:border-purple-500 pb-1"
            />
            <span className="text-sm text-gray-400 ml-2">{totalCards} cards</span>
            <button onClick={saveDeck} className="ml-auto flex items-center gap-2 bg-purple-600 hover:bg-purple-700 px-4 py-2 rounded-lg text-sm font-medium">
              <Save size={15} /> Save
            </button>
          </div>

          {/* Search / autocomplete */}
          <div className="relative mb-5" ref={searchRef}>
            <input
              type="text"
              placeholder="Add card by name…"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full bg-gray-800 border border-gray-600 rounded-lg px-4 py-2.5 text-sm focus:outline-none focus:border-purple-500"
            />
            {suggestions.length > 0 && (
              <div className="absolute z-10 mt-1 w-full bg-gray-800 border border-gray-600 rounded-xl shadow-xl max-h-60 overflow-y-auto">
                {suggestions.map((s) => (
                  <button
                    key={s.id}
                    onClick={() => addCard(s)}
                    className="w-full text-left px-4 py-2 text-sm hover:bg-gray-700 flex items-center gap-3"
                  >
                    {s.image_uri && <CardImage src={s.image_uri} alt={s.name} className="w-8 rounded" />}
                    <span>{s.name}</span>
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Card list */}
          <div className="space-y-2">
            {active.cards.map((card) => {
              const owned = ownedCounts[card.card_name] ?? 0
              return (
                <div key={card.card_id} className="flex items-center gap-3 bg-gray-900 rounded-xl px-4 py-3">
                  {card.image_uri && <CardImage src={card.image_uri} alt={card.card_name} className="w-10 rounded" />}
                  <span className="flex-1 text-sm font-medium">{card.card_name}</span>
                  <span className="text-xs text-gray-400">({owned} owned)</span>
                  <div className="flex items-center gap-2">
                    <button onClick={() => changeQty(card.card_id, -1)} className="w-7 h-7 rounded-lg bg-gray-700 hover:bg-gray-600 flex items-center justify-center text-lg leading-none">−</button>
                    <span className="w-6 text-center text-sm font-semibold">{card.quantity}</span>
                    <button onClick={() => changeQty(card.card_id, +1)} className="w-7 h-7 rounded-lg bg-gray-700 hover:bg-gray-600 flex items-center justify-center text-lg leading-none">+</button>
                  </div>
                </div>
              )
            })}
            {active.cards.length === 0 && (
              <p className="text-center text-gray-500 py-8 text-sm">Search for cards above to add them</p>
            )}
          </div>
        </div>
      ) : (
        <div className="flex-1 flex items-center justify-center text-gray-500">
          Select a deck or create a new one
        </div>
      )}
    </div>
  )
}
