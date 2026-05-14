import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { feedApi } from '../api/client'
import PostCard from '../components/blog/PostCard'
import InterestsModal from '../components/feed/InterestsModal'
import Navbar from '../components/layout/Navbar'

export default function FeedPage() {
  const [posts, setPosts] = useState([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [showInterests, setShowInterests] = useState(false)

  const loadFeed = async (p = 1) => {
    setLoading(true)
    try {
      const { data } = await feedApi.getFeed(p)
      setPosts(prev => p === 1 ? data.posts : [...prev, ...data.posts])
      setPage(p)
    } catch (err) {
      console.error(err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { loadFeed(1) }, [])

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950">
      <Navbar />
      <main className="max-w-2xl mx-auto px-4 py-8">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white">Your Feed</h2>
          <button
            onClick={() => setShowInterests(true)}
            className="text-sm text-gray-500 dark:text-gray-400 hover:text-gray-900 dark:hover:text-white
              border border-gray-200 dark:border-gray-700 px-3 py-1.5 rounded-lg transition-colors"
          >
            Customize interests
          </button>
        </div>

        {loading && posts.length === 0 ? (
          <div className="space-y-4">
            {[1,2,3].map(i => <PostCardSkeleton key={i} />)}
          </div>
        ) : posts.length === 0 ? (
          <div className="text-center py-16">
            <p className="text-gray-400 dark:text-gray-600 mb-4">Your feed is empty.</p>
            <button
              onClick={() => setShowInterests(true)}
              className="text-sm font-medium text-gray-900 dark:text-white underline"
            >
              Set your interests
            </button>
          </div>
        ) : (
          <div className="space-y-6">
            {posts.map(post => <PostCard key={post.id} post={post} />)}
            <button
              onClick={() => loadFeed(page + 1)}
              disabled={loading}
              className="w-full py-3 text-sm text-gray-500 dark:text-gray-400
                hover:text-gray-900 dark:hover:text-white transition-colors disabled:opacity-50"
            >
              {loading ? 'Loading…' : 'Load more'}
            </button>
          </div>
        )}
      </main>

      {showInterests && (
        <InterestsModal
          onClose={() => setShowInterests(false)}
          onSave={() => { setShowInterests(false); loadFeed(1) }}
        />
      )}
    </div>
  )
}

function PostCardSkeleton() {
  return (
    <div className="animate-pulse bg-white dark:bg-gray-900 rounded-xl p-5 border border-gray-100 dark:border-gray-800">
      <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-1/4 mb-3" />
      <div className="h-6 bg-gray-200 dark:bg-gray-700 rounded w-3/4 mb-2" />
      <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-full mb-1" />
      <div className="h-4 bg-gray-200 dark:bg-gray-700 rounded w-2/3" />
    </div>
  )
}
