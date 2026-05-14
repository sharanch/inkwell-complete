import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider, useAuth } from './context/AuthContext'
import { ThemeProvider } from './context/ThemeContext'
import LoginPage from './pages/LoginPage'
import FeedPage from './pages/FeedPage'
import WritePage from './pages/WritePage'
import MyPostsPage from './pages/MyPostsPage'
import PostDetailPage from './pages/PostDetailPage'

function ProtectedRoute({ children }) {
  const { isAuthenticated } = useAuth()
  return isAuthenticated ? children : <Navigate to="/login" replace />
}

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/feed" element={<ProtectedRoute><FeedPage /></ProtectedRoute>} />
            <Route path="/write" element={<ProtectedRoute><WritePage /></ProtectedRoute>} />
            <Route path="/my-posts" element={<ProtectedRoute><MyPostsPage /></ProtectedRoute>} />
            <Route path="/post/:id" element={<ProtectedRoute><PostDetailPage /></ProtectedRoute>} />
            <Route path="*" element={<Navigate to="/feed" replace />} />
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </ThemeProvider>
  )
}

export default App
