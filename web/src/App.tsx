import React from 'react'
import { Routes, Route, Navigate } from 'react-router-dom'
import { useAuthStore } from './stores/authStore'
import MainLayout from './components/Layout/MainLayout'
import Login from './pages/Login'
import Dashboard from './pages/Dashboard'
import VMs from './pages/VMs/List'
import VMCreate from './pages/VMs/Create'
import VMDetail from './pages/VMs/Detail'
import VMEdit from './pages/VMs/Edit'
import VMConsole from './pages/VMs/Console'
import Monitor from './pages/VMs/Monitor'
import VMSnapshots from './pages/VMs/Snapshots'
import Templates from './pages/Templates/List'
import TemplateUpload from './pages/Templates/Upload'
import TemplateEdit from './pages/Templates/Edit'
import UserManagement from './pages/Admin/Users'
import AuditLogs from './pages/Admin/AuditLogs'
import AlertRules from './pages/Admin/AlertRules'
import AlertHistory from './pages/Admin/AlertHistory'
import SystemDashboard from './pages/Dashboard/SystemDashboard'
import Profile from './pages/Settings/Profile'

const ProtectedRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuthStore()

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return <>{children}</>
}

const App: React.FC = () => {

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
        <Route path="dashboard" element={<SystemDashboard />} />
        <Route path="vms" element={<VMs />} />
        <Route path="vms/create" element={<VMCreate />} />
        <Route path="vms/:id/edit" element={<VMEdit />} />
        <Route path="vms/:id/console" element={<VMConsole />} />
        <Route path="vms/:id/monitor" element={<Monitor />} />
        <Route path="vms/:id/snapshots" element={<VMSnapshots />} />
        <Route path="vms/:id" element={<VMDetail />} />
        <Route path="templates" element={<Templates />} />
        <Route path="templates/upload" element={<TemplateUpload />} />
        <Route path="templates/:id/edit" element={<TemplateEdit />} />
        <Route path="admin/users" element={<UserManagement />} />
        <Route path="admin/audit-logs" element={<AuditLogs />} />
        <Route path="admin/alerts" element={<AlertRules />} />
        <Route path="admin/alert-history" element={<AlertHistory />} />
        <Route path="settings/profile" element={<Profile />} />
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default App
