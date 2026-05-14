import { useState, useEffect } from 'react'
import { feedApi } from '../../api/client'

const POPULAR_TAGS = [
  'technology', 'programming', 'design', 'science', 'health',
  'business', 'finance', 'culture', 'politics', 'travel',
  'food', 'sport', 'music', 'film', 'philosophy',
  'history', 'education', 'environment', 'ai', 'security',
]

export default function InterestsModal({ onClose, onSave }) {
  const [selected, setSelected] = useState(new Set())
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    feedApi.getInterests()
      .then(({ data }) => setSelected(new Set(data.tags || [])))
      .finally(() => setLoading(false))
  }, [])

  const toggle = (tag) => setSelected(prev => {
    const s = new Set(prev)
    s.has(tag) ? s.delete(tag) : s.add(tag)
    return s
  })

  const handleSave = async () => {
    setSaving(true)
    try {
      await feedApi.setInterests([...selected])
      onSave()
    } catch (err) {
      console.error(err)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/40 dark:bg-black/60">
      <div className="w-full max-w-md bg-white dark:bg-gray-900 rounded-2xl shadow-xl p-6">
        <div className="flex items-center justify-between mb-4">
          <h3 className="font-semibold text-gray-900 dark:text-white">Your interests</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300">
            <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={2}>
              <path d="M18 6L6 18M6 6l12 12" strokeLinecap="round" />
            </svg>
          </button>
        </div>
        <p className="text-sm text-gray-500 dark:text-gray-400 mb-4">
          Select topics to personalize your feed.
        </p>

        {loading ? (
          <div className="h-32 flex items-center justify-center text-gray-400">Loading…</div>
        ) : (
          <div className="flex flex-wrap gap-2 mb-6">
            {POPULAR_TAGS.map(tag => (
              <button
                key={tag}
                onClick={() => toggle(tag)}
                className={`px-3 py-1.5 rounded-full text-sm font-medium transition-all ${
                  selected.has(tag)
                    ? 'bg-gray-900 dark:bg-white text-white dark:text-gray-900'
                    : 'bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 hover:bg-gray-200 dark:hover:bg-gray-700'
                }`}
              >
                {tag}
              </button>
            ))}
          </div>
        )}

        <div className="flex gap-2">
          <button
            onClick={onClose}
            className="flex-1 py-2 text-sm text-gray-600 dark:text-gray-400 border border-gray-200 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex-1 py-2 text-sm font-medium bg-gray-900 dark:bg-white text-white dark:text-gray-900 rounded-lg hover:opacity-90 disabled:opacity-50 transition-opacity"
          >
            {saving ? 'Saving…' : 'Save interests'}
          </button>
        </div>
      </div>
    </div>
  )
}
