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
  templateId?: string
  cpuAllocated: number
  memoryAllocated: number
  diskAllocated: number
  ipAddress?: string
  macAddress?: string
  vncPort?: number
  createdAt: string
  updatedAt?: string
}

export interface VMDetail extends VM {
  template_name?: string
  owner_id?: string
  vnc_port?: number
  mac_address?: string
  is_installed?: boolean
  install_status?: string
  install_progress?: number
  agent_installed?: boolean
}

export interface CreateVMRequest {
  name: string
  description?: string
  template_id: string
  cpu_allocated: number
  memory_allocated: number
  disk_allocated: number
  boot_order?: string
  autostart?: boolean
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

export interface Template {
  id: string
  name: string
  description?: string
  osType: string
  osVersion?: string
  architecture: string
  format: string
  cpuMin: number
  cpuMax: number
  memoryMin: number
  memoryMax: number
  diskMin: number
  diskMax: number
  templatePath: string
  iconUrl?: string
  screenshotUrls?: string[]
  diskSize: number
  isPublic: boolean
  isActive: boolean
  downloads: number
  createdAt: string
  updatedAt?: string
  createdBy?: string
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

export interface VMSnapshot {
  id: string
  name: string
  description?: string
  state?: string
  size: number
  created_at: string
  updated_at?: string
}

export const vmsApi = {
  list: (params?: { page?: number; page_size?: number; status?: string; search?: string }) =>
    client.get('/vms', { params }).then(res => res.data),

  get: (id: string) =>
    client.get(`/vms/${id}`).then(res => res.data),

  create: (data: CreateVMRequest) =>
    client.post('/vms', data).then(res => res.data),

  update: (id: string, data: {
    name?: string;
    boot_order?: string;
    autostart?: boolean;
  }) =>
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
    client.get(`/vms/${id}/stats`).then(res => res.data),

  startInstallation: (id: string) =>
    client.post(`/vms/${id}/start-installation`).then(res => res.data),

  finishInstallation: (id: string) =>
    client.post(`/vms/${id}/finish-installation`).then(res => res.data),

  installAgent: (id: string, data?: { agent_type?: string; script?: string }) =>
    client.post(`/vms/${id}/install-agent`, data || {}).then(res => res.data),

  getInstallationStatus: (id: string) =>
    client.get(`/vms/${id}/installation-status`).then(res => res.data),

  mountISO: (id: string, isoId: string) =>
    client.post(`/vms/${id}/mount-iso`, { isoId }).then(res => res.data),

  unmountISO: (id: string) =>
    client.delete(`/vms/${id}/mount-iso`).then(res => res.data),

  getMountedISO: (id: string) =>
    client.get(`/vms/${id}/mounted-iso`).then(res => res.data),

  clone: (id: string, data: { name: string; description?: string }) =>
    client.post(`/vms/${id}/clone`, data).then(res => res.data)
}

export const snapshotsApi = {
  list: (vmId: string) =>
    client.get(`/vms/${vmId}/snapshots`).then(res => res.data),

  create: (vmId: string, data: { name: string; description?: string }) =>
    client.post(`/vms/${vmId}/snapshots`, data).then(res => res.data),

  restore: (vmId: string, name: string) =>
    client.post(`/vms/${vmId}/snapshots/${name}/restore`, {}).then(res => res.data),

  delete: (vmId: string, name: string) =>
    client.delete(`/vms/${vmId}/snapshots/${name}`).then(res => res.data)
}

export const templatesApi = {
  list: (params?: { page?: number; page_size?: number }) =>
    client.get('/templates', { params }).then(res => res.data),

  get: (id: string) =>
    client.get(`/templates/${id}`).then(res => res.data),

  create: (data: Partial<Template>) =>
    client.post('/templates', data).then(res => res.data),

  update: (id: string, data: {
    name?: string;
    description?: string;
    is_public?: boolean;
    is_active?: boolean;
  }) =>
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
  avatar?: string
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

  getResources: () =>
    client.get('/admin/system/resources').then(res => res.data),

  getAuditLogs: (params?: { page?: number; page_size?: number; user_id?: string; action?: string }) =>
    client.get('/admin/audit-logs', { params }).then(res => res.data)
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

export interface AlertHistory {
  id: string
  alertRuleId: string
  vmId?: string
  severity: string
  metric: string
  currentValue: number
  threshold: number
  condition: string
  message: string
  status: string
  resolvedAt?: string
  notifiedAt?: string
  createdAt: string
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

export const alertHistoryApi = {
  list: (params?: { page?: number; page_size?: number; status?: string }) =>
    client.get('/admin/alert-histories', { params }).then(res => res.data),

  get: (id: string) =>
    client.get(`/admin/alert-histories/${id}`).then(res => res.data),

  resolve: (id: string) =>
    client.post(`/admin/alert-histories/${id}/resolve`).then(res => res.data),

  getActive: () =>
    client.get('/admin/alert-histories/active').then(res => res.data),

  getStats: () =>
    client.get('/admin/alert-histories/stats').then(res => res.data)
}

export interface ISO {
  id: string
  name: string
  description?: string
  fileName: string
  fileSize: number
  isoPath: string
  md5?: string
  sha256?: string
  osType?: string
  osVersion?: string
  architecture: string
  status: string
  uploadedBy?: string
  createdAt: string
  updatedAt?: string
}

export interface ISOUpload {
  id: string
  name: string
  description?: string
  fileName: string
  fileSize: number
  architecture?: string
  osType?: string
  osVersion?: string
  uploadPath?: string
  status: string
  progress: number
  errorMessage?: string
  uploadedBy?: string
  createdAt: string
  completedAt?: string
}

export const isosApi = {
  list: (params?: { page?: number; page_size?: number; search?: string; architecture?: string }) =>
    client.get('/isos', { params }).then(res => res.data),

  get: (id: string) =>
    client.get(`/isos/${id}`).then(res => res.data),

  delete: (id: string) =>
    client.delete(`/isos/${id}`).then(res => res.data),

  initUpload: (data: {
    name: string
    description?: string
    file_name: string
    file_size: number
    architecture?: string
    os_type?: string
    os_version?: string
    chunk_size: number
  }) =>
    client.post('/isos/upload/init', data).then(res => res.data),

  uploadPart: (uploadId: string, chunkIndex: number, totalChunks: number, formData: FormData) =>
    client.post(`/isos/upload/part?upload_id=${uploadId}&chunk_index=${chunkIndex}&total_chunks=${totalChunks}`, formData, {
      headers: { 'Content-Type': 'multipart/form-data' }
    }).then(res => res.data),

  completeUpload: (uploadId: string, data: {
    total_chunks: number
    checksum?: string
    name?: string
    description?: string
    os_type?: string
    os_version?: string
  }) =>
    client.post(`/isos/upload/complete?upload_id=${uploadId}`, data).then(res => res.data),

  getUploadStatus: (uploadId: string) =>
    client.get(`/isos/upload/status?upload_id=${uploadId}`).then(res => res.data)
}
