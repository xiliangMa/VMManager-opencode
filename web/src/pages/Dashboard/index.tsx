import React, { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { Row, Col, Card, Statistic, Progress, Typography, Spin, Empty } from 'antd'
import { useTranslation } from 'react-i18next'
import {
  DesktopOutlined,
  TeamOutlined,
  FileOutlined,
  RocketOutlined,
  ThunderboltOutlined,
  CloudOutlined,
  HddOutlined
} from '@ant-design/icons'
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer
} from 'recharts'
import { useAuthStore } from '../../stores/authStore'
import { systemApi } from '../../api/client'

const { Title, Text } = Typography

interface SystemStats {
  vmCount: number
  runningVmCount: number
  stoppedVmCount: number
  totalUsers: number
  activeUsers: number
  totalTemplates: number
  publicTemplates: number
}

interface SystemResources {
  cpu_percent: number
  memory_percent: number
  disk_percent: number
  total_memory_mb: number
  used_memory_mb: number
  total_disk_gb: number
  used_disk_gb: number
  total_cpu_cores: number
}

interface DataPoint {
  time: string
  cpu: number
  memory: number
}

const Dashboard: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { user } = useAuthStore()
  const [loading, setLoading] = useState(true)
  const [systemStats, setSystemStats] = useState<SystemStats>({
    vmCount: 0,
    runningVmCount: 0,
    stoppedVmCount: 0,
    totalUsers: 0,
    activeUsers: 0,
    totalTemplates: 0,
    publicTemplates: 0
  })
  const [systemResources, setSystemResources] = useState<SystemResources>({
    cpu_percent: 0,
    memory_percent: 0,
    disk_percent: 0,
    total_memory_mb: 0,
    used_memory_mb: 0,
    total_disk_gb: 0,
    used_disk_gb: 0,
    total_cpu_cores: 8
  })
  const [historyData, setHistoryData] = useState<DataPoint[]>([])
  const intervalRef = useRef<any>(null)

  useEffect(() => {
    fetchDashboardData()
    intervalRef.current = setInterval(fetchSystemResources, 5000)
    
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
      }
    }
  }, [])

  const fetchDashboardData = async () => {
    setLoading(true)
    try {
      await Promise.all([
        fetchSystemStats(),
        fetchSystemResources()
      ])
    } catch (error) {
      console.error('Failed to fetch dashboard data:', error)
    } finally {
      setLoading(false)
    }
  }

  const fetchSystemStats = async () => {
    try {
      const response = await systemApi.getStats().catch(() => ({ code: 0, data: {} }))
      if (response.code === 0) {
        const data = response.data || {}
        setSystemStats({
          vmCount: data.total_vms !== undefined ? data.total_vms : 0,
          runningVmCount: data.running_vms !== undefined ? data.running_vms : 0,
          stoppedVmCount: data.stopped_vms !== undefined ? data.stopped_vms : 0,
          totalUsers: data.total_users !== undefined ? data.total_users : 0,
          activeUsers: data.active_users !== undefined ? data.active_users : 0,
          totalTemplates: data.total_templates !== undefined ? data.total_templates : 0,
          publicTemplates: data.public_templates !== undefined ? data.public_templates : 0
        })
      }
    } catch (error) {
      console.error('Failed to fetch system stats:', error)
    }
  }

  const fetchSystemResources = async () => {
    try {
      const response = await systemApi.getResources().catch(() => ({ code: 0, data: {} }))
      if (response.code === 0) {
        const data = response.data || {}
        const cpuPercent = data.cpu_percent !== undefined ? data.cpu_percent : 0
        const memoryPercent = data.memory_percent !== undefined ? data.memory_percent : 0
        
        setSystemResources({
          cpu_percent: cpuPercent,
          memory_percent: memoryPercent,
          disk_percent: data.disk_percent !== undefined ? data.disk_percent : 0,
          total_memory_mb: data.total_memory_mb !== undefined ? data.total_memory_mb : 0,
          used_memory_mb: data.used_memory_mb !== undefined ? data.used_memory_mb : 0,
          total_disk_gb: data.total_disk_gb !== undefined ? data.total_disk_gb : 0,
          used_disk_gb: data.used_disk_gb !== undefined ? data.used_disk_gb : 0,
          total_cpu_cores: 8
        })

        setHistoryData(prev => {
          const newData = [...prev, {
            time: new Date().toLocaleTimeString(),
            cpu: cpuPercent,
            memory: memoryPercent
          }]
          return newData.slice(-20)
        })
      }
    } catch (error) {
      console.error('Failed to fetch system resources:', error)
    }
  }

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: '100px 0' }}>
        <Spin size="large" />
        <div style={{ marginTop: 16 }}>
          <Text type="secondary">{t('common.loading')}</Text>
        </div>
      </div>
    )
  }

  return (
    <div style={{ padding: '0 24px' }}>
      <div style={{ marginBottom: 24 }}>
        <Title level={2} style={{ marginBottom: 8 }}>
          {t('dashboard.welcome')}, {user?.username} 👋
        </Title>
        <Text type="secondary" style={{ fontSize: 16 }}>
          {t('dashboard.systemOverview')}
        </Text>
      </div>

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card 
            style={{ 
              background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
              border: 'none',
              borderRadius: '12px',
              boxShadow: '0 4px 12px rgba(102, 126, 234, 0.4)',
              height: '100%',
              cursor: 'pointer',
              transition: 'all 0.3s ease'
            }}
            hoverable
            onClick={() => navigate('/vms')}
          >
            <Statistic
              title={<span style={{ color: 'rgba(255, 255, 255, 0.9)' }}>{t('vm.vmList')}</span>}
              value={systemStats.vmCount}
              prefix={<DesktopOutlined style={{ color: '#fff' }} />}
              valueStyle={{ color: '#fff', fontSize: '32px', fontWeight: 'bold' }}
              suffix={<span style={{ color: 'rgba(255, 255, 255, 0.7)', fontSize: 14 }}>{t('vm.vmList')}</span>}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card 
            style={{ 
              background: 'linear-gradient(135deg, #11998e 0%, #38ef7d 100%)',
              border: 'none',
              borderRadius: '12px',
              boxShadow: '0 4px 12px rgba(17, 153, 142, 0.4)',
              height: '100%',
              cursor: 'pointer',
              transition: 'all 0.3s ease'
            }}
            hoverable
            onClick={() => navigate('/vms')}
          >
            <Statistic
              title={<span style={{ color: 'rgba(255, 255, 255, 0.9)' }}>{t('vm.running')}</span>}
              value={systemStats.runningVmCount}
              prefix={<RocketOutlined style={{ color: '#fff' }} />}
              valueStyle={{ color: '#fff', fontSize: '32px', fontWeight: 'bold' }}
              suffix={<span style={{ color: 'rgba(255, 255, 255, 0.7)', fontSize: 14 }}>/ {systemStats.vmCount}</span>}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card 
            style={{ 
              background: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
              border: 'none',
              borderRadius: '12px',
              boxShadow: '0 4px 12px rgba(240, 147, 251, 0.4)',
              height: '100%',
              cursor: 'pointer',
              transition: 'all 0.3s ease'
            }}
            hoverable
            onClick={() => navigate('/admin/users')}
          >
            <Statistic
              title={<span style={{ color: 'rgba(255, 255, 255, 0.9)' }}>{t('admin.totalUsers')}</span>}
              value={systemStats.totalUsers}
              prefix={<TeamOutlined style={{ color: '#fff' }} />}
              valueStyle={{ color: '#fff', fontSize: '32px', fontWeight: 'bold' }}
              suffix={<span style={{ color: 'rgba(255, 255, 255, 0.7)', fontSize: 14 }}>{systemStats.activeUsers} {t('common.active')}</span>}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card 
            style={{ 
              background: 'linear-gradient(135deg, #fa709a 0%, #fee140 100%)',
              border: 'none',
              borderRadius: '12px',
              boxShadow: '0 4px 12px rgba(250, 112, 154, 0.4)',
              height: '100%',
              cursor: 'pointer',
              transition: 'all 0.3s ease'
            }}
            hoverable
            onClick={() => navigate('/templates')}
          >
            <Statistic
              title={<span style={{ color: 'rgba(255, 255, 255, 0.9)' }}>{t('template.templateList')}</span>}
              value={systemStats.totalTemplates}
              prefix={<FileOutlined style={{ color: '#fff' }} />}
              valueStyle={{ color: '#fff', fontSize: '32px', fontWeight: 'bold' }}
              suffix={<span style={{ color: 'rgba(255, 255, 255, 0.7)', fontSize: 14 }}>{systemStats.publicTemplates} {t('template.public')}</span>}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} lg={8}>
          <Card 
            title={
              <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <ThunderboltOutlined style={{ color: '#1890ff' }} />
                <span>{t('system.cpuResources')}</span>
              </span>
            }
            style={{ borderRadius: '12px', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.08)', height: '100%', border: '1px solid #91d5ff' }}
          >
            <div style={{ marginBottom: 16, textAlign: 'center' }}>
              <Progress
                type="circle"
                percent={systemResources.cpu_percent}
                strokeColor={{
                  '0%': '#1890ff',
                  '100%': '#40a9ff'
                }}
                trailColor="#d6e4ff"
                strokeWidth={12}
                width={120}
                format={(percent) => (
                  <span style={{ fontSize: 24, fontWeight: 'bold', color: '#1890ff' }}>
                    {percent?.toFixed(1)}%
                  </span>
                )}
              />
            </div>
            <div style={{ textAlign: 'center', marginBottom: 8 }}>
              <Text type="secondary" style={{ fontSize: 14 }}>
                {systemResources.total_cpu_cores} {t('system.cores')}
              </Text>
            </div>
            <div style={{ marginTop: 12, display: 'flex', justifyContent: 'space-between' }}>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {t('system.used')}: {systemResources.cpu_percent.toFixed(1)}%
              </Text>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {t('system.available')}: {(100 - systemResources.cpu_percent).toFixed(1)}%
              </Text>
            </div>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card 
            title={
              <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <CloudOutlined style={{ color: '#722ed1' }} />
                <span>{t('system.memoryResources')}</span>
              </span>
            }
            style={{ borderRadius: '12px', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.08)', height: '100%', border: '1px solid #b37feb' }}
          >
            <div style={{ marginBottom: 16, textAlign: 'center' }}>
              <Progress
                type="circle"
                percent={systemResources.memory_percent}
                strokeColor={{
                  '0%': '#722ed1',
                  '100%': '#b37feb'
                }}
                trailColor="#d3adf7"
                strokeWidth={12}
                width={120}
                format={(percent) => (
                  <span style={{ fontSize: 24, fontWeight: 'bold', color: '#722ed1' }}>
                    {percent?.toFixed(1)}%
                  </span>
                )}
              />
            </div>
            <div style={{ textAlign: 'center', marginBottom: 8 }}>
              <Text type="secondary" style={{ fontSize: 14 }}>
                {(systemResources.used_memory_mb / 1024).toFixed(1)} GB / {(systemResources.total_memory_mb / 1024).toFixed(1)} GB
              </Text>
            </div>
            <div style={{ marginTop: 12, display: 'flex', justifyContent: 'space-between' }}>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {t('system.used')}: {(systemResources.used_memory_mb / 1024).toFixed(1)} GB
              </Text>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {t('system.available')}: {((systemResources.total_memory_mb - systemResources.used_memory_mb) / 1024).toFixed(1)} GB
              </Text>
            </div>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card 
            title={
              <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <HddOutlined style={{ color: '#fa8c16' }} />
                <span>{t('system.diskResources')}</span>
              </span>
            }
            style={{ borderRadius: '12px', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.08)', height: '100%', border: '1px solid #ffc53d' }}
          >
            <div style={{ marginBottom: 16, textAlign: 'center' }}>
              <Progress
                type="circle"
                percent={systemResources.disk_percent}
                strokeColor={{
                  '0%': '#fa8c16',
                  '100%': '#ffc53d'
                }}
                trailColor="#ffd591"
                strokeWidth={12}
                width={120}
                format={(percent) => (
                  <span style={{ fontSize: 24, fontWeight: 'bold', color: '#fa8c16' }}>
                    {percent?.toFixed(1)}%
                  </span>
                )}
              />
            </div>
            <div style={{ textAlign: 'center', marginBottom: 8 }}>
              <Text type="secondary" style={{ fontSize: 14 }}>
                {systemResources.used_disk_gb} GB / {systemResources.total_disk_gb} GB
              </Text>
            </div>
            <div style={{ marginTop: 12, display: 'flex', justifyContent: 'space-between' }}>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {t('system.used')}: {systemResources.used_disk_gb} GB
              </Text>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {t('system.available')}: {systemResources.total_disk_gb - systemResources.used_disk_gb} GB
              </Text>
            </div>
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} lg={12}>
          <Card 
            title={
              <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <ThunderboltOutlined style={{ color: '#1890ff' }} />
                <span>{t('system.cpuUsageTrend')}</span>
              </span>
            }
            style={{ borderRadius: '12px', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.08)', border: '1px solid #91d5ff' }}
          >
            {historyData.length > 0 ? (
              <ResponsiveContainer width="100%" height={250}>
                <AreaChart data={historyData}>
                  <defs>
                    <linearGradient id="colorCpu" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#1890ff" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#1890ff" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                  <XAxis dataKey="time" tick={{ fontSize: 11 }} />
                  <YAxis domain={[0, 100]} tickFormatter={(v) => `${v}%`} />
                  <Tooltip
                    formatter={(value: number) => [`${value.toFixed(1)}%`, 'CPU']}
                    labelFormatter={(label) => `Time: ${label}`}
                    contentStyle={{ borderRadius: '8px', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.15)' }}
                  />
                  <Area
                    type="monotone"
                    dataKey="cpu"
                    name="CPU"
                    stroke="#1890ff"
                    fill="url(#colorCpu)"
                    strokeWidth={2}
                  />
                </AreaChart>
              </ResponsiveContainer>
            ) : (
              <Empty description={t('common.noData')} style={{ padding: '60px 0' }} />
            )}
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card 
            title={
              <span style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <CloudOutlined style={{ color: '#722ed1' }} />
                <span>{t('system.memoryUsageTrend')}</span>
              </span>
            }
            style={{ borderRadius: '12px', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.08)', border: '1px solid #b37feb' }}
          >
            {historyData.length > 0 ? (
              <ResponsiveContainer width="100%" height={250}>
                <AreaChart data={historyData}>
                  <defs>
                    <linearGradient id="colorMemory" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#722ed1" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#722ed1" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                  <XAxis dataKey="time" tick={{ fontSize: 11 }} />
                  <YAxis domain={[0, 100]} tickFormatter={(v) => `${v}%`} />
                  <Tooltip
                    formatter={(value: number) => [`${value.toFixed(1)}%`, t('vm.monitor.memory')]}
                    labelFormatter={(label) => `Time: ${label}`}
                    contentStyle={{ borderRadius: '8px', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.15)' }}
                  />
                  <Area
                    type="monotone"
                    dataKey="memory"
                    name={t('vm.monitor.memory')}
                    stroke="#722ed1"
                    fill="url(#colorMemory)"
                    strokeWidth={2}
                  />
                </AreaChart>
              </ResponsiveContainer>
            ) : (
              <Empty description={t('common.noData')} style={{ padding: '60px 0' }} />
            )}
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default Dashboard
