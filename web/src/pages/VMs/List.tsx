import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Table, Button, Tag, Space, Card, Input, Select, message, Popconfirm, Row, Col, Statistic, Dropdown, Modal, List, Badge } from 'antd'
import { PlusOutlined, SearchOutlined, VideoCameraOutlined, EditOutlined, DeleteOutlined, PlayCircleOutlined, PoweroffOutlined, SyncOutlined, DesktopOutlined, DownOutlined, ExclamationCircleOutlined } from '@ant-design/icons'
import { vmsApi, VM } from '../../api/client'
import { useTable } from '../../hooks/useTable'
import { useVMStatus } from '../../context/VMStatusContext'
import dayjs from 'dayjs'

const VMs: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [lockingVms, setLockingVms] = useState<Set<string>>(new Set())
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])
  const [batchLoading, setBatchLoading] = useState(false)
  const { statuses: realtimeStatuses, connected: wsConnected } = useVMStatus()

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
      render: (status: string, record: VM) => {
        const realtimeStatus = realtimeStatuses[record.id]
        const displayStatus = realtimeStatus?.status || status
        const isRealtime = !!realtimeStatus && wsConnected
        return (
          <Space>
            <Tag color={statusColors[displayStatus] || 'default'}>
              {t(`vm.${displayStatus}`) || displayStatus}
            </Tag>
            {isRealtime && (
              <Badge status="success" title={t('sync.syncing')} />
            )}
          </Space>
        )
      }
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

  const handleBatchStart = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('batch.selectVMs'))
      return
    }

    setBatchLoading(true)
    try {
      const result = await vmsApi.batchStart(selectedRowKeys as string[])
      if (result.data?.failed?.length > 0) {
        Modal.warning({
          title: t('batch.partialSuccess'),
          content: (
            <div>
              <p>{t('batch.successCount')}: {result.data.success.length}</p>
              <p>{t('batch.failedCount')}: {result.data.failed.length}</p>
              <List
                size="small"
                dataSource={result.data.failed}
                renderItem={(item: any) => (
                  <List.Item>
                    {item.name || item.vm_id}: {item.reason}
                  </List.Item>
                )}
              />
            </div>
          )
        })
      } else {
        message.success(t('batch.startSuccess'))
      }
      setSelectedRowKeys([])
      refresh()
    } catch (error: any) {
      message.error(error?.response?.data?.message || t('batch.startFailed'))
    } finally {
      setBatchLoading(false)
    }
  }

  const handleBatchStop = async (force = false) => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('batch.selectVMs'))
      return
    }

    Modal.confirm({
      title: force ? t('batch.forceStopConfirm') : t('batch.stopConfirm'),
      icon: <ExclamationCircleOutlined />,
      content: t('batch.selectedCount', { count: selectedRowKeys.length }),
      onOk: async () => {
        setBatchLoading(true)
        try {
          const result = await vmsApi.batchStop(selectedRowKeys as string[], force)
          if (result.data?.failed?.length > 0) {
            Modal.warning({
              title: t('batch.partialSuccess'),
              content: (
                <div>
                  <p>{t('batch.successCount')}: {result.data.success.length}</p>
                  <p>{t('batch.failedCount')}: {result.data.failed.length}</p>
                  <List
                    size="small"
                    dataSource={result.data.failed}
                    renderItem={(item: any) => (
                      <List.Item>
                        {item.name || item.vm_id}: {item.reason}
                      </List.Item>
                    )}
                  />
                </div>
              )
            })
          } else {
            message.success(t('batch.stopSuccess'))
          }
          setSelectedRowKeys([])
          refresh()
        } catch (error: any) {
          message.error(error?.response?.data?.message || t('batch.stopFailed'))
        } finally {
          setBatchLoading(false)
        }
      }
    })
  }

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning(t('batch.selectVMs'))
      return
    }

    Modal.confirm({
      title: t('batch.deleteConfirm'),
      icon: <ExclamationCircleOutlined />,
      content: t('batch.deleteWarning', { count: selectedRowKeys.length }),
      okType: 'danger',
      onOk: async () => {
        setBatchLoading(true)
        try {
          const result = await vmsApi.batchDelete(selectedRowKeys as string[])
          if (result.data?.failed?.length > 0) {
            Modal.warning({
              title: t('batch.partialSuccess'),
              content: (
                <div>
                  <p>{t('batch.successCount')}: {result.data.success.length}</p>
                  <p>{t('batch.failedCount')}: {result.data.failed.length}</p>
                  <List
                    size="small"
                    dataSource={result.data.failed}
                    renderItem={(item: any) => (
                      <List.Item>
                        {item.name || item.vm_id}: {item.reason}
                      </List.Item>
                    )}
                  />
                </div>
              )
            })
          } else {
            message.success(t('batch.deleteSuccess'))
          }
          setSelectedRowKeys([])
          refresh()
        } catch (error: any) {
          message.error(error?.response?.data?.message || t('batch.deleteFailed'))
        } finally {
          setBatchLoading(false)
        }
      }
    })
  }

  const rowSelection = {
    selectedRowKeys,
    onChange: (newSelectedRowKeys: React.Key[]) => {
      setSelectedRowKeys(newSelectedRowKeys)
    }
  }

  const batchMenuItems = [
    {
      key: 'start',
      label: t('batch.start'),
      icon: <PlayCircleOutlined />,
      onClick: handleBatchStart
    },
    {
      key: 'stop',
      label: t('batch.stop'),
      icon: <PoweroffOutlined />,
      onClick: () => handleBatchStop(false)
    },
    {
      key: 'force-stop',
      label: t('batch.forceStop'),
      icon: <PoweroffOutlined />,
      danger: true,
      onClick: () => handleBatchStop(true)
    },
    {
      type: 'divider' as const
    },
    {
      key: 'delete',
      label: t('batch.delete'),
      icon: <DeleteOutlined />,
      danger: true,
      onClick: handleBatchDelete
    }
  ]

  return (
    <div>
      <Card>
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={6}>
            <Statistic 
              title={t('common.totalVMs')} 
              value={pagination.total} 
              prefix={<DesktopOutlined />} 
            />
          </Col>
        </Row>

        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
          <Space>
            <Input
              placeholder={t('common.search')}
              prefix={<SearchOutlined />}
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              onPressEnter={refresh}
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
            <Button onClick={refresh}>{t('common.refresh')}</Button>
            {selectedRowKeys.length > 0 && (
              <Dropdown menu={{ items: batchMenuItems }} disabled={batchLoading}>
                <Button loading={batchLoading}>
                  {t('batch.operations')} ({selectedRowKeys.length}) <DownOutlined />
                </Button>
              </Dropdown>
            )}
          </Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => navigate('/vms/create')}>
            {t('vm.createVM')}
          </Button>
        </div>

        <Table
          rowSelection={rowSelection}
          columns={columns}
          dataSource={data}
          rowKey="id"
          loading={loading}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            showTotal: (total) => `${t('common.total')} ${total} ${t('vm.items')}`
          }}
          onChange={(p) => {
            pagination.onChange(p.current || 1, p.pageSize || 10)
          }}
        />
      </Card>
    </div>
  )
}

export default VMs
