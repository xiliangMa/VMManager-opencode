import React, { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Table, Card, Button, Space, Tag, message, Popconfirm, Modal, Form, Input, InputNumber, Select } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, UserOutlined } from '@ant-design/icons'
import { usersApi, User } from '../../api/client'
import { useTable } from '../../hooks/useTable'
import dayjs from 'dayjs'

const Users: React.FC = () => {
  const { t } = useTranslation()
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingUser, setEditingUser] = useState<User | null>(null)
  const [form] = Form.useForm()
  const { data, loading, pagination, refresh } = useTable<User>({
    api: usersApi.list
  })

  const handleCreate = () => {
    setEditingUser(null)
    form.resetFields()
    setIsModalOpen(true)
  }

  const handleEdit = (user: User) => {
    setEditingUser(user)
    form.setFieldsValue(user)
    setIsModalOpen(true)
  }

  const handleSubmit = async (values: any) => {
    try {
      if (editingUser) {
        await usersApi.update(editingUser.id, values)
        message.success(t('common.success'))
      } else {
        await usersApi.create(values)
        message.success(t('common.success'))
      }
      setIsModalOpen(false)
      refresh()
    } catch (error) {
      message.error(t('common.error'))
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await usersApi.delete(id)
      message.success(t('common.success'))
      refresh()
    } catch (error) {
      message.error(t('common.error'))
    }
  }

  const roleColors: Record<string, string> = {
    admin: 'red',
    user: 'blue',
    viewer: 'green'
  }

  const columns = [
    {
      title: 'Username',
      dataIndex: 'username',
      key: 'username',
      render: (text: string) => (
        <Space>
          <UserOutlined />
          {text}
        </Space>
      )
    },
    {
      title: t('auth.email'),
      dataIndex: 'email',
      key: 'email'
    },
    {
      title: 'Role',
      dataIndex: 'role',
      key: 'role',
      render: (role: string) => (
        <Tag color={roleColors[role] || 'default'}>
          {role}
        </Tag>
      )
    },
    {
      title: 'CPU Quota',
      dataIndex: 'quota_cpu',
      key: 'quota_cpu',
      render: (quota: number) => quota || '-'
    },
    {
      title: 'Memory Quota (MB)',
      dataIndex: 'quota_memory',
      key: 'quota_memory',
      render: (quota: number) => quota || '-'
    },
    {
      title: 'VM Count',
      dataIndex: 'quota_vm_count',
      key: 'quota_vm_count',
      render: (quota: number) => quota || '-'
    },
    {
      title: 'Status',
      dataIndex: 'is_active',
      key: 'is_active',
      render: (isActive: boolean) => (
        <Tag color={isActive ? 'green' : 'red'}>
          {isActive ? 'Active' : 'Inactive'}
        </Tag>
      )
    },
    {
      title: 'Created At',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm')
    },
    {
      title: t('common.edit'),
      key: 'actions',
      render: (_: any, record: User) => (
        <Space>
          <Button type="text" icon={<EditOutlined />} onClick={() => handleEdit(record)} />
          <Popconfirm
            title="Delete User"
            description="Are you sure to delete this user?"
            onConfirm={() => handleDelete(record.id)}
          >
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      )
    }
  ]

  return (
    <Card
      title={t('admin.userManagement')}
      extra={
        <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
          Create User
        </Button>
      }
    >
      <Table
        columns={columns}
        dataSource={data}
        rowKey="id"
        loading={loading}
        pagination={pagination}
      />

      <Modal
        title={editingUser ? 'Edit User' : 'Create User'}
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
            name="username"
            label="Username"
            rules={[{ required: true, message: 'Please enter username' }]}
          >
            <Input />
          </Form.Item>

          <Form.Item
            name="email"
            label="Email"
            rules={[{ required: true, type: 'email', message: 'Please enter valid email' }]}
          >
            <Input />
          </Form.Item>

          {!editingUser && (
            <Form.Item
              name="password"
              label="Password"
              rules={[{ required: true, message: 'Please enter password' }]}
            >
              <Input.Password />
            </Form.Item>
          )}

          <Form.Item
            name="role"
            label="Role"
            rules={[{ required: true, message: 'Please select role' }]}
          >
            <Select
              options={[
                { label: 'Admin', value: 'admin' },
                { label: 'User', value: 'user' },
                { label: 'Viewer', value: 'viewer' }
              ]}
            />
          </Form.Item>

          <Form.Item
            name="quota_cpu"
            label="CPU Quota"
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="quota_memory"
            label="Memory Quota (MB)"
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="quota_disk"
            label="Disk Quota (GB)"
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="quota_vm_count"
            label="VM Count Quota"
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {t('common.submit')}
              </Button>
              <Button onClick={() => setIsModalOpen(false)}>
                {t('common.cancel')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}

export default Users
