import React, { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Card, Table, Button, Tag, Space, Modal, Form, Input, Select, Switch, message, Popconfirm, Row, Col, Statistic, Progress, Drawer } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, PlayCircleOutlined, PoweroffOutlined, ReloadOutlined, DatabaseOutlined, FolderOutlined } from '@ant-design/icons'
import { storageApi, StoragePool, StorageVolume } from '../../api/client'

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const StoragePools: React.FC = () => {
  const { t } = useTranslation()
  const [pools, setPools] = useState<StoragePool[]>([])
  const [loading, setLoading] = useState(false)
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingPool, setEditingPool] = useState<StoragePool | null>(null)
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 })
  const [form] = Form.useForm()

  const [volumeDrawerOpen, setVolumeDrawerOpen] = useState(false)
  const [selectedPool, setSelectedPool] = useState<StoragePool | null>(null)
  const [volumes, setVolumes] = useState<StorageVolume[]>([])
  const [volumeLoading, setVolumeLoading] = useState(false)
  const [volumeModalOpen, setVolumeModalOpen] = useState(false)
  const [volumeForm] = Form.useForm()

  const fetchPools = useCallback(async (page = 1, pageSize = 10) => {
    setLoading(true)
    try {
      const response = await storageApi.listPools({ page, page_size: pageSize })
      if (response.code === 0) {
        setPools(response.data || [])
        setPagination({
          current: page,
          pageSize,
          total: response.meta?.total || 0
        })
      }
    } catch (_error) {
      message.error(t('storage.failedToListPools'))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    fetchPools()
  }, [fetchPools])

  const poolTypeOptions = [
    { label: t('storage.dir'), value: 'dir' },
    { label: t('storage.fs'), value: 'fs' },
    { label: t('storage.logical'), value: 'logical' }
  ]

  const handleAdd = () => {
    setEditingPool(null)
    form.resetFields()
    form.setFieldsValue({
      poolType: 'dir',
      autostart: true
    })
    setIsModalOpen(true)
  }

  const handleEdit = (pool: StoragePool) => {
    setEditingPool(pool)
    form.setFieldsValue(pool)
    setIsModalOpen(true)
  }

  const handleDelete = async (id: string) => {
    try {
      await storageApi.deletePool(id)
      message.success(t('storage.deletePoolSuccess'))
      fetchPools(pagination.current, pagination.pageSize)
    } catch (_error) {
      message.error(t('storage.failedToDeletePool'))
    }
  }

  const handleSubmit = async (values: any) => {
    try {
      if (editingPool) {
        await storageApi.updatePool(editingPool.id, values)
        message.success(t('storage.updatePoolSuccess'))
      } else {
        await storageApi.createPool(values)
        message.success(t('storage.createPoolSuccess'))
      }
      setIsModalOpen(false)
      fetchPools(pagination.current, pagination.pageSize)
    } catch (_error) {
      message.error(t('storage.failedToCreatePool'))
    }
  }

  const handleStart = async (id: string) => {
    try {
      await storageApi.startPool(id)
      message.success(t('storage.startPoolSuccess'))
      fetchPools(pagination.current, pagination.pageSize)
    } catch (_error) {
      message.error(t('storage.failedToStartPool'))
    }
  }

  const handleStop = async (id: string) => {
    try {
      await storageApi.stopPool(id)
      message.success(t('storage.stopPoolSuccess'))
      fetchPools(pagination.current, pagination.pageSize)
    } catch (_error) {
      message.error(t('storage.failedToStopPool'))
    }
  }

  const handleRefresh = async (id: string) => {
    try {
      await storageApi.refreshPool(id)
      message.success(t('storage.refreshPoolSuccess'))
      fetchPools(pagination.current, pagination.pageSize)
    } catch (_error) {
      message.error(t('storage.failedToRefreshPool'))
    }
  }

  const handleTableChange = (paginationInfo: any) => {
    fetchPools(paginationInfo.current, paginationInfo.pageSize)
  }

  const fetchVolumes = async (poolId: string) => {
    setVolumeLoading(true)
    try {
      const response = await storageApi.listVolumes(poolId)
      if (response.code === 0) {
        setVolumes(response.data || [])
      }
    } catch (_error) {
      message.error(t('storage.failedToListVolumes'))
    } finally {
      setVolumeLoading(false)
    }
  }

  const handleViewVolumes = (pool: StoragePool) => {
    setSelectedPool(pool)
    setVolumeDrawerOpen(true)
    fetchVolumes(pool.id)
  }

  const handleCreateVolume = async (values: any) => {
    if (!selectedPool) return
    try {
      await storageApi.createVolume(selectedPool.id, values)
      message.success(t('storage.createVolumeSuccess'))
      setVolumeModalOpen(false)
      volumeForm.resetFields()
      fetchVolumes(selectedPool.id)
    } catch (_error) {
      message.error(t('storage.failedToCreateVolume'))
    }
  }

  const handleDeleteVolume = async (volumeId: string) => {
    if (!selectedPool) return
    try {
      await storageApi.deleteVolume(selectedPool.id, volumeId)
      message.success(t('storage.deleteVolumeSuccess'))
      fetchVolumes(selectedPool.id)
    } catch (_error) {
      message.error(t('storage.failedToDeleteVolume'))
    }
  }

  const columns = [
    {
      title: t('storage.poolName'),
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: StoragePool) => (
        <a onClick={() => handleViewVolumes(record)}>{name}</a>
      )
    },
    {
      title: t('storage.poolType'),
      dataIndex: 'poolType',
      key: 'poolType',
      render: (type: string) => {
        const option = poolTypeOptions.find(t => t.value === type)
        return <Tag>{option?.label || type}</Tag>
      }
    },
    {
      title: t('storage.poolPath'),
      dataIndex: 'targetPath',
      key: 'targetPath'
    },
    {
      title: t('storage.capacity'),
      key: 'capacity',
      render: (_: any, record: StoragePool) => {
        const usage = record.capacity > 0 ? Math.round((record.used / record.capacity) * 100) : 0
        return (
          <div style={{ minWidth: 150 }}>
            <div style={{ marginBottom: 4 }}>
              <span>{formatBytes(record.used)}</span>
              <span style={{ color: '#999', marginLeft: 8 }}>/ {formatBytes(record.capacity)}</span>
            </div>
            <Progress percent={usage} size="small" showInfo={false} />
          </div>
        )
      }
    },
    {
      title: t('storage.usage'),
      key: 'usage',
      render: (_: any, record: StoragePool) => {
        const usage = record.capacity > 0 ? Math.round((record.used / record.capacity) * 100) : 0
        return <span>{usage}%</span>
      }
    },
    {
      title: t('storage.active'),
      dataIndex: 'active',
      key: 'active',
      render: (active: boolean) => (
        <Tag color={active ? 'green' : 'default'}>
          {active ? t('storage.active') : t('storage.inactive')}
        </Tag>
      )
    },
    {
      title: t('storage.autostart'),
      dataIndex: 'autostart',
      key: 'autostart',
      render: (autostart: boolean) => (
        <Tag color={autostart ? 'blue' : 'default'}>
          {autostart ? t('option.on') : t('option.off')}
        </Tag>
      )
    },
    {
      title: t('table.action'),
      key: 'actions',
      render: (_: any, record: StoragePool) => (
        <Space>
          {record.active ? (
            <Button
              type="text"
              danger
              icon={<PoweroffOutlined />}
              onClick={() => handleStop(record.id)}
            />
          ) : (
            <Button
              type="text"
              icon={<PlayCircleOutlined />}
              onClick={() => handleStart(record.id)}
            />
          )}
          <Button
            type="text"
            icon={<ReloadOutlined />}
            onClick={() => handleRefresh(record.id)}
          />
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          />
          <Popconfirm
            title={t('popconfirm.deletePool')}
            onConfirm={() => handleDelete(record.id)}
          >
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      )
    }
  ]

  const volumeColumns = [
    {
      title: t('storage.volumeName'),
      dataIndex: 'name',
      key: 'name'
    },
    {
      title: t('storage.volumeSize'),
      key: 'size',
      render: (_: any, record: StorageVolume) => (
        <span>{formatBytes(record.capacity)}</span>
      )
    },
    {
      title: t('storage.volumeFormat'),
      dataIndex: 'format',
      key: 'format',
      render: (format: string) => format ? <Tag>{format}</Tag> : '-'
    },
    {
      title: t('storage.volumePath'),
      dataIndex: 'path',
      key: 'path',
      ellipsis: true
    },
    {
      title: t('table.action'),
      key: 'actions',
      render: (_: any, record: StorageVolume) => (
        <Popconfirm
          title={t('popconfirm.deleteVolume')}
          onConfirm={() => handleDeleteVolume(record.id)}
        >
          <Button type="text" danger icon={<DeleteOutlined />} />
        </Popconfirm>
      )
    }
  ]

  const activeCount = pools.filter(p => p.active).length
  const inactiveCount = pools.filter(p => !p.active).length
  const totalCapacity = pools.reduce((sum, p) => sum + p.capacity, 0)
  const totalUsed = pools.reduce((sum, p) => sum + p.used, 0)

  return (
    <div>
      <Card>
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={4}>
            <Statistic 
              title={t('storage.totalPools')} 
              value={pagination.total} 
              prefix={<DatabaseOutlined />} 
            />
          </Col>
          <Col span={4}>
            <Statistic 
              title={t('storage.activePools')} 
              value={activeCount} 
              valueStyle={{ color: '#3f8600' }}
            />
          </Col>
          <Col span={4}>
            <Statistic 
              title={t('storage.inactivePools')} 
              value={inactiveCount} 
              valueStyle={{ color: '#cf1322' }}
            />
          </Col>
          <Col span={6}>
            <Statistic 
              title={t('storage.totalCapacity')} 
              value={formatBytes(totalCapacity)} 
            />
          </Col>
          <Col span={6}>
            <Statistic 
              title={t('storage.totalUsed')} 
              value={formatBytes(totalUsed)} 
              valueStyle={{ color: totalUsed > totalCapacity * 0.8 ? '#cf1322' : '#3f8600' }}
            />
          </Col>
        </Row>

        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
          <Button onClick={() => fetchPools(pagination.current, pagination.pageSize)}>
            <ReloadOutlined /> {t('common.refresh')}
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            {t('storage.createPool')}
          </Button>
        </div>

        <Table
          columns={columns}
          dataSource={pools}
          rowKey="id"
          loading={loading}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            showTotal: (total) => `${t('common.total')} ${total} ${t('storage.items')}`
          }}
          onChange={handleTableChange}
        />
      </Card>

      <Modal
        title={editingPool ? t('storage.editPool') : t('storage.createPool')}
        open={isModalOpen}
        onCancel={() => setIsModalOpen(false)}
        onOk={() => form.submit()}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
        >
          <Form.Item
            name="name"
            label={t('storage.poolName')}
            rules={[{ required: true, message: t('storage.poolNameRequired') }]}
          >
            <Input placeholder={t('storage.poolNamePlaceholder')} disabled={!!editingPool} />
          </Form.Item>

          <Form.Item
            name="description"
            label={t('common.description')}
          >
            <Input.TextArea rows={2} placeholder={t('common.descriptionPlaceholder')} />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="poolType"
                label={t('storage.poolType')}
                rules={[{ required: true }]}
              >
                <Select options={poolTypeOptions} disabled={!!editingPool} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="autostart"
                label={t('storage.autostart')}
                valuePropName="checked"
              >
                <Switch checkedChildren={t('option.on')} unCheckedChildren={t('option.off')} />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            noStyle
            shouldUpdate={(prevValues, currentValues) => prevValues.poolType !== currentValues.poolType}
          >
            {({ getFieldValue }) => {
              const poolType = getFieldValue('poolType')
              return (
                <>
                  <Form.Item
                    name="targetPath"
                    label={t('storage.targetPath')}
                    rules={[{ required: poolType === 'dir' || poolType === 'fs', message: t('storage.poolPathRequired') }]}
                  >
                    <Input placeholder={t('storage.poolPathPlaceholder')} />
                  </Form.Item>

                  {poolType === 'fs' && (
                    <Form.Item
                      name="sourcePath"
                      label={t('storage.sourcePath')}
                    >
                      <Input placeholder={t('storage.sourcePathPlaceholder')} />
                    </Form.Item>
                  )}
                </>
              )
            }}
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={
          <Space>
            <FolderOutlined />
            {selectedPool?.name} - {t('storage.volumes')}
          </Space>
        }
        placement="right"
        width={800}
        open={volumeDrawerOpen}
        onClose={() => setVolumeDrawerOpen(false)}
      >
        <div style={{ marginBottom: 16 }}>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setVolumeModalOpen(true)}>
            {t('storage.createVolume')}
          </Button>
        </div>

        <Table
          columns={volumeColumns}
          dataSource={volumes}
          rowKey="id"
          loading={volumeLoading}
          pagination={{ pageSize: 10 }}
        />
      </Drawer>

      <Modal
        title={t('storage.createVolume')}
        open={volumeModalOpen}
        onCancel={() => {
          setVolumeModalOpen(false)
          volumeForm.resetFields()
        }}
        onOk={() => volumeForm.submit()}
      >
        <Form
          form={volumeForm}
          layout="vertical"
          onFinish={handleCreateVolume}
        >
          <Form.Item
            name="name"
            label={t('storage.volumeName')}
            rules={[{ required: true, message: t('storage.volumeNameRequired') }]}
          >
            <Input placeholder={t('storage.volumeNamePlaceholder')} />
          </Form.Item>

          <Form.Item
            name="capacity"
            label={t('storage.volumeSize')}
            rules={[{ required: true, message: t('storage.volumeSizeRequired') }]}
          >
            <Input type="number" placeholder="10737418240" suffix="bytes" />
          </Form.Item>

          <Form.Item
            name="format"
            label={t('storage.volumeFormat')}
            initialValue="qcow2"
          >
            <Select
              options={[
                { label: 'qcow2', value: 'qcow2' },
                { label: 'raw', value: 'raw' },
                { label: 'vmdk', value: 'vmdk' }
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default StoragePools
