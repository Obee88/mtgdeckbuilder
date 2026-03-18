import { useEffect, useState, useRef } from 'react'
import api from '../api'
import toast from 'react-hot-toast'
import { Plus, Trash2, X, List } from 'lucide-react'

interface CardList {
  id: string
  name: string
  card_names: string[]
}

function CardAutocomplete({ onAdd }: { onAdd: (name: string) => void }) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<string[]>([])
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    if (!open) return
    function handler(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  function handleChange(val: string) {
    setQuery(val)
    if (debounceRef.current) clearTimeout(debounceRef.current)
    if (!val.trim()) { setResults([]); setOpen(false); return }
    debounceRef.current = setTimeout(async () => {
      try {
        const res = await api.get('/cards/search', { params: { q: val } })
        // Deduplicate by name
        const seen = new Set<string>()
        const names: string[] = []
        for (const card of res.data) {
          if (!seen.has(card.name)) {
            seen.add(card.name)
            names.push(card.name)
          }
        }
        setResults(names)
        setOpen(names.length > 0)
      } catch {
        // ignore
      }
    }, 250)
  }

  function pick(name: string) {
    onAdd(name)
    setQuery('')
    setResults([])
    setOpen(false)
  }

  return (
    <div className="relative" ref={ref}>
      <div className="flex gap-2">
        <input
          type="text"
          placeholder="Search card name…"
          value={query}
          onChange={(e) => handleChange(e.target.value)}
          onFocus={() => results.length > 0 && setOpen(true)}
          className="bg-gray-800 border border-gray-600 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-purple-500 flex-1"
        />
      </div>
      {open && (
        <div className="absolute top-full mt-1 left-0 right-0 z-20 bg-gray-800 border border-gray-600 rounded-xl shadow-xl max-h-56 overflow-y-auto">
          {results.map((name) => (
            <button
              key={name}
              onMouseDown={(e) => { e.preventDefault(); pick(name) }}
              className="w-full text-left px-4 py-2 text-sm hover:bg-gray-700 truncate"
            >
              {name}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

export default function Lists() {
  const [lists, setLists] = useState<CardList[]>([])
  const [selected, setSelected] = useState<CardList | null>(null)
  const [newName, setNewName] = useState('')
  const [creating, setCreating] = useState(false)

  async function fetchLists() {
    const res = await api.get('/lists')
    setLists(res.data)
  }

  useEffect(() => { fetchLists() }, [])

  // Keep selected in sync after list updates
  useEffect(() => {
    if (selected) {
      const updated = lists.find((l) => l.id === selected.id)
      if (updated) setSelected(updated)
    }
  }, [lists])

  async function createList() {
    if (!newName.trim()) return
    try {
      const res = await api.post('/lists', { name: newName.trim() })
      setLists((prev) => [...prev, res.data])
      setNewName('')
      setCreating(false)
      setSelected(res.data)
      toast.success(`List "${res.data.name}" created`)
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to create list')
    }
  }

  async function deleteList(list: CardList) {
    try {
      await api.delete(`/lists/${list.id}`)
      setLists((prev) => prev.filter((l) => l.id !== list.id))
      if (selected?.id === list.id) setSelected(null)
      toast.success(`List "${list.name}" deleted`)
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to delete')
    }
  }

  async function addCard(name: string) {
    if (!selected) return
    try {
      const res = await api.post(`/lists/${selected.id}/cards`, { card_name: name })
      // res.data is the updated list
      setLists((prev) => prev.map((l) => l.id === selected.id ? res.data : l))
      toast.success(`"${name}" added to list`)
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to add card')
    }
  }

  async function removeCard(name: string) {
    if (!selected) return
    try {
      await api.delete(`/lists/${selected.id}/cards`, { data: { card_name: name } })
      setLists((prev) => prev.map((l) =>
        l.id === selected.id
          ? { ...l, card_names: l.card_names.filter((n) => n !== name) }
          : l
      ))
    } catch (err: any) {
      toast.error(err.response?.data?.error || 'Failed to remove card')
    }
  }

  return (
    <div className="flex gap-6 h-full">
      {/* Sidebar */}
      <div className="w-64 flex-shrink-0">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-bold">Lists</h2>
          <button
            onClick={() => setCreating((v) => !v)}
            className="p-1.5 rounded-lg hover:bg-gray-700 text-gray-400 hover:text-white transition-colors"
            title="New list"
          >
            <Plus size={18} />
          </button>
        </div>

        {creating && (
          <div className="mb-3 flex gap-2">
            <input
              autoFocus
              type="text"
              placeholder="List name…"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter') createList(); if (e.key === 'Escape') setCreating(false) }}
              className="bg-gray-800 border border-gray-600 rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:border-purple-500 flex-1 min-w-0"
            />
            <button onClick={createList} className="px-3 py-1.5 bg-purple-600 hover:bg-purple-700 rounded-lg text-sm font-medium">
              Add
            </button>
          </div>
        )}

        {lists.length === 0 && !creating && (
          <p className="text-sm text-gray-500">No lists yet. Click + to create one.</p>
        )}

        <div className="space-y-1">
          {lists.map((list) => (
            <div
              key={list.id}
              onClick={() => setSelected(list)}
              className={`flex items-center gap-2 px-3 py-2.5 rounded-lg cursor-pointer group transition-colors ${
                selected?.id === list.id
                  ? 'bg-purple-700 text-white'
                  : 'text-gray-300 hover:bg-gray-800'
              }`}
            >
              <List size={15} className="flex-shrink-0" />
              <span className="text-sm font-medium flex-1 truncate">{list.name}</span>
              <span className="text-xs opacity-60">{list.card_names.length}</span>
              <button
                onClick={(e) => { e.stopPropagation(); deleteList(list) }}
                className="opacity-0 group-hover:opacity-100 p-0.5 rounded hover:text-red-400 transition-all"
              >
                <Trash2 size={13} />
              </button>
            </div>
          ))}
        </div>
      </div>

      {/* Main panel */}
      <div className="flex-1 min-w-0">
        {!selected ? (
          <div className="flex items-center justify-center h-64 text-gray-500 text-sm">
            Select a list to view its cards
          </div>
        ) : (
          <div>
            <h3 className="text-lg font-semibold mb-4">{selected.name}</h3>

            <div className="mb-4 max-w-sm">
              <CardAutocomplete onAdd={addCard} />
            </div>

            {selected.card_names.length === 0 ? (
              <p className="text-sm text-gray-500">No cards in this list yet. Search above to add some.</p>
            ) : (
              <div className="space-y-1 max-w-sm">
                {selected.card_names.map((name) => (
                  <div
                    key={name}
                    className="flex items-center gap-3 bg-gray-900 border border-gray-700 rounded-lg px-4 py-2.5"
                  >
                    <span className="text-sm flex-1">{name}</span>
                    <button
                      onClick={() => removeCard(name)}
                      className="p-1 rounded hover:bg-gray-700 text-gray-500 hover:text-red-400 transition-colors"
                    >
                      <X size={14} />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
