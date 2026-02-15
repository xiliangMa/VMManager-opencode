import React, { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Table, Card, Button, Space, Tag, message, Popconfirm, Modal, Form, Input, InputNumber, Select, Tooltip, Row, Col, Statistic } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, UserOutlined, SearchOutlined } from '@ant-design/icons'
import { usersApi, User } from '../../api/client'
import { useAuthStore } from '../../stores/authStore'
import { useTable } from '../../hooks/useTable'
import dayjs from 'dayjs'

const Users: React.FC = () => {
  const { t } = useTranslation()
  const { user: currentUser } = useAuthStore()
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingUser, setEditingUser] = useState<User | null>(null)
  const [form] = Form.useForm()
  const { data, loading, pagination, refresh, search, setSearch } = useTable<User>({
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
      title: t('auth.username'),
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
      title: t('form.userRole'),
      dataIndex: 'role',
      key: 'role',
      render: (role: string) => (
        <Tag color={roleColors[role] || 'default'}>
          {role === 'admin' ? t('admin.roleAdmin') : role === 'user' ? t('admin.roleUser') : t('admin.roleViewer')}
        </Tag>
      )
    },
    {
      title: t('admin.cpuQuota'),
      dataIndex: 'quota_cpu',
      key: 'quota_cpu',
      render: (quota: number) => quota || '-'
    },
    {
      title: t('admin.memoryQuota'),
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
      title: t('table.created'),
      dataIndex: 'created_at',
      key: 'created_at',
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm')
    },
    {
      title: t('common.edit'),
      key: 'actions',
      render: (_: any, record: User) => {
        const isCurrentUser = currentUser?.id === record.id
        const isAdminUser = record.role === 'admin'
        const canEdit = !isCurrentUser
        const canDelete = !isCurrentUser && !isAdminUser
        return (
          <Space>
            <Button type="text" icon={<EditOutlined />} onClick={() => handleEdit(record)} disabled={!canEdit} />
            {canDelete ? (
              <Popconfirm
                title={t('admin.deleteUser')}
                description={t('popconfirm.deleteUser')}
                onConfirm={() => handleDelete(record.id)}
              >
                <Button type="text" danger icon={<DeleteOutlined />} />
              </Popconfirm>
            ) : (
              <Tooltip title={isAdminUser ? t('tooltip.adminCannotDelete') : t('tooltip.cannotDeleteYourself')}>
                <Button type="text" danger icon={<DeleteOutlined />} disabled />
              </Tooltip>
            )}
          </Space>
        )
      }
    }
  ]

  return (
    <div>
      <Card>
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={6}>
            <Statistic 
              title={t('admin.totalUsers')} 
              value={pagination.total} 
              prefix={<UserOutlined />} 
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
            <Button onClick={refresh}>{t('common.refresh')}</Button>
          </Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
            {t('admin.createUser')}
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
            showTotal: (total) => `${t('common.total')} ${total} ${t('admin.userItems')}`
          }}
          onChange={(p) => {
            pagination.onChange(p.current || 1, p.pageSize || 10)
          }}
        />

        <Modal
          title={editingUser ? t('admin.editUser') : t('admin.createUser')}
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
            label={t('auth.username')}
            rules={[{ required: true, message: t('validation.pleaseEnterName') }]}
          >
            <Input />
          </Form.Item>

          <Form.Item
            name="email"
            label={t('auth.email')}
            rules={[{ required: true, type: 'email', message: t('validation.pleaseEnterValidEmail') }]}
          >
            <Input />
          </Form.Item>

          {!editingUser && (
            <Form.Item
              name="password"
              label={t('auth.password')}
              rules={[{ required: true, message: t('validation.pleaseEnterPassword') }]}
            >
              <Input.Password />
            </Form.Item>
          )}

          <Form.Item
            name="role"
            label={t('form.userRole')}
            rules={[{ required: true, message: t('validation.pleaseSelectRole') }]}
          >
            <Select
              options={[
                { label: t('admin.roleAdmin'), value: 'admin' },
                { label: t('admin.roleUser'), value: 'user' },
                { label: t('admin.roleViewer'), value: 'viewer' }
              ]}
            />
          </Form.Item>

          <Form.Item
            name="quota_cpu"
            label={t('admin.cpuQuota')}
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="quota_memory"
            label={t('admin.memoryQuota')}
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="quota_disk"
            label={t('admin.diskQuota')}
          >
            <InputNumber min={0} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="quota_vm_count"
            label={t('admin.vmCountQuota')}
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
    </div>
  )
}

export default Users
