import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { authApi } from '../api/client'
import { useAuth } from '../context/AuthContext'

export default function LoginPage() {
  const [step, setStep] = useState('email') // 'email' | 'otp'
  const [email, setEmail] = useState('')
  const [otp, setOtp] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const { login } = useAuth()
  const navigate = useNavigate()

  const handleRequestOTP = async (e) => {
    e.preventDefault()
    if (loading) return   // guard against double-submit
    setError('')
    setLoading(true)
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    try {
      await authApi.requestOTP(email)
      setStep('otp')
    } catch (err) {
      const status = err.response?.status
      if (status === 429) {
        setError('Code already sent — check your email, or wait a minute and try again.')
        setStep('otp') // still advance; a code was already sent
      } else {
        setError(err.response?.data?.error || 'Failed to send code. Try again.')
      }
    } finally {
      setLoading(false)
    }
  }

  const handleVerifyOTP = async (e) => {
    e.preventDefault()
    if (loading) return
    setError('')
    setLoading(true)
    try {
      const { data } = await authApi.verifyOTP(email, otp)
      login(data.user, data.access_token, data.refresh_token)
      navigate('/feed')
    } catch (err) {
      const status = err.response?.status
      if (status === 401) {
        setError('Incorrect or expired code. Try again.')
      } else {
        setError(err.response?.data?.error || 'Something went wrong. Try again.')
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-white dark:bg-gray-950 px-4">
      <div className="w-full max-w-sm">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white tracking-tight">
            inkwell
          </h1>
          <p className="text-gray-500 dark:text-gray-400 mt-1 text-sm">
            Write. Share. Discover.
          </p>
        </div>

        {step === 'email' ? (
          <form onSubmit={handleRequestOTP} className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Email address
              </label>
              <input
                type="email"
                value={email}
                onChange={e => setEmail(e.target.value)}
                placeholder="you@example.com"
                required
                disabled={loading}
                className="w-full px-4 py-2.5 rounded-lg border border-gray-200 dark:border-gray-700
                  bg-white dark:bg-gray-900 text-gray-900 dark:text-white
                  focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-white
                  placeholder-gray-400 dark:placeholder-gray-600 text-sm
                  disabled:opacity-50"
              />
            </div>
            {error && <p className="text-red-500 text-sm">{error}</p>}
            <button
              type="submit"
              disabled={loading}
              className="w-full py-2.5 px-4 bg-gray-900 dark:bg-white text-white dark:text-gray-900
                rounded-lg font-medium text-sm hover:opacity-90 disabled:opacity-50 transition-opacity"
            >
              {loading ? 'Sending…' : 'Continue with email'}
            </button>
            <p className="text-xs text-gray-400 dark:text-gray-500 text-center">
              No password needed. We'll send a one-time code.
            </p>
          </form>
        ) : (
          <form onSubmit={handleVerifyOTP} className="space-y-4">
            <div>
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                We sent a 6-digit code to <strong className="text-gray-900 dark:text-white">{email}</strong>
              </p>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Enter code
              </label>
              <input
                type="text"
                value={otp}
                onChange={e => setOtp(e.target.value.replace(/\D/g, '').slice(0, 6))}
                placeholder="000000"
                required
                autoFocus
                disabled={loading}
                className="w-full px-4 py-2.5 rounded-lg border border-gray-200 dark:border-gray-700
                  bg-white dark:bg-gray-900 text-gray-900 dark:text-white text-center
                  text-2xl tracking-[0.5em] font-mono
                  focus:outline-none focus:ring-2 focus:ring-gray-900 dark:focus:ring-white
                  disabled:opacity-50"
              />
            </div>
            {error && <p className="text-red-500 text-sm">{error}</p>}
            <button
              type="submit"
              disabled={loading || otp.length < 6}
              className="w-full py-2.5 px-4 bg-gray-900 dark:bg-white text-white dark:text-gray-900
                rounded-lg font-medium text-sm hover:opacity-90 disabled:opacity-50 transition-opacity"
            >
              {loading ? 'Verifying…' : 'Sign in'}
            </button>
            <button
              type="button"
              onClick={() => { setStep('email'); setOtp(''); setError('') }}
              className="w-full text-sm text-gray-400 dark:text-gray-500 hover:text-gray-600 dark:hover:text-gray-400 transition-colors"
            >
              Use a different email
            </button>
          </form>
        )}
      </div>
    </div>
  )
}
