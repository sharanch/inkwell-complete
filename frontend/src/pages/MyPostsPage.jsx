import { useState, useEffect } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { blogApi } from '../api/client'
import Navbar from '../components/layout/Navbar'

export default function MyPostsPage() {
  const [posts, setPosts] = useState([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [hasMore, setHasMore] = useState(true)
  const [deletingId, setDeletingId] = useState(null)
  const navigate = useNavigate()

  const loadPosts = async (p = 1) => {
    setLoading(true)
    try {
      const { data } = await blogApi.getMyPosts(p)
      const incoming = data.posts || []
      setPosts(prev => p === 1 ? incoming : [...prev, ...incoming])
      setPage(p)
      setHasMore(incoming.length === (data.page_size || 20))
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { loadPosts(1) }, [])

  const handleDelete = async (id) => {
    if (!window.confirm('Delete this post? This cannot be undone.')) return
    setDeletingId(id)
    try {
      await blogApi.deletePost(id)
      setPosts(prev => prev.filter(p => p.id !== id))
    } catch (err) {
      console.error(err)
    } finally {
      setDeletingId(null)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950">
      <Navbar />
      <main className="max-w-2xl mx-auto px-4 py-8">

        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white">My posts</h2>
          <Link
            to="/write"
            className="text-sm bg-gray-900 dark:bg-white text-white dark:text-gray-900
              px-3 py-1.5 rounded-lg font-medium hover:opacity-90 transition-opacity"
          >
            + New post
          </Link>
        </div>

        {loading && posts.length === 0 ? (
          <div className="space-y-3">
            {[1, 2, 3].map(i => <RowSkeleton key={i} />)}
          </div>
        ) : posts.length === 0 ? (
          <div className="text-center py-20">
            <p className="text-gray-400 dark:text-gray-600 mb-4">You haven't written anything yet.</p>
            <Link
              to="/write"
              className="text-sm font-medium text-gray-900 dark:text-white underline"
            >
              Write your first post
            </Link>
          </div>
        ) : (
          <div className="space-y-2">
            {posts.map(post => (
              <PostRow
                key={post.id}
                post={post}
                onEdit={() => navigate(`/write?id=${post.id}`)}
                onDelete={() => handleDelete(post.id)}
                deleting={deletingId === post.id}
              />
            ))}

            {hasMore && (
              <button
                onClick={() => loadPosts(page + 1)}
                disabled={loading}
                className="w-full py-3 text-sm text-gray-400 hover:text-gray-700
                  dark:hover:text-gray-300 transition-colors disabled:opacity-50"
              >
                {loading ? 'Loading…' : 'Load more'}
              </button>
            )}
          </div>
        )}
      </main>
    </div>
  )
}

function PostRow({ post, onEdit, onDelete, deleting }) {
  const date = post.published_at
    ? new Date(post.published_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
    : null

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl px-5 py-4
      border border-gray-100 dark:border-gray-800
      flex items-start justify-between gap-4">

      <div className="min-w-0">
        <div className="flex items-center gap-2 mb-0.5">
          <span className={`text-xs px-2 py-0.5 rounded-full font-medium ${
            post.visibility === 'public'
              ? 'bg-green-50 dark:bg-green-900/30 text-green-600 dark:text-green-400'
              : 'bg-gray-100 dark:bg-gray-800 text-gray-400'
          }`}>
            {post.visibility === 'public' ? 'Public' : 'Draft'}
          </span>
          {date && (
            <span className="text-xs text-gray-400 dark:text-gray-600">{date}</span>
          )}
        </div>

        <Link
          to={`/post/${post.id}`}
          className="block text-sm font-medium text-gray-900 dark:text-white
            hover:underline truncate"
        >
          {post.title || 'Untitled'}
        </Link>

        <div className="flex items-center gap-3 mt-1 text-xs text-gray-400">
          <span>{post.views_count || 0} views</span>
          <span>{post.likes_count || 0} likes</span>
          <span>{post.reading_mins || 1} min read</span>
        </div>
      </div>

      <div className="flex items-center gap-1 shrink-0">
        <button
          onClick={onEdit}
          className="px-3 py-1.5 text-xs text-gray-500 dark:text-gray-400
            hover:text-gray-900 dark:hover:text-white
            border border-gray-200 dark:border-gray-700 rounded-lg
            hover:border-gray-400 dark:hover:border-gray-500 transition-colors"
        >
          Edit
        </button>
        <button
          onClick={onDelete}
          disabled={deleting}
          className="px-3 py-1.5 text-xs text-red-400
            hover:text-red-600 border border-transparent
            hover:border-red-200 dark:hover:border-red-900 rounded-lg
            transition-colors disabled:opacity-40"
        >
          {deleting ? '…' : 'Delete'}
        </button>
      </div>
    </div>
  )
}

function RowSkeleton() {
  return (
    <div className="animate-pulse bg-white dark:bg-gray-900 rounded-xl px-5 py-4
      border border-gray-100 dark:border-gray-800 flex items-center justify-between">
      <div className="space-y-2 flex-1">
        <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-16" />
        <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-64" />
        <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded w-32" />
      </div>
      <div className="h-7 w-24 bg-gray-200 dark:bg-gray-700 rounded-lg" />
    </div>
  )
}
