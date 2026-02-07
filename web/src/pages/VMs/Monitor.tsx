import React, { useEffect, useState, useRef } from 'react'
import { useParams } from 'react-router-dom'
import { Card, Row, Col, Statistic, Progress, Spin } from 'antd'
import {
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  AreaChart,
  Area
} from 'recharts'
import { ThunderboltOutlined, CloudOutlined, HddOutlined, RiseOutlined } from '@ant-design/icons'
import { VMResourceStats, statsApi } from '../../api/client'

const Monitor: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const [stats, setStats] = useState<VMResourceStats | null>(null)
  const [loading, setLoading] = useState(true)
  const intervalRef = useRef<any>(null)

  const fetchStats = async () => {
    if (!id) return
    try {
      const response = await statsApi.getVMStats(id)
      const data = response.data || response
      setStats(data)
    } catch (error) {
      console.error('Failed to fetch VM stats:', error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchStats()
    intervalRef.current = setInterval(fetchStats, 5000)
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
      }
    }
  }, [id])

  if (loading && !stats) {
    return (
      <div style={{ textAlign: 'center', padding: 50 }}>
        <Spin size="large" />
        <p style={{ marginTop: 16 }}>Loading VM statistics...</p>
      </div>
    )
  }

  if (!stats) {
    return null
  }

  return (
    <div>
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="CPU Usage"
              value={stats.cpuUsage}
              precision={1}
              suffix="%"
              prefix={<ThunderboltOutlined style={{ color: '#1890ff' }} />}
              valueStyle={{ color: stats.cpuUsage > 80 ? '#ff4d4f' : '#52c41a' }}
            />
            <Progress
              percent={Math.min(stats.cpuUsage, 100)}
              showInfo={false}
              strokeColor={stats.cpuUsage > 80 ? '#ff4d4f' : '#52c41a'}
              size="small"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Memory Usage"
              value={stats.memoryUsage}
              precision={1}
              suffix="%"
              prefix={<CloudOutlined style={{ color: '#722ed1' }} />}
              valueStyle={{ color: stats.memoryUsage > 80 ? '#ff4d4f' : '#52c41a' }}
            />
            <Progress
              percent={Math.min(stats.memoryUsage, 100)}
              showInfo={false}
              strokeColor={stats.memoryUsage > 80 ? '#ff4d4f' : '#722ed1'}
              size="small"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Disk Usage"
              value={stats.diskUsage}
              precision={1}
              suffix="%"
              prefix={<HddOutlined style={{ color: '#fa8c16' }} />}
              valueStyle={{ color: stats.diskUsage > 80 ? '#ff4d4f' : '#52c41a' }}
            />
            <Progress
              percent={Math.min(stats.diskUsage, 100)}
              showInfo={false}
              strokeColor={stats.diskUsage > 80 ? '#ff4d4f' : '#fa8c16'}
              size="small"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card>
            <Statistic
              title="Network I/O"
              value={stats.networkIn + stats.networkOut}
              precision={1}
              suffix="MB/s"
              prefix={<RiseOutlined style={{ color: '#13c2c2' }} />}
            />
            <div style={{ fontSize: 12, color: '#888', marginTop: 8 }}>
              <span>In: {(stats.networkIn).toFixed(1)} MB/s</span>
              <span style={{ marginLeft: 16 }}>Out: {(stats.networkOut).toFixed(1)} MB/s</span>
            </div>
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card title="CPU History" size="small">
            <ResponsiveContainer width="100%" height={200}>
              <AreaChart data={stats.cpuHistory}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="timestamp" tick={{ fontSize: 12 }} />
                <YAxis domain={[0, 100]} tickFormatter={(v) => `${v}%`} />
                <Tooltip
                  formatter={(value: number) => [`${value.toFixed(2)}%`, 'CPU']}
                  labelFormatter={(label) => `Time: ${label}`}
                />
                <Area
                  type="monotone"
                  dataKey="value"
                  stroke="#1890ff"
                  fill="url(#colorCpu)"
                  strokeWidth={2}
                />
                <defs>
                  <linearGradient id="colorCpu" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#1890ff" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#1890ff" stopOpacity={0} />
                  </linearGradient>
                </defs>
              </AreaChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card title="Memory History" size="small">
            <ResponsiveContainer width="100%" height={200}>
              <AreaChart data={stats.memoryHistory}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="timestamp" tick={{ fontSize: 12 }} />
                <YAxis domain={[0, 100]} tickFormatter={(v) => `${v}%`} />
                <Tooltip
                  formatter={(value: number) => [`${value.toFixed(2)}%`, 'Memory']}
                  labelFormatter={(label) => `Time: ${label}`}
                />
                <Area
                  type="monotone"
                  dataKey="value"
                  stroke="#722ed1"
                  fill="url(#colorMem)"
                  strokeWidth={2}
                />
                <defs>
                  <linearGradient id="colorMem" x1="0" y1="0" x2="0" y2="1">
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

export default Monitor
