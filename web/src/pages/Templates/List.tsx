import React from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Table, Button, Card, Tag, Space, message, Popconfirm } from 'antd'
import { EditOutlined, DeleteOutlined, UploadOutlined } from '@ant-design/icons'
import { templatesApi, Template } from '../../api/client'
import { useTable } from '../../hooks/useTable'
import dayjs from 'dayjs'

const Templates: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const { data, loading, pagination, refresh } = useTable<Template>({
    api: templatesApi.list
  })

  const handleDelete = async (id: string) => {
    try {
      await templatesApi.delete(id)
      message.success(t('common.success'))
      refresh()
    } catch (error) {
      message.error(t('common.error'))
    }
  }

  const columns = [
    {
      title: t('template.name'),
      dataIndex: 'name',
      key: 'name',
      render: (text: string, record: Template) => (
        <a onClick={() => navigate(`/templates/${record.id}`)}>{text}</a>
      )
    },
    {
      title: t('template.osType'),
      dataIndex: 'os_type',
      key: 'os_type',
      render: (os: string) => os || '-'
    },
    {
      title: t('template.architecture'),
      dataIndex: 'architecture',
      key: 'architecture',
      render: (arch: string) => arch || 'x86_64'
    },
    {
      title: t('template.format'),
      dataIndex: 'format',
      key: 'format',
      render: (format: string) => <Tag>{format}</Tag>
    },
    {
      title: 'CPU',
      key: 'cpu',
      render: (_: any, record: Template) => `${record.cpu_min} - ${record.cpu_max}`
    },
    {
      title: t('vm.memory'),
      key: 'memory',
      render: (_: any, record: Template) => `${record.memory_min} - ${record.memory_max} MB`
    },
    {
      title: t('template.public'),
      dataIndex: 'is_public',
      key: 'is_public',
      render: (isPublic: boolean) => (
        <Tag color={isPublic ? 'green' : 'blue'}>
          {isPublic ? t('template.public') : t('template.private')}
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
      render: (_: any, record: Template) => (
        <Space>
          <Button type="text" icon={<EditOutlined />} />
          <Popconfirm
            title={t('common.delete')}
            description="Are you sure to delete this template?"
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
      title={t('template.upload')}
      extra={
        <Space>
          <Button icon={<UploadOutlined />} onClick={() => navigate('/templates/upload')}>
            {t('template.upload')}
          </Button>
        </Space>
      }
    >
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

export default Templates
