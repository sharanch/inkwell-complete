import { useState, useEffect, useCallback } from 'react'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import { blogApi } from '../api/client'
import Navbar from '../components/layout/Navbar'

const POPULAR_TAGS = [
  'technology', 'programming', 'design', 'science', 'health',
  'business', 'ai', 'culture', 'philosophy', 'security',
]

export default function WritePage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const editId = searchParams.get('id')

  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [tags, setTags] = useState([])
  const [tagInput, setTagInput] = useState('')
  const [visibility, setVisibility] = useState('private')
  const [coverUrl, setCoverUrl] = useState('')

  const [loading, setLoading] = useState(!!editId)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [savedAt, setSavedAt] = useState(null)

  // Load existing post when editing
  useEffect(() => {
    if (!editId) return
    blogApi.getPost(editId)
      .then(({ data }) => {
        setTitle(data.title || '')
        setBody(data.body || '')
        setTags(data.tags || [])
        setVisibility(data.visibility || 'private')
        setCoverUrl(data.cover_url || '')
      })
      .catch(() => setError('Failed to load post.'))
      .finally(() => setLoading(false))
  }, [editId])

  const save = useCallback(async (vis = visibility) => {
    if (!title.trim()) { setError('Title is required.'); return }
    setSaving(true)
    setError('')
    const payload = {
      title: title.trim(),
      body,
      tags,
      visibility: vis,
      cover_url: coverUrl,
    }
    try {
      if (editId) {
        await blogApi.updatePost(editId, payload)
      } else {
        const { data } = await blogApi.createPost(payload)
        // Redirect to edit URL so subsequent saves update the same post
        navigate(`/write?id=${data.id}`, { replace: true })
      }
      setSavedAt(new Date())
      setVisibility(vis)
    } catch (err) {
      setError(err.response?.data?.error || 'Failed to save. Try again.')
    } finally {
      setSaving(false)
    }
  }, [title, body, tags, visibility, coverUrl, editId, navigate])

  const handlePublish = () => save('public')
  const handleSaveDraft = () => save('private')

  const addTag = (tag) => {
    const t = tag.trim().toLowerCase().replace(/\s+/g, '-')
    if (t && !tags.includes(t) && tags.length < 5) {
      setTags(prev => [...prev, t])
    }
    setTagInput('')
  }

  const removeTag = (tag) => setTags(prev => prev.filter(t => t !== tag))

  const handleTagKeyDown = (e) => {
    if (e.key === 'Enter' || e.key === ',') {
      e.preventDefault()
      addTag(tagInput)
    } else if (e.key === 'Backspace' && !tagInput && tags.length > 0) {
      removeTag(tags[tags.length - 1])
    }
  }

  const wordCount = body.trim() ? body.trim().split(/\s+/).length : 0
  const readingMins = Math.max(1, Math.round(wordCount / 200))

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-gray-950">
        <Navbar />
        <div className="max-w-2xl mx-auto px-4 py-16 flex justify-center">
          <div className="text-gray-400">Loading…</div>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950">
      <Navbar />
      <main className="max-w-2xl mx-auto px-4 py-8">

        {/* Top bar */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-3">
            <Link
              to="/my-posts"
              className="text-sm text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 transition-colors"
            >
              ← My posts
            </Link>
            {savedAt && (
              <span className="text-xs text-gray-400 dark:text-gray-600">
                Saved {savedAt.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
              </span>
            )}
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={handleSaveDraft}
              disabled={saving}
              className="px-3 py-1.5 text-sm text-gray-600 dark:text-gray-400
                border border-gray-200 dark:border-gray-700 rounded-lg
                hover:border-gray-400 dark:hover:border-gray-500
                disabled:opacity-50 transition-colors"
            >
              {saving && visibility === 'private' ? 'Saving…' : 'Save draft'}
            </button>
            <button
              onClick={handlePublish}
              disabled={saving}
              className="px-3 py-1.5 text-sm font-medium
                bg-gray-900 dark:bg-white text-white dark:text-gray-900
                rounded-lg hover:opacity-90 disabled:opacity-50 transition-opacity"
            >
              {saving && visibility === 'public' ? 'Publishing…' : (
                editId && visibility === 'public' ? 'Update' : 'Publish'
              )}
            </button>
          </div>
        </div>

        {error && (
          <div className="mb-4 px-4 py-3 bg-red-50 dark:bg-red-900/20
            border border-red-100 dark:border-red-800 rounded-lg
            text-sm text-red-600 dark:text-red-400">
            {error}
          </div>
        )}

        <div className="bg-white dark:bg-gray-900 rounded-xl
          border border-gray-100 dark:border-gray-800 overflow-hidden">

          {/* Cover URL */}
          <div className="px-6 pt-5">
            {coverUrl ? (
              <div className="relative mb-4 group">
                <img
                  src={coverUrl}
                  alt="Cover"
                  className="w-full h-48 object-cover rounded-lg"
                  onError={() => setCoverUrl('')}
                />
                <button
                  onClick={() => setCoverUrl('')}
                  className="absolute top-2 right-2 p-1.5 bg-black/50 text-white rounded-lg
                    opacity-0 group-hover:opacity-100 transition-opacity text-xs"
                >
                  Remove
                </button>
              </div>
            ) : (
              <div className="mb-4">
                <input
                  type="url"
                  value={coverUrl}
                  onChange={e => setCoverUrl(e.target.value)}
                  placeholder="Cover image URL (optional)"
                  className="w-full text-sm text-gray-400 dark:text-gray-600
                    placeholder-gray-300 dark:placeholder-gray-700
                    bg-transparent border-none outline-none"
                />
              </div>
            )}
          </div>

          {/* Title */}
          <div className="px-6">
            <textarea
              value={title}
              onChange={e => setTitle(e.target.value)}
              placeholder="Post title"
              rows={2}
              className="w-full text-2xl font-bold text-gray-900 dark:text-white
                placeholder-gray-200 dark:placeholder-gray-800
                bg-transparent border-none outline-none resize-none leading-tight"
            />
          </div>

          <div className="mx-6 border-t border-gray-100 dark:border-gray-800 my-2" />

          {/* Body */}
          <div className="px-6 pb-6">
            <textarea
              value={body}
              onChange={e => setBody(e.target.value)}
              placeholder="Write your story…"
              rows={18}
              className="w-full text-sm text-gray-700 dark:text-gray-300
                placeholder-gray-300 dark:placeholder-gray-700
                bg-transparent border-none outline-none resize-none leading-relaxed"
            />
          </div>
        </div>

        {/* Tags */}
        <div className="mt-4 bg-white dark:bg-gray-900 rounded-xl
          border border-gray-100 dark:border-gray-800 px-5 py-4">
          <p className="text-xs font-medium text-gray-500 dark:text-gray-400 mb-3">
            Tags <span className="font-normal text-gray-400">({tags.length}/5)</span>
          </p>

          <div className="flex flex-wrap gap-1.5 mb-3">
            {tags.map(tag => (
              <span
                key={tag}
                className="flex items-center gap-1 text-xs
                  bg-gray-100 dark:bg-gray-800
                  text-gray-700 dark:text-gray-300
                  px-2.5 py-1 rounded-full"
              >
                {tag}
                <button
                  onClick={() => removeTag(tag)}
                  className="text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 leading-none"
                >
                  ×
                </button>
              </span>
            ))}
            {tags.length < 5 && (
              <input
                type="text"
                value={tagInput}
                onChange={e => setTagInput(e.target.value)}
                onKeyDown={handleTagKeyDown}
                onBlur={() => tagInput && addTag(tagInput)}
                placeholder="Add a tag…"
                className="text-xs text-gray-600 dark:text-gray-400
                  placeholder-gray-300 dark:placeholder-gray-700
                  bg-transparent border-none outline-none min-w-24"
              />
            )}
          </div>

          {/* Suggested tags */}
          <div className="flex flex-wrap gap-1.5">
            {POPULAR_TAGS.filter(t => !tags.includes(t)).slice(0, 8).map(tag => (
              <button
                key={tag}
                onClick={() => addTag(tag)}
                disabled={tags.length >= 5}
                className="text-xs text-gray-400 dark:text-gray-600
                  hover:text-gray-700 dark:hover:text-gray-300
                  hover:bg-gray-100 dark:hover:bg-gray-800
                  px-2 py-0.5 rounded-full transition-colors disabled:opacity-30"
              >
                + {tag}
              </button>
            ))}
          </div>
        </div>

        {/* Word count */}
        <p className="mt-3 text-xs text-gray-400 dark:text-gray-600 text-right">
          {wordCount} words · {readingMins} min read
        </p>

      </main>
    </div>
  )
}
