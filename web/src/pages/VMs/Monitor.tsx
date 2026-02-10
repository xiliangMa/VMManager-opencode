import React, { useEffect, useState, useRef } from 'react'
import { useParams } from 'react-router-dom'
import { Card, Row, Col, Statistic, Progress, Spin, Select } from 'antd'
import {
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  AreaChart,
  Area,
  Legend
} from 'recharts'
import { ThunderboltOutlined, CloudOutlined, HddOutlined, RiseOutlined } from '@ant-design/icons'
import { VMResourceStats, statsApi } from '../../api/client'
import { useTranslation } from 'react-i18next'
import i18n from 'i18next'

const { Option } = Select

const Monitor: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const { t } = useTranslation()
  const [stats, setStats] = useState<VMResourceStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [timeRange, setTimeRange] = useState('1h')
  const [chartRange, setChartRange] = useState('24h')
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

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp)
    return date.toLocaleTimeString(i18n.language, { hour: '2-digit', minute: '2-digit' })
  }

  const formatTimestampLong = (timestamp: string) => {
    const date = new Date(timestamp)
    return date.toLocaleString(i18n.language)
  }

  if (loading && !stats) {
    return (
      <div style={{ textAlign: 'center', padding: 50 }}>
        <Spin size="large" />
        <p style={{ marginTop: 16 }}>{t('vm.monitor.loading')}</p>
      </div>
    )
  }

  if (!stats) {
    return null
  }

  const combinedChartData = stats.cpuHistory.map((cpuPoint, i) => ({
    timestamp: cpuPoint.timestamp,
    cpu: cpuPoint.value,
    memory: stats.memoryHistory[i]?.value || 0,
    disk: stats.diskHistory[i]?.value || 0
  }))

  return (
    <div>
      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col flex="auto">
          <Select value={timeRange} onChange={setTimeRange} style={{ width: 120 }}>
            <Option value="15m">{t('vm.monitor.range.15min')}</Option>
            <Option value="1h">{t('vm.monitor.range.1hour')}</Option>
            <Option value="6h">{t('vm.monitor.range.6hours')}</Option>
            <Option value="24h">{t('vm.monitor.range.24hours')}</Option>
          </Select>
          <Select value={chartRange} onChange={setChartRange} style={{ width: 140, marginLeft: 16 }}>
            <Option value="6h">{t('vm.monitor.chart.last6h')}</Option>
            <Option value="24h">{t('vm.monitor.chart.last24h')}</Option>
            <Option value="7d">{t('vm.monitor.chart.last7d')}</Option>
          </Select>
        </Col>
        <Col>
          <span style={{ marginRight: 16, color: '#888' }}>
            {t('vm.monitor.lastUpdate')}: {new Date().toLocaleTimeString(i18n.language)}
          </span>
          <span style={{ marginRight: 8, color: '#888' }}>
            {stats.cpuHistory?.length || 0} {t('vm.monitor.dataPoints')}
          </span>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card size="small" title={<><ThunderboltOutlined style={{ color: '#1890ff', marginRight: 8 }} />{t('vm.monitor.cpuUsage')}</>}>
            <Statistic
              value={stats.cpuUsage}
              precision={1}
              suffix="%"
              valueStyle={{ color: stats.cpuUsage > 80 ? '#ff4d4f' : stats.cpuUsage > 60 ? '#faad14' : '#52c41a' }}
            />
            <Progress
              percent={Math.min(stats.cpuUsage, 100)}
              showInfo={false}
              strokeColor={stats.cpuUsage > 80 ? '#ff4d4f' : stats.cpuUsage > 60 ? '#faad14' : '#52c41a'}
              size="small"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card size="small" title={<><CloudOutlined style={{ color: '#722ed1', marginRight: 8 }} />{t('vm.monitor.memoryUsage')}</>}>
            <Statistic
              value={stats.memoryUsage}
              precision={1}
              suffix="%"
              valueStyle={{ color: stats.memoryUsage > 80 ? '#ff4d4f' : stats.memoryUsage > 60 ? '#faad14' : '#52c41a' }}
            />
            <Progress
              percent={Math.min(stats.memoryUsage, 100)}
              showInfo={false}
              strokeColor={stats.memoryUsage > 80 ? '#ff4d4f' : stats.memoryUsage > 60 ? '#faad14' : '#722ed1'}
              size="small"
            />
            <div style={{ fontSize: 12, color: '#888', marginTop: 4 }}>
              {t('vm.monitor.used')}: {(stats.memoryUsage / 100 * 8192).toFixed(0)} MB / 8192 MB
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card size="small" title={<><HddOutlined style={{ color: '#fa8c16', marginRight: 8 }} />{t('vm.monitor.diskUsage')}</>}>
            <Statistic
              value={stats.diskUsage}
              precision={1}
              suffix="%"
              valueStyle={{ color: stats.diskUsage > 80 ? '#ff4d4f' : stats.diskUsage > 60 ? '#faad14' : '#52c41a' }}
            />
            <Progress
              percent={Math.min(stats.diskUsage, 100)}
              showInfo={false}
              strokeColor={stats.diskUsage > 80 ? '#ff4d4f' : stats.diskUsage > 60 ? '#faad14' : '#fa8c16'}
              size="small"
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card size="small" title={<><RiseOutlined style={{ color: '#13c2c2', marginRight: 8 }} />{t('vm.monitor.networkIO')}</>}>
            <Statistic
              value={(stats.networkIn + stats.networkOut) * 1024}
              precision={1}
              suffix="KB/s"
              valueStyle={{ color: '#13c2c2' }}
            />
            <div style={{ fontSize: 12, color: '#888', marginTop: 4 }}>
              <span style={{ color: '#52c41a' }}>↓ {(stats.networkIn * 1024).toFixed(0)} KB/s</span>
              <span style={{ marginLeft: 16, color: '#1890ff' }}>↑ {(stats.networkOut * 1024).toFixed(0)} KB/s</span>
            </div>
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col xs={24} lg={16}>
          <Card
            size="small"
            title={t('vm.monitor.cpuMemoryChart')}
            extra={
              <Select defaultValue="area" style={{ width: 100 }} size="small">
                <Option value="area">{t('vm.monitor.chartType.area')}</Option>
                <Option value="line">{t('vm.monitor.chartType.line')}</Option>
              </Select>
            }
          >
            <ResponsiveContainer width="100%" height={280}>
              {chartRange === '24h' && combinedChartData.length > 0 ? (
                <AreaChart data={combinedChartData.slice(-100)}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                  <XAxis
                    dataKey="timestamp"
                    tick={{ fontSize: 11 }}
                    tickFormatter={formatTimestamp}
                  />
                  <YAxis domain={[0, 100]} tickFormatter={(v) => `${v}%`} />
                  <Tooltip
                    formatter={(value: number, name: string) => [`${value.toFixed(1)}%`, name === 'cpu' ? 'CPU' : name === 'memory' ? t('vm.monitor.memory') : t('vm.monitor.disk')]}
                    labelFormatter={formatTimestampLong}
                  />
                  <Legend />
                  <Area
                    type="monotone"
                    dataKey="cpu"
                    name="CPU"
                    stroke="#1890ff"
                    fill="url(#colorCpu)"
                    strokeWidth={2}
                  />
                  <Area
                    type="monotone"
                    dataKey="memory"
                    name={t('vm.monitor.memory')}
                    stroke="#722ed1"
                    fill="url(#colorMem)"
                    strokeWidth={2}
                  />
                  <defs>
                    <linearGradient id="colorCpu" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#1890ff" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#1890ff" stopOpacity={0} />
                    </linearGradient>
                    <linearGradient id="colorMem" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#722ed1" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#722ed1" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                </AreaChart>
              ) : (
                <AreaChart data={combinedChartData.slice(-50)}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                  <XAxis dataKey="timestamp" tick={{ fontSize: 11 }} tickFormatter={formatTimestamp} />
                  <YAxis domain={[0, 100]} tickFormatter={(v) => `${v}%`} />
                  <Tooltip formatter={(value: number) => [`${value.toFixed(1)}%`, 'Value']} />
                  <Area type="monotone" dataKey="cpu" stroke="#1890ff" fill="url(#colorCpu)" strokeWidth={2} />
                  <defs>
                    <linearGradient id="colorCpu" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor="#1890ff" stopOpacity={0.3} />
                      <stop offset="95%" stopColor="#1890ff" stopOpacity={0} />
                    </linearGradient>
                  </defs>
                </AreaChart>
              )}
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card size="small" title={t('vm.monitor.resourceAllocation')}>
            <div style={{ marginBottom: 16 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                <span>{t('vm.monitor.cpuCores')}</span>
                <span style={{ color: '#1890ff' }}>4 {t('vm.monitor.cores')}</span>
              </div>
              <Progress percent={75} strokeColor="#1890ff" size="small" />
            </div>
            <div style={{ marginBottom: 16 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                <span>{t('vm.monitor.memoryAllocated')}</span>
                <span style={{ color: '#722ed1' }}>8192 MB</span>
              </div>
              <Progress percent={62.5} strokeColor="#722ed1" size="small" />
            </div>
            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                <span>{t('vm.monitor.diskAllocated')}</span>
                <span style={{ color: '#fa8c16' }}>100 GB</span>
              </div>
              <Progress percent={45} strokeColor="#fa8c16" size="small" />
            </div>
          </Card>
          <Card size="small" title={t('vm.monitor.quickStats')} style={{ marginTop: 16 }}>
            <Row gutter={16}>
              <Col span={12}>
                <Statistic
                  title={t('vm.monitor.avgCPU')}
                  value={stats.cpuHistory.reduce((sum, p) => sum + (p.value || 0), 0) / Math.max(stats.cpuHistory.length, 1)}
                  precision={1}
                  suffix="%"
                  valueStyle={{ fontSize: 18, color: '#1890ff' }}
                />
              </Col>
              <Col span={12}>
                <Statistic
                  title={t('vm.monitor.avgMem')}
                  value={stats.memoryHistory.reduce((sum, p) => sum + (p.value || 0), 0) / Math.max(stats.memoryHistory.length, 1)}
                  precision={1}
                  suffix="%"
                  valueStyle={{ fontSize: 18, color: '#722ed1' }}
                />
              </Col>
            </Row>
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card size="small" title={t('vm.monitor.cpuHistory')}>
            <ResponsiveContainer width="100%" height={200}>
              <AreaChart data={stats.cpuHistory.slice(-50)}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="timestamp" tick={{ fontSize: 10 }} tickFormatter={formatTimestamp} />
                <YAxis domain={[0, 100]} tickFormatter={(v) => `${v}%`} />
                <Tooltip
                  formatter={(value: number) => [`${value.toFixed(2)}%`, 'CPU']}
                  labelFormatter={formatTimestampLong}
                />
                <Area type="monotone" dataKey="value" stroke="#1890ff" fill="url(#colorCpuDetail)" strokeWidth={2} />
                <defs>
                  <linearGradient id="colorCpuDetail" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#1890ff" stopOpacity={0.4} />
                    <stop offset="95%" stopColor="#1890ff" stopOpacity={0} />
                  </linearGradient>
                </defs>
              </AreaChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card size="small" title={t('vm.monitor.memoryHistory')}>
            <ResponsiveContainer width="100%" height={200}>
              <AreaChart data={stats.memoryHistory.slice(-50)}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="timestamp" tick={{ fontSize: 10 }} tickFormatter={formatTimestamp} />
                <YAxis domain={[0, 100]} tickFormatter={(v) => `${v}%`} />
                <Tooltip
                  formatter={(value: number) => [`${value.toFixed(2)}%`, t('vm.monitor.memory')]}
                  labelFormatter={formatTimestampLong}
                />
                <Area type="monotone" dataKey="value" stroke="#722ed1" fill="url(#colorMemDetail)" strokeWidth={2} />
                <defs>
                  <linearGradient id="colorMemDetail" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#722ed1" stopOpacity={0.4} />
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
