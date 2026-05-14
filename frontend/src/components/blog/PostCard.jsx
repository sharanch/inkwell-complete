import { Link } from 'react-router-dom'
import { blogApi } from '../../api/client'
import { useState } from 'react'

export default function PostCard({ post }) {
  const [likes, setLikes] = useState(post.likes_count || 0)
  const [liked, setLiked] = useState(false)

  const handleLike = async (e) => {
    e.preventDefault()
    try {
      const { data } = await blogApi.toggleLike(post.id)
      setLiked(data.liked)
      setLikes(l => data.liked ? l + 1 : l - 1)
    } catch {}
  }

  const date = post.published_at
    ? new Date(post.published_at).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
    : 'Draft'

  return (
    <Link to={`/post/${post.id}`} className="block group">
      <article className="bg-white dark:bg-gray-900 rounded-xl p-5
        border border-gray-100 dark:border-gray-800
        hover:border-gray-300 dark:hover:border-gray-600
        transition-all duration-150">

        {post.cover_url && (
          <img
            src={post.cover_url}
            alt=""
            className="w-full h-40 object-cover rounded-lg mb-4"
          />
        )}

        <div className="flex items-center gap-2 mb-2">
          <div className="w-6 h-6 rounded-full bg-gray-200 dark:bg-gray-700 flex items-center justify-center text-xs font-medium text-gray-600 dark:text-gray-300">
            {(post.author_name || post.author_id || '?')[0].toUpperCase()}
          </div>
          <span className="text-xs text-gray-500 dark:text-gray-400">
            {post.author_name || 'Anonymous'} · {date}
          </span>
        </div>

        <h3 className="font-semibold text-gray-900 dark:text-white text-base leading-snug mb-1 group-hover:underline">
          {post.title}
        </h3>
        {post.excerpt && (
          <p className="text-sm text-gray-500 dark:text-gray-400 line-clamp-2 mb-3">
            {post.excerpt}
          </p>
        )}

        <div className="flex items-center justify-between">
          <div className="flex gap-2 flex-wrap">
            {(post.tags || []).slice(0, 3).map(tag => (
              <span key={tag} className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded-full">
                {tag}
              </span>
            ))}
          </div>
          <div className="flex items-center gap-3 text-xs text-gray-400 dark:text-gray-500">
            <span>{post.reading_mins || 1} min read</span>
            <button
              onClick={handleLike}
              className={`flex items-center gap-1 transition-colors ${liked ? 'text-red-500' : 'hover:text-red-400'}`}
            >
              <svg className="w-4 h-4" fill={liked ? 'currentColor' : 'none'} viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z" />
              </svg>
              {likes}
            </button>
          </div>
        </div>
      </article>
    </Link>
  )
}
