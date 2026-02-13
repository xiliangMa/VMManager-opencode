import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Table, Button, Tag, Space, Card, Input, Select, message, Popconfirm } from 'antd'
import { PlusOutlined, SearchOutlined, VideoCameraOutlined, EditOutlined, DeleteOutlined, PlayCircleOutlined, PoweroffOutlined, SyncOutlined } from '@ant-design/icons'
import { vmsApi, VM } from '../../api/client'
import { useTable } from '../../hooks/useTable'
import dayjs from 'dayjs'

const VMs: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [lockingVms, setLockingVms] = useState<Set<string>>(new Set())

  const { data, loading, pagination, refresh, search, setSearch } = useTable<VM>({
    api: vmsApi.list
  })

  useEffect(() => {
    if (lockingVms.size > 0) {
      const interval = setInterval(() => {
        refresh()
        const stillLocked = Array.from(lockingVms).some(id => {
          const vm = data?.find((v: VM) => v.id === id)
          return vm && !['running', 'stopped'].includes(vm.status)
        })
        if (!stillLocked) {
          setLockingVms(new Set())
        }
      }, 2000)
      return () => clearInterval(interval)
    }
  }, [lockingVms.size, data, refresh])

  const statusColors: Record<string, string> = {
    running: 'green',
    stopped: 'red',
    suspended: 'orange',
    pending: 'blue',
    creating: 'processing',
    error: 'error',
    starting: 'processing',
    stopping: 'processing'
  }

  const isVmLocked = (id: string, status: string) => {
    return lockingVms.has(id) || ['starting', 'stopping', 'creating'].includes(status)
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
      render: (_: any, record: VM) => {
        const locked = isVmLocked(record.id, record.status)
        return (
          <Space>
            <Button
              type="text"
              icon={<VideoCameraOutlined />}
              disabled={locked || record.status !== 'running'}
              onClick={() => navigate(`/vms/${record.id}/console`)}
            />
            {!locked && (record.status === 'stopped' || record.status === 'pending' || record.status === 'creating') && (
              <Button
                type="text"
                icon={<PlayCircleOutlined />}
                onClick={() => handleStart(record.id)}
              />
            )}
            {record.status === 'running' && !locked && (
              <Button
                type="text"
                danger
                icon={<PoweroffOutlined />}
                onClick={() => handleStop(record.id)}
              />
            )}
            {locked && (
              <Tag color="processing" icon={<SyncOutlined spin />}>{t('vm.operationInProgress')}</Tag>
            )}
            <Button
              type="text"
              icon={<EditOutlined />}
              disabled={locked}
              onClick={() => navigate(`/vms/${record.id}/edit`)}
            />
            <Popconfirm
              title={t('common.delete')}
              description={t('popconfirm.deleteVm')}
              onConfirm={() => handleDelete(record.id)}
              disabled={locked}
            >
              <Button type="text" danger icon={<DeleteOutlined />} disabled={locked} />
            </Popconfirm>
          </Space>
        )
      }
    }
  ]

  const handleDelete = async (id: string) => {
    setLockingVms(prev => new Set(prev).add(id))
    try {
      await vmsApi.delete(id)
      message.success(t('common.success'))
      refresh()
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || t('common.error')
      message.error(errorMessage)
      setLockingVms(prev => {
        const next = new Set(prev)
        next.delete(id)
        return next
      })
    }
  }

  const handleStart = async (id: string) => {
    setLockingVms(prev => new Set(prev).add(id))
    try {
      await vmsApi.start(id)
      message.success(t('vm.startSuccess'))
      refresh()
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || t('vm.startFailed')
      message.error(errorMessage)
      setLockingVms(prev => {
        const next = new Set(prev)
        next.delete(id)
        return next
      })
    }
  }

  const handleStop = async (id: string) => {
    setLockingVms(prev => new Set(prev).add(id))
    try {
      await vmsApi.stop(id)
      message.success(t('vm.stopSuccess'))
      refresh()
    } catch (error: any) {
      const errorMessage = error?.response?.data?.message || error?.message || t('vm.stopFailed')
      message.error(errorMessage)
      setLockingVms(prev => {
        const next = new Set(prev)
        next.delete(id)
        return next
      })
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
