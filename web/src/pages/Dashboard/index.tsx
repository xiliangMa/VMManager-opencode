import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Row, Col, Card, Statistic, Button, Table, Tag, Progress, List, Typography, Space } from 'antd'
import { useTranslation } from 'react-i18next'
import {
  DesktopOutlined,
  TeamOutlined,
  FileOutlined,
  RocketOutlined,
  CloudServerOutlined,
  WarningOutlined,
  PlusOutlined,
  ArrowRightOutlined,
  ClockCircleOutlined
} from '@ant-design/icons'
import { useAuthStore } from '../../stores/authStore'
import { vmsApi, systemApi, VM } from '../../api/client'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'

dayjs.extend(relativeTime)

interface SystemStats {
  vmCount?: number
  runningVmCount?: number
  stoppedVmCount?: number
  totalUsers?: number
  activeUsers?: number
  totalTemplates?: number
  publicTemplates?: number
}

const Dashboard: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { user } = useAuthStore()
  const [loading, setLoading] = useState(true)
  const [systemStats, setSystemStats] = useState<SystemStats>({})
  const [recentVMs, setRecentVMs] = useState<VM[]>([])
  const [userVMCount, setUserVMCount] = useState(0)
  const [runningVMs, setRunningVMs] = useState(0)

  useEffect(() => {
    fetchDashboardData()
  }, [])

  const fetchDashboardData = async () => {
    setLoading(true)
    try {
      const [statsRes, vmsRes] = await Promise.all([
        systemApi.getStats().catch(() => ({ code: 0, data: {} })),
        vmsApi.list({ page_size: 5 }).catch(() => ({ code: 0, data: [], meta: { total: 0 } }))
      ])

      if (statsRes.code === 0) {
        const data = statsRes.data || {}
        setSystemStats({
          vmCount: data.total_vms,
          runningVmCount: data.running_vms,
          totalUsers: data.total_users,
          totalTemplates: data.total_templates,
        })
      }

      if (vmsRes.code === 0) {
        setRecentVMs((vmsRes.data as VM[]) || [])
        setUserVMCount(vmsRes.meta?.total || 0)
        setRunningVMs(((vmsRes.data as VM[]) || []).filter((vm: VM) => vm.status === 'running').length)
      }
    } catch (error) {
      console.error('Failed to fetch dashboard data:', error)
    } finally {
      setLoading(false)
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running': return 'green'
      case 'stopped': return 'red'
      case 'suspended': return 'orange'
      case 'pending': return 'blue'
      case 'creating': return 'processing'
      default: return 'default'
    }
  }

  const stats = [
    {
      title: t('vm.vmList'),
      value: systemStats.vmCount || userVMCount,
      icon: <DesktopOutlined />,
      color: '#1890ff',
      path: '/vms',
      suffix: userVMCount > 0 ? `Your VMs` : ''
    },
    {
      title: t('vm.running'),
      value: systemStats.runningVmCount || runningVMs,
      icon: <RocketOutlined />,
      color: '#52c41a',
      path: '/vms?status=running',
      suffix: 'Running'
    },
    {
      title: t('admin.totalUsers'),
      value: systemStats.totalUsers || 0,
      icon: <TeamOutlined />,
      color: '#722ed1',
      path: '/admin/users',
      suffix: `${systemStats.activeUsers || 0} active`
    },
    {
      title: 'Templates',
      value: systemStats.totalTemplates || 0,
      icon: <FileOutlined />,
      color: '#fa8c16',
      path: '/templates',
      suffix: `${systemStats.publicTemplates || 0} public`
    }
  ]

  const vmColumns = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: VM) => (
        <Button type="link" onClick={() => navigate(`/vms/${record.id}`)}>
          {name}
        </Button>
      )
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>
          {status?.toUpperCase()}
        </Tag>
      )
    },
    {
      title: 'CPU',
      dataIndex: 'cpuAllocated',
      key: 'cpu',
      render: (cpu: number) => cpu ? `${cpu} vCPU` : '-'
    },
    {
      title: 'Memory',
      dataIndex: 'memoryAllocated',
      key: 'memory',
      render: (memory: number) => memory ? `${memory} MB` : '-'
    },
    {
      title: 'Created',
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (time: string) => (
        <Space>
          <ClockCircleOutlined />
          {time ? dayjs(time).fromNow() : '-'}
        </Space>
      )
    }
  ]

  const quickActions = [
    { label: t('vm.createVM'), icon: <PlusOutlined />, path: '/vms/create', type: 'primary' as const },
    { label: 'View VMs', icon: <ArrowRightOutlined />, path: '/vms', type: 'default' as const },
    { label: 'Templates', icon: <FileOutlined />, path: '/templates', type: 'default' as const },
    { label: 'Alert Rules', icon: <WarningOutlined />, path: '/admin/alerts', type: 'default' as const }
  ]

  return (
    <div>
      <div style={{ marginBottom: 24 }}>
        <Typography.Title level={2}>
          Welcome back, {user?.username}
        </Typography.Title>
        <Typography.Text type="secondary">
          Here's what's happening with your virtual machines
        </Typography.Text>
      </div>

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        {stats.map((stat, index) => (
          <Col xs={24} sm={12} lg={6} key={index}>
            <Card hoverable loading={loading} onClick={() => navigate(stat.path)}>
              <Statistic
                title={stat.title}
                value={stat.value}
                prefix={<span style={{ color: stat.color }}>{stat.icon}</span>}
                suffix={stat.suffix ? <Typography.Text type="secondary" style={{ fontSize: 12 }}>{stat.suffix}</Typography.Text> : undefined}
              />
            </Card>
          </Col>
        ))}
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={16}>
          <Card
            title={
              <Space>
                <CloudServerOutlined />
                Recent Virtual Machines
              </Space>
            }
            extra={<Button type="link" onClick={() => navigate('/vms')}>View All</Button>}
            loading={loading}
          >
            <Table
              columns={vmColumns}
              dataSource={recentVMs}
              rowKey="id"
              pagination={false}
              size="small"
            />
          </Card>
        </Col>

        <Col xs={24} lg={8}>
          <Card title="Quick Actions" loading={loading}>
            <List
              itemLayout="horizontal"
              dataSource={quickActions}
              renderItem={(item) => (
                <List.Item>
                  <Button
                    type={item.type}
                    icon={item.icon}
                    block
                    onClick={() => navigate(item.path)}
                  >
                    {item.label}
                  </Button>
                </List.Item>
              )}
            />
          </Card>

          <Card
            title="System Resources"
            style={{ marginTop: 16 }}
            loading={loading}
            size="small"
          >
            <Space direction="vertical" style={{ width: '100%' }} size="middle">
              <div>
                <Typography.Text>CPU Usage</Typography.Text>
                <Progress percent={30} status="active" size="small" />
              </div>
              <div>
                <Typography.Text>Memory Usage</Typography.Text>
                <Progress percent={45} status="active" size="small" strokeColor="#52c41a" />
              </div>
              <div>
                <Typography.Text>Disk Usage</Typography.Text>
                <Progress percent={60} status="active" size="small" strokeColor="#fa8c16" />
              </div>
            </Space>
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default Dashboard
