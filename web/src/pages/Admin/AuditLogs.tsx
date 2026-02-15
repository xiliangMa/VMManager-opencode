import React, { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Table, Card, Tag, Space, Select, Button, Row, Col, Statistic, DatePicker, Drawer, Descriptions, message } from 'antd'
import { ExportOutlined, EyeOutlined, FileTextOutlined } from '@ant-design/icons'
import { systemApi } from '../../api/client'
import dayjs from 'dayjs'

const { RangePicker } = DatePicker

interface AuditLog {
  id: string
  userId: string
  username: string
  action: string
  resourceType: string
  resourceId: string
  details: string
  ipAddress: string
  userAgent: string
  status: string
  errorMessage: string
  createdAt: string
}

const AuditLogs: React.FC = () => {
  const { t } = useTranslation()
  const [loading, setLoading] = useState(false)
  const [logs, setLogs] = useState<AuditLog[]>([])
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 20,
    total: 0
  })
  const [actionFilter, setActionFilter] = useState<string>('')
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [dateRange, setDateRange] = useState<[dayjs.Dayjs | null, dayjs.Dayjs | null] | null>(null)
  const [detailDrawer, setDetailDrawer] = useState<{ visible: boolean; log: AuditLog | null }>({
    visible: false,
    log: null
  })

  const fetchLogs = async (page = 1, pageSize = 20) => {
    setLoading(true)
    try {
      const params: any = { page, page_size: pageSize }
      if (actionFilter) params.action = actionFilter
      if (statusFilter) params.status = statusFilter
      if (dateRange && dateRange[0] && dateRange[1]) {
        params.start_date = dateRange[0].format('YYYY-MM-DD')
        params.end_date = dateRange[1].format('YYYY-MM-DD')
      }

      const response = await systemApi.getAuditLogs(params)
      setLogs(response.data?.list || [])
      setPagination(prev => ({
        ...prev,
        current: page,
        pageSize,
        total: response.data?.meta?.total || 0
      }))
    } catch (_error) {
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchLogs()
  }, [actionFilter, statusFilter, dateRange])

  const handleExport = async () => {
    try {
      const params: any = {}
      if (actionFilter) params.action = actionFilter
      if (statusFilter) params.status = statusFilter
      if (dateRange && dateRange[0] && dateRange[1]) {
        params.start_date = dateRange[0].format('YYYY-MM-DD')
        params.end_date = dateRange[1].format('YYYY-MM-DD')
      }

      const response = await systemApi.getAuditLogs({ ...params, page_size: 1000 })
      const data = response.data?.list || []
      
      const csvContent = [
        'ID,' + t('table.user') + ',' + t('table.action') + ',' + t('table.resource') + ',' + t('table.ipAddress') + ',' + t('table.status') + ',' + t('table.created'),
        ...data.map((log: AuditLog) =>
          `${log.id},${log.username},${log.action},${log.resourceType}:${log.resourceId?.substring(0, 8) || ''},${log.ipAddress},${log.status},${log.createdAt}`
        )
      ].join('\n')

      const blob = new Blob(['\ufeff' + csvContent], { type: 'text/csv;charset=utf-8' })
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `audit_logs_${dayjs().format('YYYY-MM-DD')}.csv`
      a.click()
      window.URL.revokeObjectURL(url)
      message.success(t('common.success'))
    } catch (_error) {
      message.error(t('common.error'))
    }
  }

  const handleReset = () => {
    setActionFilter('')
    setStatusFilter('')
    setDateRange(null)
  }

  const statusColors: Record<string, string> = {
    success: 'green',
    failed: 'red',
    pending: 'orange'
  }

  const actionColors: Record<string, string> = {
    'auth.login': 'blue',
    'vm.create': 'green',
    'vm.delete': 'red',
    'vm.start': 'cyan',
    'vm.stop': 'orange',
    'vm.clone': 'purple',
    'template.delete': 'magenta'
  }

  const getActionLabel = (action: string) => {
    const actionMap: Record<string, string> = {
      'auth.login': t('audit.actionLogin'),
      'vm.create': t('audit.actionVMCreate'),
      'vm.delete': t('audit.actionVMDelete'),
      'vm.start': t('audit.actionVMStart'),
      'vm.stop': t('audit.actionVMStop'),
      'vm.clone': t('audit.actionVMClone'),
      'template.delete': t('audit.actionTemplateDelete')
    }
    return actionMap[action] || action
  }

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 100,
      render: (id: string) => id.substring(0, 8)
    },
    {
      title: t('table.user'),
      dataIndex: 'username',
      key: 'username',
      width: 120
    },
    {
      title: t('table.action'),
      dataIndex: 'action',
      key: 'action',
      width: 140,
      render: (action: string) => (
        <Tag color={actionColors[action] || 'default'}>
          {getActionLabel(action)}
        </Tag>
      )
    },
    {
      title: t('table.resource'),
      key: 'resource',
      width: 180,
      render: (_: any, record: AuditLog) => (
        <span>{record.resourceType}: {record.resourceId?.substring(0, 8) || '-'}</span>
      )
    },
    {
      title: t('table.ipAddress'),
      dataIndex: 'ipAddress',
      key: 'ipAddress',
      width: 140
    },
    {
      title: t('table.status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={statusColors[status] || 'default'}>
          {status === 'success' ? t('status.success') : t('status.failed')}
        </Tag>
      )
    },
    {
      title: t('table.created'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm:ss')
    },
    {
      title: t('table.action'),
      key: 'action_btn',
      width: 80,
      render: (_: any, record: AuditLog) => (
        <Button
          type="link"
          size="small"
          icon={<EyeOutlined />}
          onClick={() => setDetailDrawer({ visible: true, log: record })}
        >
          {t('common.view')}
        </Button>
      )
    }
  ]

  return (
    <div>
      <Card>
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={6}>
            <Statistic 
              title={t('audit.totalLogs')} 
              value={pagination.total} 
              prefix={<FileTextOutlined />} 
            />
          </Col>
        </Row>

        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between', flexWrap: 'wrap', gap: 8 }}>
          <Space wrap>
            <Select
              placeholder={t('table.action')}
              allowClear
              style={{ width: 160 }}
              value={actionFilter || undefined}
              onChange={(value) => setActionFilter(value || '')}
              options={[
                { label: t('audit.actionLogin'), value: 'auth.login' },
                { label: t('audit.actionVMCreate'), value: 'vm.create' },
                { label: t('audit.actionVMDelete'), value: 'vm.delete' },
                { label: t('audit.actionVMStart'), value: 'vm.start' },
                { label: t('audit.actionVMStop'), value: 'vm.stop' },
                { label: t('audit.actionVMClone'), value: 'vm.clone' },
                { label: t('audit.actionTemplateDelete'), value: 'template.delete' }
              ]}
            />
            <Select
              placeholder={t('table.status')}
              allowClear
              style={{ width: 120 }}
              value={statusFilter || undefined}
              onChange={(value) => setStatusFilter(value || '')}
              options={[
                { label: t('status.success'), value: 'success' },
                { label: t('status.failed'), value: 'failed' }
              ]}
            />
            <RangePicker
              value={dateRange}
              onChange={(dates) => setDateRange(dates)}
              style={{ width: 260 }}
            />
            <Button onClick={handleReset}>{t('common.reset')}</Button>
            <Button onClick={() => fetchLogs(pagination.current, pagination.pageSize)}>{t('common.refresh')}</Button>
          </Space>
          <Button type="primary" icon={<ExportOutlined />} onClick={handleExport}>
            {t('admin.exportAuditLogs')}
          </Button>
        </div>

        <Table
          columns={columns}
          dataSource={logs}
          rowKey="id"
          loading={loading}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            showTotal: (total) => `${t('common.total')} ${total} ${t('audit.logItems')}`
          }}
          onChange={(p) => {
            fetchLogs(p.current || 1, p.pageSize || 20)
          }}
        />
      </Card>

      <Drawer
        title={t('audit.logDetail')}
        placement="right"
        width={500}
        onClose={() => setDetailDrawer({ visible: false, log: null })}
        open={detailDrawer.visible}
      >
        {detailDrawer.log && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="ID">{detailDrawer.log.id}</Descriptions.Item>
            <Descriptions.Item label={t('table.user')}>{detailDrawer.log.username}</Descriptions.Item>
            <Descriptions.Item label={t('table.action')}>
              <Tag color={actionColors[detailDrawer.log.action] || 'default'}>
                {getActionLabel(detailDrawer.log.action)}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('table.resource')}>
              {detailDrawer.log.resourceType}: {detailDrawer.log.resourceId || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('table.ipAddress')}>{detailDrawer.log.ipAddress}</Descriptions.Item>
            <Descriptions.Item label={t('table.status')}>
              <Tag color={statusColors[detailDrawer.log.status] || 'default'}>
                {detailDrawer.log.status === 'success' ? t('status.success') : t('status.failed')}
              </Tag>
            </Descriptions.Item>
            {detailDrawer.log.errorMessage && (
              <Descriptions.Item label={t('audit.errorMessage')}>
                <span style={{ color: 'red' }}>{detailDrawer.log.errorMessage}</span>
              </Descriptions.Item>
            )}
            <Descriptions.Item label={t('table.created')}>
              {dayjs(detailDrawer.log.createdAt).format('YYYY-MM-DD HH:mm:ss')}
            </Descriptions.Item>
            <Descriptions.Item label={t('audit.userAgent')}>{detailDrawer.log.userAgent || '-'}</Descriptions.Item>
            {detailDrawer.log.details && (
              <Descriptions.Item label={t('audit.details')}>
                <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-all', maxHeight: 200, overflow: 'auto' }}>
                  {JSON.stringify(JSON.parse(detailDrawer.log.details), null, 2)}
                </pre>
              </Descriptions.Item>
            )}
          </Descriptions>
        )}
      </Drawer>
    </div>
  )
}

export default AuditLogs
