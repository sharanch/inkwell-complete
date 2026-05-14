import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { blogApi } from '../api/client'
import { useAuth } from '../context/AuthContext'
import Navbar from '../components/layout/Navbar'

export default function PostDetailPage() {
  const { id } = useParams()
  const { user } = useAuth()
  const navigate = useNavigate()

  const [post, setPost] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [liked, setLiked] = useState(false)
  const [likes, setLikes] = useState(0)
  const [deleting, setDeleting] = useState(false)

  useEffect(() => {
    blogApi.getPost(id)
      .then(({ data }) => {
        setPost(data)
        setLikes(data.likes_count || 0)
      })
      .catch(() => setError('Post not found or you don\'t have access.'))
      .finally(() => setLoading(false))
  }, [id])

  const handleLike = async () => {
    try {
      const { data } = await blogApi.toggleLike(id)
      setLiked(data.liked)
      setLikes(l => data.liked ? l + 1 : l - 1)
    } catch {}
  }

  const handleDelete = async () => {
    if (!window.confirm('Delete this post? This cannot be undone.')) return
    setDeleting(true)
    try {
      await blogApi.deletePost(id)
      navigate('/my-posts')
    } catch {
      setDeleting(false)
    }
  }

  if (loading) return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950">
      <Navbar />
      <main className="max-w-2xl mx-auto px-4 py-12">
        <div className="animate-pulse space-y-4">
          <div className="h-8 bg-gray-200 dark:bg-gray-800 rounded w-3/4" />
          <div className="h-4 bg-gray-200 dark:bg-gray-800 rounded w-1/4" />
          <div className="space-y-2 pt-6">
            {[...Array(8)].map((_, i) => (
              <div key={i} className="h-4 bg-gray-200 dark:bg-gray-800 rounded" style={{ width: `${75 + Math.random() * 25}%` }} />
            ))}
          </div>
        </div>
      </main>
    </div>
  )

  if (error) return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950">
      <Navbar />
      <main className="max-w-2xl mx-auto px-4 py-20 text-center">
        <p className="text-gray-400 dark:text-gray-600 mb-4">{error}</p>
        <Link to="/feed" className="text-sm font-medium text-gray-900 dark:text-white underline">
          Back to feed
        </Link>
      </main>
    </div>
  )

  const isAuthor = user?.id === post.author_id
  const date = post.published_at
    ? new Date(post.published_at).toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' })
    : new Date(post.created_at).toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' })

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950">
      <Navbar />
      <main className="max-w-2xl mx-auto px-4 py-10">

        {/* Cover image */}
        {post.cover_url && (
          <img
            src={post.cover_url}
            alt=""
            className="w-full h-64 object-cover rounded-xl mb-8"
            onError={e => e.target.style.display = 'none'}
          />
        )}

        {/* Tags */}
        {post.tags?.length > 0 && (
          <div className="flex flex-wrap gap-2 mb-4">
            {post.tags.map(tag => (
              <span key={tag} className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400 px-2.5 py-1 rounded-full">
                {tag}
              </span>
            ))}
          </div>
        )}

        {/* Title */}
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white leading-tight mb-4">
          {post.title}
        </h1>

        {/* Meta row */}
        <div className="flex items-center justify-between mb-8 pb-6 border-b border-gray-100 dark:border-gray-800">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-full bg-gray-200 dark:bg-gray-700 flex items-center justify-center text-xs font-semibold text-gray-600 dark:text-gray-300">
              {(post.author_name || post.author_id || '?')[0].toUpperCase()}
            </div>
            <div>
              <p className="text-sm font-medium text-gray-900 dark:text-white leading-none">
                {post.author_name || 'Anonymous'}
              </p>
              <p className="text-xs text-gray-400 dark:text-gray-600 mt-0.5">
                {date} · {post.reading_mins || 1} min read
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            {/* Like button */}
            <button
              onClick={handleLike}
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-colors ${
                liked
                  ? 'text-red-500 bg-red-50 dark:bg-red-900/20'
                  : 'text-gray-400 hover:text-red-400 hover:bg-gray-100 dark:hover:bg-gray-800'
              }`}
            >
              <svg className="w-4 h-4" fill={liked ? 'currentColor' : 'none'} viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" />
              </svg>
              {likes}
            </button>

            {/* Author actions */}
            {isAuthor && (
              <>
                <Link
                  to={`/write?id=${post.id}`}
                  className="px-3 py-1.5 text-sm text-gray-500 dark:text-gray-400
                    border border-gray-200 dark:border-gray-700 rounded-lg
                    hover:border-gray-400 dark:hover:border-gray-500 transition-colors"
                >
                  Edit
                </Link>
                <button
                  onClick={handleDelete}
                  disabled={deleting}
                  className="px-3 py-1.5 text-sm text-red-400
                    border border-transparent hover:border-red-200 dark:hover:border-red-900
                    rounded-lg transition-colors disabled:opacity-40"
                >
                  {deleting ? '…' : 'Delete'}
                </button>
              </>
            )}
          </div>
        </div>

        {/* Body */}
        <div className="prose prose-gray dark:prose-invert max-w-none
          text-gray-700 dark:text-gray-300 leading-relaxed text-base
          whitespace-pre-wrap">
          {post.body}
        </div>

        {/* Footer */}
        <div className="mt-12 pt-6 border-t border-gray-100 dark:border-gray-800">
          <Link
            to="/feed"
            className="text-sm text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 transition-colors"
          >
            ← Back to feed
          </Link>
        </div>

      </main>
    </div>
  )
}
