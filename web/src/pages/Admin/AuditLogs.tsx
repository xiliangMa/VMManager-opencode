import React, { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Table, Card, Tag, Space, Input, Select, Button } from 'antd'
import { ExportOutlined, SearchOutlined } from '@ant-design/icons'
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

  const { data, loading, pagination } = useTable<AuditLog>({
    api: systemApi.getAuditLogs
  })

  const handleExport = async () => {
    try {
      const response = await systemApi.getAuditLogs()
      const csvContent = [
        'ID,User,Action,Resource,IP,Status,Created At',
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
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      render: (id: string) => id.substring(0, 8)
    },
    {
      title: 'User',
      dataIndex: 'username',
      key: 'username'
    },
    {
      title: 'Action',
      dataIndex: 'action',
      key: 'action',
      render: (action: string) => <Tag>{action}</Tag>
    },
    {
      title: 'Resource',
      key: 'resource',
      render: (_: any, record: AuditLog) => (
        <span>{record.resource_type}: {record.resource_id.substring(0, 8)}</span>
      )
    },
    {
      title: 'IP Address',
      dataIndex: 'ip_address',
      key: 'ip_address'
    },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={statusColors[status] || 'default'}>
          {status}
        </Tag>
      )
    },
    {
      title: 'Created At',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm:ss')
    }
  ]

  return (
    <Card
      title={t('admin.auditLogs')}
      extra={
        <Button icon={<ExportOutlined />} onClick={handleExport}>
          Export
        </Button>
      }
    >
      <Space style={{ marginBottom: 16 }}>
        <Input
          placeholder="Search"
          prefix={<SearchOutlined />}
          style={{ width: 200 }}
        />
        <Select
          placeholder="Action"
          allowClear
          style={{ width: 150 }}
          value={actionFilter || undefined}
          onChange={(value) => setActionFilter(value || '')}
          options={[
            { label: 'Login', value: 'login' },
            { label: 'VM Create', value: 'vm.create' },
            { label: 'VM Delete', value: 'vm.delete' },
            { label: 'Template Upload', value: 'template.upload' }
          ]}
        />
        <Select
          placeholder="Status"
          allowClear
          style={{ width: 120 }}
          value={statusFilter || undefined}
          onChange={(value) => setStatusFilter(value || '')}
          options={[
            { label: 'Success', value: 'success' },
            { label: 'Failed', value: 'failed' }
          ]}
        />
      </Space>

      <Table
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        pagination={pagination}
      />
    </Card>
  )
}

export default AuditLogs
