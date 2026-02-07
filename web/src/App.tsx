import React from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from './stores/authStore'
import MainLayout from './components/Layout/MainLayout'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import VMs from './pages/VMs/List'
import VMCreate from './pages/VMs/Create'
import VMDetail from './pages/VMs/Detail'
import VMConsole from './pages/VMs/Console'
import Templates from './pages/Templates/List'
import TemplateUpload from './pages/Templates/Upload'
import UserManagement from './pages/Admin/Users'
import AuditLogs from './pages/Admin/AuditLogs'
import Profile from './pages/Settings/Profile'

const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuthStore()
  const { t } = useTranslation()

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}

const App: React.FC = () => {
  const { t } = useTranslation()

  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <MainLayout />
          </ProtectedRoute>
        }
      >
        <Route index element={<Dashboard />} />
        <Route path="vms" element={<VMs />} />
        <Route path="vms/create" element={<VMCreate />} />
        <Route path="vms/:id" element={<VMDetail />} />
        <Route path="vms/:id/console" element={<VMConsole />} />
        <Route path="templates" element={<Templates />} />
        <Route path="templates/upload" element={<TemplateUpload />} />
        <Route path="admin/users" element={<UserManagement />} />
        <Route path="admin/audit-logs" element={<AuditLogs />} />
        <Route path="settings/profile" element={<Profile />} />
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default App
