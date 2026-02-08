import React, { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Table, Button, Tag, Space, Modal, Form, Input, message, Popconfirm, Empty, Tooltip, Descriptions } from 'antd'
import { PlusOutlined, DeleteOutlined, RestOutlined, CloudUploadOutlined } from '@ant-design/icons'
import { snapshotsApi, VMSnapshot } from '../../api/client'

const VMSnapshots: React.FC = () => {
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
      message.error('Failed to load snapshots')
    } finally {
      setLoading(false)
    }
  }, [vmId])

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
      message.success('Snapshot created successfully')
      setIsModalOpen(false)
      fetchSnapshots()
    } catch (error) {
      message.error('Failed to create snapshot')
    }
  }

  const handleRestore = async (snapshot: VMSnapshot) => {
    if (!vmId) return
    try {
      await snapshotsApi.restore(vmId, snapshot.name)
      message.success(`Restoring to snapshot: ${snapshot.name}`)
      fetchSnapshots()
    } catch (error) {
      message.error('Failed to restore snapshot')
    }
  }

  const handleDelete = async (snapshot: VMSnapshot) => {
    if (!vmId) return
    try {
      await snapshotsApi.delete(vmId, snapshot.name)
      message.success('Snapshot deleted')
      fetchSnapshots()
    } catch (error) {
      message.error('Failed to delete snapshot')
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
      title: 'Name',
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: VMSnapshot) => (
        <Button type="link" onClick={() => handleViewDetail(record)}>
          {name}
        </Button>
      )
    },
    {
      title: 'State',
      dataIndex: 'state',
      key: 'state',
      render: (state: string) => (
        <Tag color={getStateColor(state)}>
          {state?.toUpperCase()}
        </Tag>
      )
    },
    {
      title: 'Size',
      dataIndex: 'size',
      key: 'size',
      render: (size: number) => formatSize(size)
    },
    {
      title: 'Created',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time: string) => new Date(time).toLocaleString()
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: any, record: VMSnapshot) => (
        <Space>
          <Tooltip title="Restore">
            <Button
              type="text"
              icon={<RestOutlined />}
              onClick={() => handleRestore(record)}
            />
          </Tooltip>
          <Popconfirm
            title="Delete this snapshot?"
            description="This action cannot be undone."
            onConfirm={() => handleDelete(record)}
          >
            <Tooltip title="Delete">
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
              Back
            </Button>
            VM Snapshots
            <Tag color="blue">{snapshots.length}</Tag>
          </Space>
        }
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            Create Snapshot
          </Button>
        }
      >
        {snapshots.length === 0 ? (
          <Empty
            description="No snapshots available"
            image={Empty.PRESENTED_IMAGE_SIMPLE}
          >
            <Button type="primary" onClick={handleCreate}>
              Create First Snapshot
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
        title="Create Snapshot"
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
            label="Snapshot Name"
            rules={[
              { required: true, message: 'Please enter snapshot name' },
              { pattern: /^[a-zA-Z0-9][a-zA-Z0-9_-]*$/, message: 'Name can only contain letters, numbers, hyphens and underscores' }
            ]}
            extra="Unique identifier for this snapshot"
          >
            <Input placeholder="e.g., before-upgrade-2024" />
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
        title="Snapshot Details"
        open={isDetailOpen}
        onCancel={() => setIsDetailOpen(false)}
        footer={[
          <Button key="restore" type="primary" icon={<RestOutlined />} onClick={() => {
            if (selectedSnapshot) {
              handleRestore(selectedSnapshot)
              setIsDetailOpen(false)
            }
          }}>
            Restore
          </Button>,
          <Button key="close" onClick={() => setIsDetailOpen(false)}>
            Close
          </Button>
        ]}
      >
        {selectedSnapshot && (
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="Name">{selectedSnapshot.name}</Descriptions.Item>
            <Descriptions.Item label="State">
              <Tag color={getStateColor(selectedSnapshot.state || '')}>
                {selectedSnapshot.state?.toUpperCase() || 'UNKNOWN'}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="Size">{formatSize(selectedSnapshot.size)}</Descriptions.Item>
            <Descriptions.Item label="Created">
              {new Date(selectedSnapshot.created_at).toLocaleString()}
            </Descriptions.Item>
            <Descriptions.Item label="Updated">
              {selectedSnapshot.updated_at ? new Date(selectedSnapshot.updated_at).toLocaleString() : '-'}
            </Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  )
}

export default VMSnapshots
