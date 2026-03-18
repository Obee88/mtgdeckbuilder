import { useEffect, useState } from 'react'
import api from '../api'
import { useAuth } from '../AuthContext'
import toast from 'react-hot-toast'
import { ThumbsDown, Gavel } from 'lucide-react'
import CardImage from '../components/CardImage'

interface MarketCard {
  id: string
  card_name: string
  image_uri: string
  rarity: string
  mana_cost: string
  type_line: string
  current_bid: number
  current_bidder_id: string | null
  current_bidder: string | null
  bid_expires_at: string | null
  hate_count: number
  hater_ids: string[]
  status: string
}

const rarityColor: Record<string, string> = {
  common: 'border-gray-600',
  uncommon: 'border-blue-400',
  rare: 'border-yellow-400',
  'mythic rare': 'border-orange-500',
}

function BidTimer({ expiresAt }: { expiresAt: string }) {
  const [label, setLabel] = useState('')
  useEffect(() => {
    function calc() {
      const ms = new Date(expiresAt).getTime() - Date.now()
      if (ms <= 0) { setLabel('Ending soon…'); return }
      const h = Math.floor(ms / 3600000)
      const m = Math.floor((ms % 3600000) / 60000)
      const s = Math.floor((ms % 60000) / 1000)
      setLabel(`${h}h ${m}m ${s}s`)
    }
    calc()
    const id = setInterval(calc, 1000)
    return () => clearInterval(id)
  }, [expiresAt])
  return <span className="text-green-400 text-xs font-mono">{label}</span>
}

export default function Market() {
  const { user, refreshUser } = useAuth()
  const [cards, setCards] = useState<MarketCard[]>([])

  async function fetchMarket() {
    const res = await api.get('/market')
    setCards(res.data)
  }

  useEffect(() => {
    fetchMarket()
    const id = setInterval(fetchMarket, 30000)
    return () => clearInterval(id)
  }, [])

  async function bid(card: MarketCard) {
    try {
      await api.post(`/market/${card.id}/bid`)
      toast.success(`Bid placed on ${card.card_name}!`)
      fetchMarket()
      refreshUser()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to bid')
    }
  }

  async function hate(card: MarketCard) {
    try {
      await api.post(`/market/${card.id}/hate`)
      toast.success(`Hated ${card.card_name}`)
      fetchMarket()
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to hate')
    }
  }

  const nextBid = (card: MarketCard) =>
    card.current_bidder_id ? Math.ceil(card.current_bid * 1.1) : card.current_bid

  return (
    <div>
      <div className="flex items-center gap-4 mb-6">
        <h2 className="text-2xl font-bold">Market</h2>
        <span className="text-sm text-gray-400">{cards.length}/20 cards listed</span>
      </div>

      <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
        {cards.map((card) => {
          const alreadyBidder = card.current_bidder_id === user?.id
          const alreadyHated = card.hater_ids?.includes(user?.id ?? '')
          const hasBid = !!card.current_bidder_id
          const bidCost = nextBid(card)
          const canBid = !alreadyBidder && (user?.jad ?? 0) - (user?.jad_locked ?? 0) >= bidCost
          const canHate = !hasBid && !alreadyHated

          return (
            <div key={card.id} className={`rounded-xl overflow-hidden border-2 ${rarityColor[card.rarity] ?? 'border-gray-700'} bg-gray-900 flex flex-col`}>
              {card.image_uri ? (
                <CardImage src={card.image_uri} alt={card.card_name} className="w-full" />
              ) : (
                <div className="aspect-[2/3] bg-gray-800 flex items-center justify-center text-xs text-gray-400 p-2 text-center">
                  {card.card_name}
                </div>
              )}

              <div className="p-3 flex-1 flex flex-col gap-2">
                <p className="text-xs font-semibold truncate">{card.card_name}</p>

                {/* Bid info */}
                <div className="text-xs text-gray-400">
                  <span className="text-yellow-400 font-bold">{card.current_bid} JAD</span>
                  {card.current_bidder && (
                    <span className="ml-1">by {card.current_bidder}</span>
                  )}
                </div>

                {card.bid_expires_at && (
                  <div className="text-xs">
                    Wins in: <BidTimer expiresAt={card.bid_expires_at} />
                  </div>
                )}

                {/* Hate count */}
                <div className="flex items-center gap-1 text-xs text-gray-500">
                  <ThumbsDown size={11} />
                  <span>{card.hate_count}/4 hates</span>
                </div>

                {/* Actions */}
                <div className="flex gap-2 mt-auto pt-1">
                  <button
                    onClick={() => bid(card)}
                    disabled={!canBid}
                    className="flex-1 flex items-center justify-center gap-1 bg-yellow-600 hover:bg-yellow-500 disabled:opacity-30 disabled:cursor-not-allowed text-white text-xs font-semibold py-1.5 rounded-lg transition-colors"
                  >
                    <Gavel size={12} />
                    {alreadyBidder ? 'Leading' : `Bid ${bidCost}`}
                  </button>
                  <button
                    onClick={() => hate(card)}
                    disabled={!canHate}
                    className="p-1.5 rounded-lg bg-gray-700 hover:bg-red-700 disabled:opacity-30 disabled:cursor-not-allowed text-gray-300 hover:text-white transition-colors"
                    title="Hate this card"
                  >
                    <ThumbsDown size={14} />
                  </button>
                </div>
              </div>
            </div>
          )
        })}
      </div>

      {cards.length === 0 && (
        <p className="text-center text-gray-500 py-16">Market is being stocked…</p>
      )}
    </div>
  )
}
