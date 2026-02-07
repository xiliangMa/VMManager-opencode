import React, { useEffect, useState } from 'react'
import { Row, Col, Card, Statistic, Progress, Spin, Typography } from 'antd'
import {
  DesktopOutlined,
  TeamOutlined,
  RiseOutlined,
  ThunderboltOutlined
} from '@ant-design/icons'
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell
} from 'recharts'
import { SystemResourceStats, statsApi } from '../../api/client'

const { Title } = Typography

const SystemDashboard: React.FC = () => {
  const [stats, setStats] = useState<SystemResourceStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [history, setHistory] = useState<any>({ cpu: [], memory: [] })

  const fetchStats = async () => {
    try {
      const response = await statsApi.getSystemStats()
      const data = response.data || response
      setStats(data)
    } catch (error) {
      console.error('Failed to fetch system stats:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    const cpuHistory = Array.from({ length: 20 }, (_, i) => ({
      time: new Date(Date.now() - (19 - i) * 5000).toLocaleTimeString(),
      value: 30 + Math.random() * 30
    }))
    const memoryHistory = Array.from({ length: 20 }, (_, i) => ({
      time: new Date(Date.now() - (19 - i) * 5000).toLocaleTimeString(),
      value: 35 + Math.random() * 20
    }))
    setHistory({ cpu: cpuHistory, memory: memoryHistory })
    fetchStats()
  }, [])

  const pieData = stats ? [
    { name: 'Used', value: stats.usedCpu },
    { name: 'Available', value: stats.totalCpu - stats.usedCpu }
  ] : []

  const COLORS = ['#1890ff', '#f0f0f0']

  if (loading) {
    return (
      <div style={{ textAlign: 'center', padding: 100 }}>
        <Spin size="large" />
        <p style={{ marginTop: 16 }}>Loading system statistics...</p>
      </div>
    )
  }

  if (!stats) {
    return <div>Failed to load system statistics</div>
  }

  const statCards = [
    {
      title: 'Total VMs',
      value: stats.vmCount,
      suffix: '',
      icon: <DesktopOutlined />,
      color: '#1890ff',
      path: '/vms'
    },
    {
      title: 'Running VMs',
      value: stats.runningVmCount,
      suffix: `/${stats.vmCount}`,
      icon: <RiseOutlined />,
      color: '#52c41a',
      path: '/vms'
    },
    {
      title: 'Active Users',
      value: stats.activeUsers,
      suffix: '',
      icon: <TeamOutlined />,
      color: '#722ed1',
      path: '/admin/users'
    },
    {
      title: 'CPU Usage',
      value: stats.cpuPercent,
      suffix: '%',
      icon: <ThunderboltOutlined />,
      color: '#fa8c16',
      path: '/vms'
    }
  ]

  return (
    <div>
      <Title level={3}>System Dashboard</Title>

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        {statCards.map((item, index) => (
          <Col xs={24} sm={12} lg={6} key={index}>
            <Card hoverable style={{ cursor: 'default' }}>
              <Statistic
                title={item.title}
                value={item.value}
                suffix={item.suffix}
                prefix={React.cloneElement(item.icon, { style: { color: item.color } })}
                valueStyle={{ color: item.color }}
              />
            </Card>
          </Col>
        ))}
      </Row>

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} lg={8}>
          <Card title="CPU Resources" size="small">
            <Row align="middle" gutter={16}>
              <Col span={12}>
                <ResponsiveContainer width="100%" height={150}>
                  <PieChart>
                    <Pie
                      data={pieData}
                      cx="50%"
                      cy="50%"
                      innerRadius={40}
                      outerRadius={60}
                      paddingAngle={5}
                      dataKey="value"
                    >
                      {pieData.map((_, index) => (
                        <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                      ))}
                    </Pie>
                    <Tooltip formatter={(value: number) => `${value.toFixed(1)}%`} />
                  </PieChart>
                </ResponsiveContainer>
              </Col>
              <Col span={12}>
                <div style={{ textAlign: 'center' }}>
                  <Typography.Text strong style={{ fontSize: 24 }}>{stats.cpuPercent.toFixed(1)}%</Typography.Text>
                  <br />
                  <Typography.Text type="secondary">
                    {stats.usedCpu} / {stats.totalCpu} Cores
                  </Typography.Text>
                </div>
              </Col>
            </Row>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="Memory Resources" size="small">
            <Progress
              percent={stats.memoryPercent}
              strokeColor={stats.memoryPercent > 80 ? '#ff4d4f' : '#722ed1'}
              format={(percent) => `${(percent as number).toFixed(1)}%`}
            />
            <div style={{ marginTop: 16 }}>
              <Typography.Text type="secondary">
                Used: {(stats.usedMemory / 1024).toFixed(1)} GB / Total: {(stats.totalMemory / 1024).toFixed(1)} GB
              </Typography.Text>
            </div>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card title="Disk Resources" size="small">
            <Progress
              percent={stats.diskPercent}
              strokeColor={stats.diskPercent > 80 ? '#ff4d4f' : '#fa8c16'}
              format={(percent) => `${(percent as number).toFixed(1)}%`}
            />
            <div style={{ marginTop: 16 }}>
              <Typography.Text type="secondary">
                Used: {stats.usedDisk} GB / Total: {stats.totalDisk} GB
              </Typography.Text>
            </div>
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title="CPU Usage Trend" size="small">
            <ResponsiveContainer width="100%" height={250}>
              <AreaChart data={history.cpu}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="time" tick={{ fontSize: 12 }} />
                <YAxis domain={[0, 100]} tickFormatter={(v) => `${v}%`} />
                <Tooltip
                  formatter={(value: number) => [`${value.toFixed(1)}%`, 'CPU']}
                  labelFormatter={(label) => `Time: ${label}`}
                />
                <Area
                  type="monotone"
                  dataKey="value"
                  stroke="#1890ff"
                  fill="url(#colorCpuTrend)"
                  strokeWidth={2}
                />
                <defs>
                  <linearGradient id="colorCpuTrend" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#1890ff" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#1890ff" stopOpacity={0} />
                  </linearGradient>
                </defs>
              </AreaChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="Memory Usage Trend" size="small">
            <ResponsiveContainer width="100%" height={250}>
              <AreaChart data={history.memory}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="time" tick={{ fontSize: 12 }} />
                <YAxis domain={[0, 100]} tickFormatter={(v) => `${v}%`} />
                <Tooltip
                  formatter={(value: number) => [`${value.toFixed(1)}%`, 'Memory']}
                  labelFormatter={(label) => `Time: ${label}`}
                />
                <Area
                  type="monotone"
                  dataKey="value"
                  stroke="#722ed1"
                  fill="url(#colorMemTrend)"
                  strokeWidth={2}
                />
                <defs>
                  <linearGradient id="colorMemTrend" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#722ed1" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#722ed1" stopOpacity={0} />
                  </linearGradient>
                </defs>
              </AreaChart>
            </ResponsiveContainer>
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default SystemDashboard
