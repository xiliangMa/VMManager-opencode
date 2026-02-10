import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Table, Button, Tag, Space, Card, Input, Select, message, Popconfirm } from 'antd'
import { PlusOutlined, SearchOutlined, VideoCameraOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons'
import { vmsApi, VM } from '../../api/client'
import { useTable } from '../../hooks/useTable'
import dayjs from 'dayjs'

const VMs: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [statusFilter, setStatusFilter] = useState<string>('')

  const { data, loading, pagination, refresh, search, setSearch } = useTable<VM>({
    api: vmsApi.list
  })

  const statusColors: Record<string, string> = {
    running: 'green',
    stopped: 'red',
    suspended: 'orange',
    pending: 'blue',
    creating: 'processing',
    error: 'error'
  }

  const columns = [
    {
      title: t('vm.name'),
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: VM) => (
        <a onClick={() => navigate(`/vms/${record.id}`)}>{text}</a>
      )
    },
    {
      title: t('vm.status'),
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={statusColors[status] || 'default'}>
          {t(`vm.${status}`) || status}
        </Tag>
      )
    },
    {
      title: t('vm.cpu'),
      dataIndex: 'cpuAllocated',
      key: 'cpu',
      render: (cpu: number) => cpu ? `${cpu} vCPU` : '-'
    },
    {
      title: t('vm.memory'),
      dataIndex: 'memoryAllocated',
      key: 'memory',
      render: (memory: number) => memory ? `${memory} MB` : '-'
    },
    {
      title: t('vm.disk'),
      dataIndex: 'diskAllocated',
      key: 'disk',
      render: (disk: number) => disk ? `${disk} GB` : '-'
    },
    {
      title: t('vm.ipAddress'),
      dataIndex: 'ipAddress',
      key: 'ipAddress',
      render: (ip: string) => ip || '-'
    },
    {
      title: t('detail.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (date: string) => date ? dayjs(date).format('YYYY-MM-DD HH:mm') : '-'
    },
    {
      title: t('common.edit'),
      key: 'actions',
      render: (_: any, record: VM) => (
        <Space>
          <Button 
            type="text" 
            icon={<VideoCameraOutlined />}
            onClick={() => navigate(`/vms/${record.id}/console`)}
          />
          <Button type="text" icon={<EditOutlined />} />
          <Popconfirm
            title={t('common.delete')}
            description={t('popconfirm.deleteVm')}
            onConfirm={() => handleDelete(record.id)}
          >
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      )
    }
  ]

  const handleDelete = async (id: string) => {
    try {
      await vmsApi.delete(id)
      message.success(t('common.success'))
      refresh()
    } catch (error) {
      message.error(t('common.error'))
    }
  }

  return (
    <Card
      title={t('vm.vmList')}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/vms/create')}>
          {t('vm.createVM')}
        </Button>
      }
    >
      <Space style={{ marginBottom: 16 }}>
        <Input
          placeholder={t('common.search')}
          prefix={<SearchOutlined />}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          style={{ width: 200 }}
        />
        <Select
          placeholder={t('vm.status')}
          allowClear
          style={{ width: 150 }}
          value={statusFilter || undefined}
          onChange={(value) => setStatusFilter(value || '')}
          options={[
            { label: t('vm.running'), value: 'running' },
            { label: t('vm.stopped'), value: 'stopped' },
            { label: t('vm.suspended'), value: 'suspended' }
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

export default VMs
