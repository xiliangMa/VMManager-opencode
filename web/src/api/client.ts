import axios from 'axios'

const API_URL = import.meta.env.VITE_API_URL || '/api/v1'

const client = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json'
  }
})

client.interceptors.request.use((config) => {
  const token = localStorage.getItem('auth-storage')
    ? JSON.parse(localStorage.getItem('auth-storage') || '{}')?.state?.token
    : null
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

export { client }

export interface VM {
  id: string
  name: string
  description?: string
  status: string
  cpu_allocated: number
  memory_allocated: number
  disk_allocated: number
  ip_address?: string
  mac_address?: string
  vnc_port?: number
  template_id?: string
  owner_id: string
  boot_order?: string
  autostart?: boolean
  created_at: string
  updated_at: string
}

export interface CreateVMRequest {
  name: string
  description?: string
  template_id: string
  cpu: number
  memory: number
  disk: number
  boot_order?: string
  autostart?: boolean
}

export interface Template {
  id: string
  name: string
  description?: string
  os_type: string
  os_version?: string
  architecture: string
  format: string
  cpu_min: number
  cpu_max: number
  memory_min: number
  memory_max: number
  disk_min: number
  disk_max: number
  template_path: string
  icon_url?: string
  disk_size: number
  is_public: boolean
  is_active: boolean
  downloads: number
  created_at: string
}

export interface User {
  id: string
  username: string
  email: string
  role: string
  is_active: boolean
  quota_cpu: number
  quota_memory: number
  quota_disk: number
  quota_vm_count: number
  created_at: string
}

export const vmsApi = {
  list: (params?: { page?: number; page_size?: number; status?: string; search?: string }) =>
    client.get('/vms', { params }).then(res => res.data),

  get: (id: string) =>
    client.get(`/vms/${id}`).then(res => res.data),

  create: (data: CreateVMRequest) =>
    client.post('/vms', data).then(res => res.data),

  update: (id: string, data: Partial<VM>) =>
    client.put(`/vms/${id}`, data).then(res => res.data),

  delete: (id: string) =>
    client.delete(`/vms/${id}`).then(res => res.data),

  start: (id: string) =>
    client.post(`/vms/${id}/start`).then(res => res.data),

  stop: (id: string) =>
    client.post(`/vms/${id}/stop`).then(res => res.data),

  forceStop: (id: string) =>
    client.post(`/vms/${id}/force-stop`).then(res => res.data),

  restart: (id: string) =>
    client.post(`/vms/${id}/restart`).then(res => res.data),

  suspend: (id: string) =>
    client.post(`/vms/${id}/suspend`).then(res => res.data),

  resume: (id: string) =>
    client.post(`/vms/${id}/resume`).then(res => res.data),

  getConsole: (id: string) =>
    client.get(`/vms/${id}/console`).then(res => res.data),

  getStats: (id: string) =>
    client.get(`/vms/${id}/stats`).then(res => res.data)
}

export const templatesApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    client.get('/templates', { params }).then(res => res.data),

  get: (id: string) =>
    client.get(`/templates/${id}`).then(res => res.data),

  create: (data: Partial<Template>) =>
    client.post('/templates', data).then(res => res.data),

  update: (id: string, data: Partial<Template>) =>
    client.put(`/templates/${id}`, data).then(res => res.data),

  delete: (id: string) =>
    client.delete(`/templates/${id}`).then(res => res.data),

  initUpload: (data: {
    name: string
    description?: string
    file_name: string
    file_size: number
    format: string
    architecture?: string
    chunk_size: number
  }) =>
    client.post('/templates/upload/init', data).then(res => res.data),

  uploadPart: (uploadId: string, chunkIndex: number, totalChunks: number, formData: FormData) =>
    client.post(`/templates/upload/part?upload_id=${uploadId}&chunk_index=${chunkIndex}&total_chunks=${totalChunks}`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    }).then(res => res.data),

  completeUpload: (uploadId: string, data: { 
    total_chunks: number; 
    checksum?: string;
    name: string;
    description?: string;
    os_type: string;
    os_version?: string;
    architecture?: string;
    format?: string;
    cpu_min?: number;
    cpu_max?: number;
    memory_min?: number;
    memory_max?: number;
    disk_min?: number;
    disk_max?: number;
    is_public?: boolean;
  }) =>
    client.post(`/templates/upload/complete/${uploadId}`, data).then(res => res.data),

  getUploadStatus: (uploadId: string) =>
    client.get(`/templates/upload/${uploadId}/status`).then(res => res.data),

  abortUpload: (uploadId: string) =>
    client.delete(`/templates/upload/${uploadId}`).then(res => res.data)
}

export const usersApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    client.get('/admin/users', { params }).then(res => res.data),

  get: (id: string) =>
    client.get(`/admin/users/${id}`).then(res => res.data),

  create: (data: Partial<User>) =>
    client.post('/admin/users', data).then(res => res.data),

  update: (id: string, data: Partial<User>) =>
    client.put(`/admin/users/${id}`, data).then(res => res.data),

  delete: (id: string) =>
    client.delete(`/admin/users/${id}`).then(res => res.data),

  updateQuota: (id: string, quota: Partial<User>) =>
    client.put(`/admin/users/${id}/quota`, quota).then(res => res.data),

  updateRole: (id: string, role: { role: string }) =>
    client.put(`/admin/users/${id}/role`, role).then(res => res.data)
}

export interface AuthProfile {
  email?: string
  language?: string
  timezone?: string
  password?: string
}

export const authApi = {
  getProfile: () =>
    client.get('/auth/profile').then(res => res.data),

  updateProfile: (data: AuthProfile) =>
    client.put('/auth/profile', data).then(res => res.data)
}

export const systemApi = {
  getInfo: () =>
    client.get('/admin/system/info').then(res => res.data),

  getStats: () =>
    client.get('/admin/system/stats').then(res => res.data),

  getAuditLogs: () =>
    client.get('/admin/audit-logs').then(res => res.data)
}

export interface DataPoint {
  timestamp: string
  value: number
}

export interface VMResourceStats {
  cpuUsage: number
  memoryUsage: number
  diskUsage: number
  networkIn: number
  networkOut: number
  cpuHistory: DataPoint[]
  memoryHistory: DataPoint[]
  diskHistory: DataPoint[]
}

export interface SystemResourceStats {
  totalCpu: number
  usedCpu: number
  cpuPercent: number
  totalMemory: number
  usedMemory: number
  memoryPercent: number
  totalDisk: number
  usedDisk: number
  diskPercent: number
  vmCount: number
  runningVmCount: number
  activeUsers: number
}

export const statsApi = {
  getVMStats: (id: string) =>
    client.get(`/vms/${id}/stats`).then(res => res.data),

  getVMHistory: (id: string) =>
    client.get(`/vms/${id}/history`).then(res => res.data),

  getSystemStats: () =>
    client.get('/admin/system/stats').then(res => res.data)
}

export interface AlertRule {
  id: string
  name: string
  description?: string
  metric: string
  condition: string
  threshold: number
  duration: number
  severity: string
  enabled: boolean
  notifyChannels: string[]
  notifyUsers?: string[]
  vmIds?: string[]
  isGlobal: boolean
  created_by?: string
  created_at?: string
  updated_at?: string
}

export const alertRulesApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    client.get('/admin/alert-rules', { params }).then(res => res.data),

  get: (id: string) =>
    client.get(`/admin/alert-rules/${id}`).then(res => res.data),

  create: (data: Partial<AlertRule>) =>
    client.post('/admin/alert-rules', data).then(res => res.data),

  update: (id: string, data: Partial<AlertRule>) =>
    client.put(`/admin/alert-rules/${id}`, data).then(res => res.data),

  delete: (id: string) =>
    client.delete(`/admin/alert-rules/${id}`).then(res => res.data),

  toggle: (id: string) =>
    client.post(`/admin/alert-rules/${id}/toggle`).then(res => res.data),

  getStats: () =>
    client.get('/admin/alert-rules/stats/summary').then(res => res.data)
}

export interface VMSnapshot {
  id: string
  name: string
  description?: string
  vm_id: string
  state: 'running' | 'shutdown' | 'disk-only'
  size: number
  created_at: string
  updated_at: string
}

export const snapshotsApi = {
  list: (vmId: string) =>
    client.get(`/vms/${vmId}/snapshots`).then(res => res.data),

  get: (vmId: string, name: string) =>
    client.get(`/vms/${vmId}/snapshots/${name}`).then(res => res.data),

  create: (vmId: string, data: { name: string; description?: string }) =>
    client.post(`/vms/${vmId}/snapshots`, data).then(res => res.data),

  restore: (vmId: string, name: string) =>
    client.post(`/vms/${vmId}/snapshots/${name}/restore`, { name }).then(res => res.data),

  delete: (vmId: string, name: string) =>
    client.delete(`/vms/${vmId}/snapshots/${name}`).then(res => res.data)
}
