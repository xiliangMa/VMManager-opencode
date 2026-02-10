import React, { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Table, Button, Tag, Space, Modal, Form, Input, message, Popconfirm, Empty, Tooltip, Descriptions } from 'antd'
import { PlusOutlined, DeleteOutlined, RestOutlined, CloudUploadOutlined } from '@ant-design/icons'
import { snapshotsApi, VMSnapshot } from '../../api/client'

const VMSnapshots: React.FC = () => {
  const { t } = useTranslation()
  const { id: vmId } = useParams()
  const navigate = useNavigate()

  const [snapshots, setSnapshots] = useState<VMSnapshot[]>([])
  const [loading, setLoading] = useState(false)
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [selectedSnapshot, setSelectedSnapshot] = useState<VMSnapshot | null>(null)
  const [isDetailOpen, setIsDetailOpen] = useState(false)
  const [form] = Form.useForm()

  const fetchSnapshots = useCallback(async () => {
    if (!vmId) return
    setLoading(true)
    try {
      const response = await snapshotsApi.list(vmId)
      if (response.code === 0) {
        setSnapshots(response.data || [])
      }
    } catch (error) {
      message.error(t('alert.loadingSnapshots'))
    } finally {
      setLoading(false)
    }
  }, [vmId, t])

  useEffect(() => {
    fetchSnapshots()
  }, [fetchSnapshots])

  const handleCreate = () => {
    setIsModalOpen(true)
    form.resetFields()
  }

  const handleSubmit = async (values: { name: string; description?: string }) => {
    if (!vmId) return
    try {
      await snapshotsApi.create(vmId, values)
      message.success(t('alert.snapshotCreatedSuccessfully'))
      setIsModalOpen(false)
      fetchSnapshots()
    } catch (error) {
      message.error(t('alert.failedToCreateSnapshot'))
    }
  }

  const handleRestore = async (snapshot: VMSnapshot) => {
    if (!vmId) return
    try {
      await snapshotsApi.restore(vmId, snapshot.name)
      message.success(`${t('alert.restoringTo')} ${snapshot.name}`)
      fetchSnapshots()
    } catch (error) {
      message.error(t('alert.failedToRestoreSnapshot'))
    }
  }

  const handleDelete = async (snapshot: VMSnapshot) => {
    if (!vmId) return
    try {
      await snapshotsApi.delete(vmId, snapshot.name)
      message.success(t('alert.snapshotDeletedSuccessfully'))
      fetchSnapshots()
    } catch (error) {
      message.error(t('alert.failedToDeleteSnapshot'))
    }
  }

  const handleViewDetail = (snapshot: VMSnapshot) => {
    setSelectedSnapshot(snapshot)
    setIsDetailOpen(true)
  }

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const getStateColor = (state: string) => {
    switch (state) {
      case 'running': return 'green'
      case 'shutdown': return 'orange'
      case 'disk-only': return 'blue'
      default: return 'default'
    }
  }

  const columns = [
    {
      title: t('table.name'),
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: VMSnapshot) => (
        <Button type="link" onClick={() => handleViewDetail(record)}>
          {name}
        </Button>
      )
    },
    {
      title: t('table.state'),
      dataIndex: 'state',
      key: 'state',
      render: (state: string) => (
        <Tag color={getStateColor(state)}>
          {state?.toUpperCase()}
        </Tag>
      )
    },
    {
      title: t('table.size'),
      dataIndex: 'size',
      key: 'size',
      render: (size: number) => formatSize(size)
    },
    {
      title: t('table.created'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time: string) => new Date(time).toLocaleString()
    },
    {
      title: t('table.action'),
      key: 'actions',
      render: (_: any, record: VMSnapshot) => (
        <Space>
          <Tooltip title={t('snapshot.restoreSnapshot')}>
            <Button
              type="text"
              icon={<RestOutlined />}
              onClick={() => handleRestore(record)}
            />
          </Tooltip>
          <Popconfirm
            title={t('popconfirm.deleteSnapshot')}
            description={t('message.actionCannotBeUndone')}
            onConfirm={() => handleDelete(record)}
          >
            <Tooltip title={t('common.delete')}>
              <Button type="text" danger icon={<DeleteOutlined />} />
            </Tooltip>
          </Popconfirm>
        </Space>
      )
    }
  ]

  return (
    <div>
      <Card
        title={
          <Space>
            <Button icon={<CloudUploadOutlined />} onClick={() => navigate(`/vms/${vmId}`)}>
              {t('common.back')}
            </Button>
            {t('vm.snapshots')}
            <Tag color="blue">{snapshots.length}</Tag>
          </Space>
        }
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            {t('snapshot.createSnapshot')}
          </Button>
        }
      >
        {snapshots.length === 0 ? (
          <Empty
            description={t('snapshot.noSnapshots')}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          >
            <Button type="primary" onClick={handleCreate}>
              {t('snapshot.createSnapshot')}
            </Button>
          </Empty>
        ) : (
          <Table
            columns={columns}
            dataSource={snapshots}
            rowKey="id"
            loading={loading}
            pagination={{ pageSize: 10 }}
          />
        )}
      </Card>

      <Modal
        title={t('snapshot.createSnapshot')}
        open={isModalOpen}
        onCancel={() => setIsModalOpen(false)}
        footer={null}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
        >
          <Form.Item
            name="name"
            label={t('snapshot.name')}
            rules={[
              { required: true, message: t('alert.pleaseEnterName') },
              { pattern: /^[a-zA-Z0-9][a-zA-Z0-9_-]*$/, message: t('alert.namePattern') }
            ]}
            extra={t('helper.uniqueIdentifier')}
          >
            <Input placeholder={t('placeholder.ruleName')} />
          </Form.Item>

          <Form.Item
            name="description"
            label={t('snapshot.description')}
            extra={t('placeholder.snapshotDescription')}
          >
            <Input.TextArea rows={3} placeholder={t('placeholder.snapshotDescription')} />
          </Form.Item>

          <Form.Item
            name="description"
            label="Description"
            extra="Optional description for this snapshot"
          >
            <Input.TextArea rows={3} placeholder="Describe the purpose of this snapshot" />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                Create
              </Button>
              <Button onClick={() => setIsModalOpen(false)}>
                Cancel
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('snapshot.createSnapshot')}
        open={isDetailOpen}
        onCancel={() => setIsDetailOpen(false)}
        footer={
          <Button onClick={() => setIsDetailOpen(false)}>
            {t('common.close')}
          </Button>
        }
      >
        {selectedSnapshot && (
          <Descriptions column={1}>
            <Descriptions.Item label={t('table.name')}>
              {selectedSnapshot.name}
            </Descriptions.Item>
            <Descriptions.Item label={t('table.state')}>
              <Tag color={getStateColor(selectedSnapshot.state || '')}>
                {(selectedSnapshot.state || '').toUpperCase()}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('table.size')}>
              {formatSize(selectedSnapshot.size)}
            </Descriptions.Item>
            <Descriptions.Item label={t('table.created')}>
              {new Date(selectedSnapshot.created_at).toLocaleString()}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  )
}

export default VMSnapshots
