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
import VMInstallation from './pages/VMs/Installation'
import Templates from './pages/Templates/List'
import TemplateUpload from './pages/Templates/Upload'
import TemplateEdit from './pages/Templates/Edit'
import ISOs from './pages/ISO/List'
import ISOUpload from './pages/ISO/Upload'
import UserManagement from './pages/Admin/Users'
import AuditLogs from './pages/Admin/AuditLogs'
import AlertRules from './pages/Admin/AlertRules'
import AlertHistory from './pages/Admin/AlertHistory'
import VirtualNetworks from './pages/Admin/VirtualNetworks'
import StoragePools from './pages/Admin/StoragePools'
import Profile from './pages/Settings'
import { VMStatusProvider } from './context/VMStatusContext'

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
            <VMStatusProvider>
              <MainLayout />
            </VMStatusProvider>
          </ProtectedRoute>
        }
      >
        <Route index element={<Dashboard />} />
        <Route path="vms" element={<VMs />} />
        <Route path="vms/create" element={<VMCreate />} />
        <Route path="vms/:id/edit" element={<VMEdit />} />
        <Route path="vms/:id/console" element={<VMConsole />} />
        <Route path="vms/:id/monitor" element={<Monitor />} />
        <Route path="vms/:id/snapshots" element={<VMSnapshots />} />
        <Route path="vms/:id/installation" element={<VMInstallation />} />
        <Route path="vms/:id" element={<VMDetail />} />
        <Route path="templates" element={<Templates />} />
        <Route path="templates/upload" element={<TemplateUpload />} />
        <Route path="templates/:id/edit" element={<TemplateEdit />} />
        <Route path="isos" element={<ISOs />} />
        <Route path="isos/upload" element={<ISOUpload />} />
        <Route path="admin/users" element={<UserManagement />} />
        <Route path="admin/audit-logs" element={<AuditLogs />} />
        <Route path="admin/alerts" element={<AlertRules />} />
        <Route path="admin/alert-history" element={<AlertHistory />} />
        <Route path="admin/networks" element={<VirtualNetworks />} />
        <Route path="admin/storage" element={<StoragePools />} />
        <Route path="settings/profile" element={<Profile />} />
      </Route>

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default App
