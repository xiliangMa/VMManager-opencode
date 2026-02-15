import React from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Table, Button, Card, Tag, Space, message, Popconfirm, Input, Row, Col, Statistic } from 'antd'
import { EditOutlined, DeleteOutlined, UploadOutlined, SearchOutlined, FileOutlined } from '@ant-design/icons'
import { templatesApi, Template } from '../../api/client'
import { useTable } from '../../hooks/useTable'
import dayjs from 'dayjs'

const Templates: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const { data, loading, pagination, refresh, search, setSearch } = useTable<Template>({
    api: templatesApi.list
  })

  const handleDelete = async (id: string) => {
    try {
      await templatesApi.delete(id)
      message.success(t('common.success'))
      refresh()
    } catch (error: any) {
      const errorMsg = error?.response?.data?.message || error?.message || t('common.error')
      message.error(errorMsg)
    }
  }

  const columns = [
    {
      title: t('template.name'),
      dataIndex: 'name',
      key: 'name'
    },
    {
      title: t('template.osType'),
      dataIndex: 'osType',
      key: 'os_type',
      render: (os: string) => os || '-'
    },
    {
      title: t('template.architecture'),
      dataIndex: 'architecture',
      key: 'architecture',
      render: (arch: string) => arch?.toUpperCase() || '-'
    },
    {
      title: t('template.format'),
      dataIndex: 'format',
      key: 'format',
      render: (format: string) => <Tag>{format}</Tag>
    },
    {
      title: t('table.vcpu'),
      key: 'cpu',
      render: (_: any, record: Template) => `${record.cpuMin} - ${record.cpuMax}`
    },
    {
      title: t('vm.memory'),
      key: 'memory',
      render: (_: any, record: Template) => `${record.memoryMin} - ${record.memoryMax} MB`
    },
    {
      title: t('template.public'),
      dataIndex: 'isPublic',
      key: 'is_public',
      render: (isPublic: boolean) => (
        <Tag color={isPublic ? 'green' : 'blue'}>
          {isPublic ? t('template.public') : t('template.private')}
        </Tag>
      )
    },
    {
      title: t('detail.createdAt'),
      dataIndex: 'createdAt',
      key: 'created_at',
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm')
    },
    {
      title: t('common.edit'),
      key: 'actions',
      render: (_: any, record: Template) => (
        <Space>
          <Button type="text" icon={<EditOutlined />} onClick={() => navigate(`/templates/${record.id}/edit`)} />
          <Popconfirm
            title={t('common.delete')}
            description={t('popconfirm.deleteTemplate')}
            onConfirm={() => handleDelete(record.id)}
          >
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      )
    }
  ]

  return (
    <div>
      <Card>
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={6}>
            <Statistic 
              title={t('template.totalTemplates')} 
              value={pagination.total} 
              prefix={<FileOutlined />} 
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
          <Button type="primary" icon={<UploadOutlined />} onClick={() => navigate('/templates/upload')}>
            {t('template.upload')}
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
            showTotal: (total) => `${t('common.total')} ${total} ${t('template.items')}`
          }}
          onChange={(p) => {
            pagination.onChange(p.current || 1, p.pageSize || 10)
          }}
        />
      </Card>
    </div>
  )
}

export default Templates
