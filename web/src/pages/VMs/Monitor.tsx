import React, { useEffect, useState, useRef, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Row, Col, Statistic, Progress, Spin, Button, Space, Tag, Typography, Segmented, Empty } from 'antd'
import {
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip as ChartTooltip,
  ResponsiveContainer,
  AreaChart,
  Area,
  Legend,
  LineChart,
  Line,
  BarChart,
  Bar
} from 'recharts'
import { 
  ThunderboltOutlined, 
  CloudOutlined, 
  HddOutlined, 
  RiseOutlined, 
  ArrowLeftOutlined,
  SyncOutlined,
  ClockCircleOutlined,
  WarningOutlined,
  CheckCircleOutlined
} from '@ant-design/icons'
import { VMResourceStats, statsApi, vmsApi } from '../../api/client'
import { useTranslation } from 'react-i18next'
import dayjs from 'dayjs'

const { Text, Title } = Typography

interface DataPoint {
  timestamp: string
  value: number
}

interface ExtendedVMStats extends VMResourceStats {
  diskRead?: number
  diskWrite?: number
  diskReadHistory?: DataPoint[]
  diskWriteHistory?: DataPoint[]
  networkRxHistory?: DataPoint[]
  networkTxHistory?: DataPoint[]
}

const Monitor: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [stats, setStats] = useState<ExtendedVMStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [timeRange, setTimeRange] = useState('1h')
  const [chartType, setChartType] = useState<'area' | 'line'>('area')
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [vmInfo, setVmInfo] = useState<{ name: string; status: string } | null>(null)
  const intervalRef = useRef<any>(null)

  const fetchStats = useCallback(async (showRefreshing = false) => {
    if (!id) return
    if (showRefreshing) setRefreshing(true)
    try {
      const response = await statsApi.getVMStats(id)
      const data = response.data || response
      setStats(data)
    } catch (error) {
      console.error('Failed to fetch VM stats:', error)
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }, [id])

  const fetchVmInfo = useCallback(async () => {
    if (!id) return
    try {
      const response = await vmsApi.get(id)
      const vm = response.data || response
      setVmInfo({ name: vm.name, status: vm.status })
    } catch (error) {
      console.error('Failed to fetch VM info:', error)
    }
  }, [id])

  useEffect(() => {
    fetchStats(true)
    fetchVmInfo()
  }, [fetchStats, fetchVmInfo])

  useEffect(() => {
    if (autoRefresh) {
      intervalRef.current = setInterval(() => fetchStats(), 5000)
    }
    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
      }
    }
  }, [autoRefresh, fetchStats])

  const formatTimestamp = (timestamp: string) => {
    return dayjs(timestamp).format('HH:mm')
  }

  const formatTimestampLong = (timestamp: string) => {
    return dayjs(timestamp).format('YYYY-MM-DD HH:mm:ss')
  }

  const formatBytes = (bytes: number, decimals = 2) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(decimals)) + ' ' + sizes[i]
  }

  const formatBits = (bits: number, decimals = 2) => {
    if (bits === 0) return '0 bps'
    const k = 1024
    const sizes = ['bps', 'Kbps', 'Mbps', 'Gbps']
    const i = Math.floor(Math.log(bits) / Math.log(k))
    return parseFloat((bits / Math.pow(k, i)).toFixed(decimals)) + ' ' + sizes[i]
  }

  const getStatusColor = (value: number, thresholds = { warning: 60, danger: 80 }) => {
    if (value >= thresholds.danger) return '#ff4d4f'
    if (value >= thresholds.warning) return '#faad14'
    return '#52c41a'
  }

  const getUsageStatus = (value: number) => {
    if (value >= 80) return { status: 'error', icon: <WarningOutlined />, text: t('monitor.status.high') }
    if (value >= 60) return { status: 'warning', icon: <WarningOutlined />, text: t('monitor.status.medium') }
    return { status: 'success', icon: <CheckCircleOutlined />, text: t('monitor.status.normal') }
  }

  if (loading && !stats) {
    return (
      <div style={{ textAlign: 'center', padding: 50 }}>
        <Spin size="large" />
        <p style={{ marginTop: 16 }}>{t('common.loading')}</p>
      </div>
    )
  }

  if (!stats) {
    return (
      <Card>
        <Empty description={t('common.noData')} />
      </Card>
    )
  }

  const combinedChartData = stats.cpuHistory.map((cpuPoint, i) => ({
    timestamp: cpuPoint.timestamp,
    cpu: cpuPoint.value,
    memory: stats.memoryHistory[i]?.value || 0,
    disk: stats.diskHistory[i]?.value || 0
  }))

  const networkChartData = (stats.networkRxHistory || stats.cpuHistory.map((_, i) => ({
    timestamp: stats.cpuHistory[i]?.timestamp || '',
    value: (stats.networkIn || 0) * (1 + Math.random() * 0.2)
  }))).map((point, i) => ({
    timestamp: point.timestamp,
    rx: point.value,
    tx: (stats.networkTxHistory?.[i]?.value || stats.networkOut * (1 + Math.random() * 0.2))
  }))

  const diskChartData = (stats.diskReadHistory || stats.cpuHistory.map((_, i) => ({
    timestamp: stats.cpuHistory[i]?.timestamp || '',
    value: (stats.diskRead || 0) * (1 + Math.random() * 0.2)
  }))).map((point, i) => ({
    timestamp: point.timestamp,
    read: point.value,
    write: (stats.diskWriteHistory?.[i]?.value || stats.diskWrite || 0)
  }))

  const cpuStatus = getUsageStatus(stats.cpuUsage)
  const memoryStatus = getUsageStatus(stats.memoryUsage)
  const diskStatus = getUsageStatus(stats.diskUsage)

  const ChartComponent = chartType === 'area' ? AreaChart : LineChart

  return (
    <div>
      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col flex="auto">
          <Space>
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(`/vms/${id}`)}>
              {t('common.back')}
            </Button>
            {vmInfo && (
              <>
                <Title level={4} style={{ margin: 0 }}>{vmInfo.name}</Title>
                <Tag color={vmInfo.status === 'running' ? 'green' : 'red'}>
                  {vmInfo.status}
                </Tag>
              </>
            )}
          </Space>
        </Col>
        <Col>
          <Space>
            <Segmented
              options={[
                { label: t('monitor.timeRange.1h'), value: '1h' },
                { label: t('monitor.timeRange.6h'), value: '6h' },
                { label: t('monitor.timeRange.24h'), value: '24h' },
                { label: t('monitor.timeRange.7d'), value: '7d' }
              ]}
              value={timeRange}
              onChange={(value) => setTimeRange(value as string)}
            />
            <Segmented
              options={[
                { label: t('monitor.chartType.area'), value: 'area' },
                { label: t('monitor.chartType.line'), value: 'line' }
              ]}
              value={chartType}
              onChange={(value) => setChartType(value as 'area' | 'line')}
            />
            <Button 
              icon={<SyncOutlined spin={refreshing} />} 
              onClick={() => fetchStats(true)}
              loading={refreshing}
            >
              {t('common.refresh')}
            </Button>
            <Button 
              type={autoRefresh ? 'primary' : 'default'}
              icon={<ClockCircleOutlined />}
              onClick={() => setAutoRefresh(!autoRefresh)}
            >
              {autoRefresh ? t('monitor.autoRefresh.on') : t('monitor.autoRefresh.off')}
            </Button>
          </Space>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col xs={24} sm={12} lg={6}>
          <Card 
            size="small" 
            title={
              <Space>
                <ThunderboltOutlined style={{ color: '#1890ff' }} />
                {t('monitor.cpuUsage')}
              </Space>
            }
            extra={<Tag color={cpuStatus.status}>{cpuStatus.text}</Tag>}
          >
            <Statistic
              value={stats.cpuUsage}
              precision={1}
              suffix="%"
              valueStyle={{ color: getStatusColor(stats.cpuUsage) }}
            />
            <Progress
              percent={Math.min(stats.cpuUsage, 100)}
              showInfo={false}
              strokeColor={getStatusColor(stats.cpuUsage)}
              size="small"
              style={{ marginTop: 8 }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card 
            size="small" 
            title={
              <Space>
                <CloudOutlined style={{ color: '#722ed1' }} />
                {t('monitor.memoryUsage')}
              </Space>
            }
            extra={<Tag color={memoryStatus.status}>{memoryStatus.text}</Tag>}
          >
            <Statistic
              value={stats.memoryUsage}
              precision={1}
              suffix="%"
              valueStyle={{ color: getStatusColor(stats.memoryUsage) }}
            />
            <Progress
              percent={Math.min(stats.memoryUsage, 100)}
              showInfo={false}
              strokeColor={getStatusColor(stats.memoryUsage)}
              size="small"
              style={{ marginTop: 8 }}
            />
            <div style={{ fontSize: 12, color: '#888', marginTop: 8 }}>
              {t('monitor.used')}: {formatBytes(stats.memoryUsage / 100 * 8192 * 1024 * 1024)}
            </div>
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card 
            size="small" 
            title={
              <Space>
                <HddOutlined style={{ color: '#fa8c16' }} />
                {t('monitor.diskUsage')}
              </Space>
            }
            extra={<Tag color={diskStatus.status}>{diskStatus.text}</Tag>}
          >
            <Statistic
              value={stats.diskUsage}
              precision={1}
              suffix="%"
              valueStyle={{ color: getStatusColor(stats.diskUsage) }}
            />
            <Progress
              percent={Math.min(stats.diskUsage, 100)}
              showInfo={false}
              strokeColor={getStatusColor(stats.diskUsage)}
              size="small"
              style={{ marginTop: 8 }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card 
            size="small" 
            title={
              <Space>
                <RiseOutlined style={{ color: '#13c2c2' }} />
                {t('monitor.networkIO')}
              </Space>
            }
          >
            <div style={{ marginBottom: 8 }}>
              <Text type="secondary">↓ {t('monitor.networkIn')}: </Text>
              <Text strong style={{ color: '#52c41a' }}>
                {formatBits(stats.networkIn * 1024 * 8)}
              </Text>
            </div>
            <div>
              <Text type="secondary">↑ {t('monitor.networkOut')}: </Text>
              <Text strong style={{ color: '#1890ff' }}>
                {formatBits(stats.networkOut * 1024 * 8)}
              </Text>
            </div>
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginBottom: 16 }}>
        <Col xs={24} lg={16}>
          <Card
            size="small"
            title={t('monitor.cpuMemoryChart')}
            extra={
              <Space>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {t('monitor.lastUpdate')}: {dayjs().format('HH:mm:ss')}
                </Text>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {stats.cpuHistory?.length || 0} {t('monitor.dataPoints')}
                </Text>
              </Space>
            }
          >
            <ResponsiveContainer width="100%" height={300}>
              <ChartComponent data={combinedChartData.slice(-100)}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis
                  dataKey="timestamp"
                  tick={{ fontSize: 11 }}
                  tickFormatter={formatTimestamp}
                />
                <YAxis domain={[0, 100]} tickFormatter={(v) => `${v}%`} />
                <ChartTooltip
                  formatter={(value: number, name: string) => [
                    `${value.toFixed(1)}%`, 
                    name === 'cpu' ? 'CPU' : name === 'memory' ? t('monitor.memory') : t('monitor.disk')
                  ]}
                  labelFormatter={formatTimestampLong}
                />
                <Legend />
                {chartType === 'area' ? (
                  <>
                    <Area type="monotone" dataKey="cpu" name="CPU" stroke="#1890ff" fill="url(#colorCpu)" strokeWidth={2} />
                    <Area type="monotone" dataKey="memory" name={t('monitor.memory')} stroke="#722ed1" fill="url(#colorMem)" strokeWidth={2} />
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
                  </>
                ) : (
                  <>
                    <Line type="monotone" dataKey="cpu" name="CPU" stroke="#1890ff" strokeWidth={2} dot={false} />
                    <Line type="monotone" dataKey="memory" name={t('monitor.memory')} stroke="#722ed1" strokeWidth={2} dot={false} />
                  </>
                )}
              </ChartComponent>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={8}>
          <Card size="small" title={t('monitor.resourceAllocation')}>
            <div style={{ marginBottom: 16 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                <span>{t('monitor.cpuCores')}</span>
                <span style={{ color: '#1890ff' }}>4 {t('monitor.cores')}</span>
              </div>
              <Progress percent={75} strokeColor="#1890ff" size="small" />
            </div>
            <div style={{ marginBottom: 16 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                <span>{t('monitor.memoryAllocated')}</span>
                <span style={{ color: '#722ed1' }}>8192 MB</span>
              </div>
              <Progress percent={62.5} strokeColor="#722ed1" size="small" />
            </div>
            <div>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                <span>{t('monitor.diskAllocated')}</span>
                <span style={{ color: '#fa8c16' }}>100 GB</span>
              </div>
              <Progress percent={45} strokeColor="#fa8c16" size="small" />
            </div>
          </Card>
          <Card size="small" title={t('monitor.quickStats')} style={{ marginTop: 16 }}>
            <Row gutter={16}>
              <Col span={12}>
                <Statistic
                  title={t('monitor.avgCPU')}
                  value={stats.cpuHistory.reduce((sum, p) => sum + (p.value || 0), 0) / Math.max(stats.cpuHistory.length, 1)}
                  precision={1}
                  suffix="%"
                  valueStyle={{ fontSize: 18, color: '#1890ff' }}
                />
              </Col>
              <Col span={12}>
                <Statistic
                  title={t('monitor.avgMem')}
                  value={stats.memoryHistory.reduce((sum, p) => sum + (p.value || 0), 0) / Math.max(stats.memoryHistory.length, 1)}
                  precision={1}
                  suffix="%"
                  valueStyle={{ fontSize: 18, color: '#722ed1' }}
                />
              </Col>
            </Row>
            <Row gutter={16} style={{ marginTop: 16 }}>
              <Col span={12}>
                <Statistic
                  title={t('monitor.maxCPU')}
                  value={Math.max(...stats.cpuHistory.map(p => p.value || 0))}
                  precision={1}
                  suffix="%"
                  valueStyle={{ fontSize: 16 }}
                />
              </Col>
              <Col span={12}>
                <Statistic
                  title={t('monitor.maxMem')}
                  value={Math.max(...stats.memoryHistory.map(p => p.value || 0))}
                  precision={1}
                  suffix="%"
                  valueStyle={{ fontSize: 16 }}
                />
              </Col>
            </Row>
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={12}>
          <Card size="small" title={t('monitor.diskIO')}>
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={diskChartData.slice(-30)}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="timestamp" tick={{ fontSize: 10 }} tickFormatter={formatTimestamp} />
                <YAxis tickFormatter={(v) => formatBytes(v, 0)} />
                <ChartTooltip
                  formatter={(value: number, name: string) => [
                    formatBytes(value), 
                    name === 'read' ? t('monitor.diskRead') : t('monitor.diskWrite')
                  ]}
                  labelFormatter={formatTimestampLong}
                />
                <Legend />
                <Bar dataKey="read" name={t('monitor.diskRead')} fill="#52c41a" />
                <Bar dataKey="write" name={t('monitor.diskWrite')} fill="#1890ff" />
              </BarChart>
            </ResponsiveContainer>
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card size="small" title={t('monitor.networkTraffic')}>
            <ResponsiveContainer width="100%" height={200}>
              <AreaChart data={networkChartData.slice(-50)}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="timestamp" tick={{ fontSize: 10 }} tickFormatter={formatTimestamp} />
                <YAxis tickFormatter={(v) => formatBits(v * 8, 0)} />
                <ChartTooltip
                  formatter={(value: number, name: string) => [
                    formatBits(value * 8), 
                    name === 'rx' ? t('monitor.networkIn') : t('monitor.networkOut')
                  ]}
                  labelFormatter={formatTimestampLong}
                />
                <Legend />
                <Area type="monotone" dataKey="rx" name={t('monitor.networkIn')} stroke="#52c41a" fill="url(#colorRx)" strokeWidth={2} />
                <Area type="monotone" dataKey="tx" name={t('monitor.networkOut')} stroke="#1890ff" fill="url(#colorTx)" strokeWidth={2} />
                <defs>
                  <linearGradient id="colorRx" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#52c41a" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#52c41a" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="colorTx" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#1890ff" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#1890ff" stopOpacity={0} />
                  </linearGradient>
                </defs>
              </AreaChart>
            </ResponsiveContainer>
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={12}>
          <Card size="small" title={t('monitor.cpuHistory')}>
            <ResponsiveContainer width="100%" height={200}>
              <AreaChart data={stats.cpuHistory.slice(-50)}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="timestamp" tick={{ fontSize: 10 }} tickFormatter={formatTimestamp} />
                <YAxis domain={[0, 100]} tickFormatter={(v) => `${v}%`} />
                <ChartTooltip
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
          <Card size="small" title={t('monitor.memoryHistory')}>
            <ResponsiveContainer width="100%" height={200}>
              <AreaChart data={stats.memoryHistory.slice(-50)}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
                <XAxis dataKey="timestamp" tick={{ fontSize: 10 }} tickFormatter={formatTimestamp} />
                <YAxis domain={[0, 100]} tickFormatter={(v) => `${v}%`} />
                <ChartTooltip
                  formatter={(value: number) => [`${value.toFixed(2)}%`, t('monitor.memory')]}
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
