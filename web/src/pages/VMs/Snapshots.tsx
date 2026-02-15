import React, { useState, useEffect, useCallback } from 'react'
import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Card, Table, Button, Tag, Space, Modal, Form, Input, message, Popconfirm, Empty, Tooltip } from 'antd'
import { PlusOutlined, DeleteOutlined, UndoOutlined, SyncOutlined, CheckCircleOutlined } from '@ant-design/icons'
import { snapshotApi, vmsApi, VMSnapshot } from '../../api/client'
import dayjs from 'dayjs'

interface VMSnapshotsProps {
  vmId?: string
  vmStatus?: string
}

const VMSnapshots: React.FC<VMSnapshotsProps> = ({ vmId: propVmId, vmStatus: propVmStatus }) => {
  const { id: routeId } = useParams()
  const { t } = useTranslation()
  
  const vmId = propVmId || routeId || ''
  const [vmStatus, setVmStatus] = useState(propVmStatus || '')
  
  const [snapshots, setSnapshots] = useState<VMSnapshot[]>([])
  const [loading, setLoading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [form] = Form.useForm()

  useEffect(() => {
    if (!propVmStatus && vmId) {
      const fetchVM = async () => {
        try {
          const response = await vmsApi.get(vmId)
          if (response.code === 0) {
            setVmStatus(response.data.status)
          }
        } catch (error) {
          console.error('Failed to fetch VM status:', error)
        }
      }
      fetchVM()
    }
  }, [vmId, propVmStatus])

  const fetchSnapshots = useCallback(async () => {
    setLoading(true)
    try {
      const response = await snapshotApi.listSnapshots(vmId)
      if (response.code === 0) {
        setSnapshots(response.data || [])
      }
    } catch (error) {
      message.error(t('snapshot.failedToList'))
    } finally {
      setLoading(false)
    }
  }, [vmId, t])

  useEffect(() => {
    fetchSnapshots()
  }, [fetchSnapshots])

  const handleCreateSnapshot = async (values: any) => {
    try {
      await snapshotApi.createSnapshot(vmId, values)
      message.success(t('snapshot.createSuccess'))
      setModalOpen(false)
      form.resetFields()
      fetchSnapshots()
    } catch (error) {
      message.error(t('snapshot.failedToCreate'))
    }
  }

  const handleRestoreSnapshot = async (snapshotId: string) => {
    try {
      await snapshotApi.restoreSnapshot(vmId, snapshotId)
      message.success(t('snapshot.restoreSuccess'))
      fetchSnapshots()
    } catch (error) {
      message.error(t('snapshot.failedToRestore'))
    }
  }

  const handleDeleteSnapshot = async (snapshotId: string) => {
    try {
      await snapshotApi.deleteSnapshot(vmId, snapshotId)
      message.success(t('snapshot.deleteSuccess'))
      fetchSnapshots()
    } catch (error) {
      message.error(t('snapshot.failedToDelete'))
    }
  }

  const handleSyncSnapshots = async () => {
    try {
      const response = await snapshotApi.syncSnapshots(vmId)
      if (response.code === 0) {
        message.success(t('snapshot.syncSuccess', { count: response.data?.synced || 0 }))
        fetchSnapshots()
      }
    } catch (error) {
      message.error(t('snapshot.failedToSync'))
    }
  }

  const columns = [
    {
      title: t('snapshot.name'),
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: VMSnapshot) => (
        <Space>
          {name}
          {record.isCurrent && (
            <Tag color="green" icon={<CheckCircleOutlined />}>
              {t('snapshot.current')}
            </Tag>
          )}
        </Space>
      )
    },
    {
      title: t('common.description'),
      dataIndex: 'description',
      key: 'description',
      render: (desc: string) => desc || '-'
    },
    {
      title: t('snapshot.status'),
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const statusConfig: Record<string, { color: string; text: string }> = {
          created: { color: 'success', text: t('snapshot.statusCreated') },
          creating: { color: 'processing', text: t('snapshot.statusCreating') },
          deleting: { color: 'warning', text: t('snapshot.statusDeleting') }
        }
        const config = statusConfig[status] || { color: 'default', text: status }
        return <Tag color={config.color}>{config.text}</Tag>
      }
    },
    {
      title: t('snapshot.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm:ss')
    },
    {
      title: t('table.action'),
      key: 'actions',
      render: (_: any, record: VMSnapshot) => (
        <Space>
          <Tooltip title={t('snapshot.restoreTooltip')}>
            <Popconfirm
              title={t('snapshot.confirmRestore')}
              description={t('snapshot.restoreWarning')}
              onConfirm={() => handleRestoreSnapshot(record.id)}
              okText={t('common.confirm')}
              cancelText={t('common.cancel')}
            >
              <Button 
                type="text" 
                icon={<UndoOutlined />} 
                disabled={record.isCurrent || vmStatus === 'running'}
              />
            </Popconfirm>
          </Tooltip>
          {!record.isCurrent && (
            <Tooltip title={t('snapshot.deleteTooltip')}>
              <Popconfirm
                title={t('snapshot.confirmDelete')}
                onConfirm={() => handleDeleteSnapshot(record.id)}
                okText={t('common.confirm')}
                cancelText={t('common.cancel')}
              >
                <Button type="text" danger icon={<DeleteOutlined />} />
              </Popconfirm>
            </Tooltip>
          )}
        </Space>
      )
    }
  ]

  return (
    <Card>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <Space>
          <Button onClick={fetchSnapshots}>
            <SyncOutlined /> {t('common.refresh')}
          </Button>
          <Button onClick={handleSyncSnapshots}>
            <SyncOutlined /> {t('snapshot.syncFromLibvirt')}
          </Button>
        </Space>
        <Button 
          type="primary" 
          icon={<PlusOutlined />} 
          onClick={() => setModalOpen(true)}
          disabled={vmStatus !== 'running'}
        >
          {t('snapshot.createSnapshot')}
        </Button>
      </div>

      {vmStatus !== 'running' && (
        <div style={{ marginBottom: 16 }}>
          <Tag color="warning">{t('snapshot.vmNotRunning')}</Tag>
        </div>
      )}

      <Table
        columns={columns}
        dataSource={snapshots}
        rowKey="id"
        loading={loading}
        pagination={false}
        locale={{
          emptyText: (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description={t('snapshot.noSnapshots')}
            />
          )
        }}
      />

      <Modal
        title={t('snapshot.createSnapshot')}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => form.submit()}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleCreateSnapshot}
        >
          <Form.Item
            name="name"
            label={t('snapshot.name')}
            rules={[{ required: true, message: t('snapshot.nameRequired') }]}
          >
            <Input placeholder={t('snapshot.namePlaceholder')} />
          </Form.Item>

          <Form.Item
            name="description"
            label={t('common.description')}
          >
            <Input.TextArea rows={2} placeholder={t('snapshot.descriptionPlaceholder')} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}

export default VMSnapshots
