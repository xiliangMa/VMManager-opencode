import React, { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Card, Table, Button, Tag, Space, message, Select, Statistic, Row, Col } from 'antd'
import { ReloadOutlined, CheckCircleOutlined, ExclamationCircleOutlined } from '@ant-design/icons'
import { alertHistoryApi, AlertHistory as AlertHistoryType } from '../../api/client'

const AlertHistoryPage: React.FC = () => {
  const { t } = useTranslation()
  const [histories, setHistories] = useState<AlertHistoryType[]>([])
  const [loading, setLoading] = useState(false)
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 })
  const [stats, setStats] = useState({ total: 0, critical: 0, warning: 0, info: 0 })
  const [statusFilter, setStatusFilter] = useState<string>('')

  const fetchHistories = useCallback(async (page = 1, pageSize = 10) => {
    setLoading(true)
    try {
      const params: any = { page, page_size: pageSize }
      if (statusFilter) {
        params.status = statusFilter
      }
      const response = await alertHistoryApi.list(params)
      if (response.code === 0) {
        setHistories(response.data || [])
        setPagination({
          current: page,
          pageSize,
          total: response.meta?.total || 0
        })
      }
    } catch (_error) {
      message.error(t('alert.loadingHistory'))
    } finally {
      setLoading(false)
    }
  }, [t, statusFilter])

  const fetchStats = useCallback(async () => {
    try {
      const response = await alertHistoryApi.getStats()
      if (response.code === 0) {
        setStats(response.data || { total: 0, critical: 0, warning: 0, info: 0 })
      }
    } catch (_error) {
    }
  }, [])

  useEffect(() => {
    fetchHistories()
    fetchStats()
  }, [fetchHistories, fetchStats])

  const handleResolve = async (id: string) => {
    try {
      await alertHistoryApi.resolve(id)
      message.success(t('alert.historyResolved'))
      fetchHistories(pagination.current, pagination.pageSize)
      fetchStats()
    } catch (error) {
      message.error(t('alert.failedToResolveHistory'))
    }
  }

  const handleTableChange = (paginationInfo: any) => {
    fetchHistories(paginationInfo.current, paginationInfo.pageSize)
  }

  const handleStatusFilter = (value: string) => {
    setStatusFilter(value)
    fetchHistories(1, pagination.pageSize)
  }

  const severityOptions = [
    { label: t('severity.critical'), value: 'critical', color: 'red' },
    { label: t('severity.warning'), value: 'warning', color: 'orange' },
    { label: t('severity.info'), value: 'info', color: 'blue' }
  ]

  const statusOptions = [
    { label: t('status.triggered'), value: 'triggered' },
    { label: t('status.resolved'), value: 'resolved' }
  ]

  const columns = [
    {
      title: t('alerts.time'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (time: string) => new Date(time).toLocaleString()
    },
    {
      title: t('alerts.severity'),
      dataIndex: 'severity',
      key: 'severity',
      render: (severity: string) => {
        const option = severityOptions.find(s => s.value === severity)
        return <Tag color={option?.color}>{option?.label}</Tag>
      }
    },
    {
      title: t('alerts.metric'),
      dataIndex: 'metric',
      key: 'metric',
      render: (metric: string) => {
        const metricLabels: Record<string, string> = {
          cpu_usage: t('metric.cpuUsage'),
          memory_usage: t('metric.memoryUsage'),
          disk_usage: t('metric.diskUsage'),
          network_in: t('metric.networkIn'),
          network_out: t('metric.networkOut'),
          vm_status: t('metric.vmStatus')
        }
        return metricLabels[metric] || metric
      }
    },
    {
      title: t('alerts.condition'),
      key: 'condition',
      render: (_: any, record: AlertHistoryType) => (
        <span>
          {record.currentValue} {record.condition} {record.threshold}
        </span>
      )
    },
    {
      title: t('alerts.message'),
      dataIndex: 'message',
      key: 'message',
      ellipsis: true
    },
    {
      title: t('alerts.status'),
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const option = statusOptions.find(s => s.value === status)
        const color = status === 'triggered' ? 'red' : 'green'
        const icon = status === 'triggered' ? <ExclamationCircleOutlined /> : <CheckCircleOutlined />
        return (
          <Tag color={color} icon={icon}>
            {option?.label || status}
          </Tag>
        )
      }
    },
    {
      title: t('table.action'),
      key: 'actions',
      render: (_: any, record: AlertHistoryType) => (
        <Space>
          {record.status === 'triggered' && (
            <Button
              type="link"
              size="small"
              onClick={() => handleResolve(record.id)}
            >
              {t('alert.resolve')}
            </Button>
          )}
        </Space>
      )
    }
  ]

  return (
    <div>
      <Card>
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={6}>
            <Statistic
              title={t('alerts.totalAlerts')}
              value={stats.total}
              valueStyle={{ color: '#cf1322' }}
              prefix={<ExclamationCircleOutlined />}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title={t('alerts.severityCritical')}
              value={stats.critical}
              valueStyle={{ color: '#cf1322' }}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title={t('alerts.severityWarning')}
              value={stats.warning}
              valueStyle={{ color: '#faad14' }}
            />
          </Col>
          <Col span={6}>
            <Statistic
              title={t('alerts.severityInfo')}
              value={stats.info}
              valueStyle={{ color: '#1890ff' }}
            />
          </Col>
        </Row>

        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
          <Space>
            <Select
              placeholder={t('alerts.status')}
              style={{ width: 150 }}
              allowClear
              onChange={handleStatusFilter}
              options={statusOptions}
            />
            <Button icon={<ReloadOutlined />} onClick={() => fetchHistories()}>
              {t('common.refresh')}
            </Button>
          </Space>
        </div>

        <Table
          columns={columns}
          dataSource={histories}
          rowKey="id"
          loading={loading}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            showTotal: (total) => `${t('common.total')} ${total} ${t('alerts.historyItems')}`
          }}
          onChange={handleTableChange}
        />
      </Card>
    </div>
  )
}

export default AlertHistoryPage
