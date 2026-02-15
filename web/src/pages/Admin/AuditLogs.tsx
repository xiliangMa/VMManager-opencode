import React, { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Table, Card, Tag, Space, Input, Select, Button, Row, Col, Statistic } from 'antd'
import { ExportOutlined, SearchOutlined, FileTextOutlined } from '@ant-design/icons'
import { systemApi } from '../../api/client'
import { useTable } from '../../hooks/useTable'
import dayjs from 'dayjs'

interface AuditLog {
  id: string
  user_id: string
  username: string
  action: string
  resource_type: string
  resource_id: string
  details: string
  ip_address: string
  status: string
  created_at: string
}

const AuditLogs: React.FC = () => {
  const { t } = useTranslation()
  const [actionFilter, setActionFilter] = useState<string>('')
  const [statusFilter, setStatusFilter] = useState<string>('')

  const { data, loading, pagination, refresh } = useTable<AuditLog>({
    api: systemApi.getAuditLogs
  })

  const handleExport = async () => {
    try {
      const response = await systemApi.getAuditLogs()
      const csvContent = [
        t('table.name') + ',' + t('table.user') + ',' + t('table.action') + ',' + t('table.resource') + ',' + t('table.ipAddress') + ',' + t('table.status') + ',' + t('table.created'),
        ...response.data.map((log: AuditLog) =>
          `${log.id},${log.username},${log.action},${log.resource_type},${log.ip_address},${log.status},${log.created_at}`
        )
      ].join('\n')

      const blob = new Blob([csvContent], { type: 'text/csv' })
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `audit_logs_${dayjs().format('YYYY-MM-DD')}.csv`
      a.click()
      window.URL.revokeObjectURL(url)
    } catch (error) {
      console.error('Export failed:', error)
    }
  }

  const statusColors: Record<string, string> = {
    success: 'green',
    failed: 'red',
    pending: 'orange'
  }

  const columns = [
    {
      title: t('table.name'),
      dataIndex: 'id',
      key: 'id',
      render: (id: string) => id.substring(0, 8)
    },
    {
      title: t('table.user'),
      dataIndex: 'username',
      key: 'username'
    },
    {
      title: t('table.action'),
      dataIndex: 'action',
      key: 'action',
      render: (action: string) => <Tag>{action}</Tag>
    },
    {
      title: t('table.resource'),
      key: 'resource',
      render: (_: any, record: AuditLog) => (
        <span>{record.resource_type}: {record.resource_id.substring(0, 8)}</span>
      )
    },
    {
      title: t('table.ipAddress'),
      dataIndex: 'ip_address',
      key: 'ip_address'
    },
    {
      title: t('table.status'),
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={statusColors[status] || 'default'}>
          {status}
        </Tag>
      )
    },
    {
      title: t('table.created'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm:ss')
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

        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
          <Space>
            <Input
              placeholder={t('common.search')}
              prefix={<SearchOutlined />}
              style={{ width: 200 }}
            />
            <Select
              placeholder={t('table.action')}
              allowClear
              style={{ width: 150 }}
              value={actionFilter || undefined}
              onChange={(value) => setActionFilter(value || '')}
              options={[
                { label: t('status.login'), value: 'login' },
                { label: t('status.vmCreate'), value: 'vm.create' },
                { label: t('status.vmDelete'), value: 'vm.delete' },
                { label: t('status.templateUpload'), value: 'template.upload' }
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
            <Button onClick={refresh}>{t('common.refresh')}</Button>
          </Space>
          <Button type="primary" icon={<ExportOutlined />} onClick={handleExport}>
            {t('common.export')}
          </Button>
        </div>

        <Table
          columns={columns}
          dataSource={data}
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
            pagination.onChange(p.current || 1, p.pageSize || 10)
          }}
        />
      </Card>
    </div>
  )
}

export default AuditLogs
