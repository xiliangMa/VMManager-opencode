import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Table, Button, Card, Tag, Space, message, Popconfirm, Input, Row, Col, Statistic, Drawer } from 'antd'
import { EditOutlined, DeleteOutlined, UploadOutlined, SearchOutlined, FileOutlined } from '@ant-design/icons'
import { templatesApi, Template, VM } from '../../api/client'
import { useTable } from '../../hooks/useTable'
import dayjs from 'dayjs'

const Templates: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const { data, loading, pagination, refresh, search, setSearch } = useTable<Template>({
    api: templatesApi.list
  })

  const [drawerVisible, setDrawerVisible] = useState(false)
  const [selectedTemplate, setSelectedTemplate] = useState<Template | null>(null)
  const [vmList, setVMList] = useState<VM[]>([])
  const [vmLoading, setVMLoading] = useState(false)
  const [vmTotal, setVMTotal] = useState(0)
  const [vmPage, setVMPage] = useState(1)

  const handleShowVMs = async (template: Template) => {
    setSelectedTemplate(template)
    setDrawerVisible(true)
    setVMLoading(true)
    setVMPage(1)
    try {
      const response = await templatesApi.getVMs(template.id, { page: 1, page_size: 10 })
      setVMList(response.data?.items || response.items || [])
      setVMTotal(response.data?.total || response.total || 0)
    } catch (error: any) {
      message.error(t('common.error'))
    } finally {
      setVMLoading(false)
    }
  }

  const handleVMPageChange = async (page: number) => {
    if (!selectedTemplate) return
    setVMPage(page)
    setVMLoading(true)
    try {
      const response = await templatesApi.getVMs(selectedTemplate.id, { page, page_size: 10 })
      setVMList(response.data?.items || response.items || [])
      setVMTotal(response.data?.total || response.total || 0)
    } catch (error: any) {
      message.error(t('common.error'))
    } finally {
      setVMLoading(false)
    }
  }

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
      key: 'name',
      render: (text: string, record: Template) => (
        <a onClick={() => handleShowVMs(record)}>{text}</a>
      )
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

      <Drawer
        title={`${t('template.vmList')} - ${selectedTemplate?.name || ''}`}
        placement="right"
        width={720}
        onClose={() => setDrawerVisible(false)}
        open={drawerVisible}
      >
        <Table
          columns={[
            {
              title: t('vm.name'),
              dataIndex: 'name',
              key: 'name',
              render: (text: string, record: VM) => (
                <a onClick={() => {
                  setDrawerVisible(false)
                  navigate(`/vms/${record.id}`)
                }}>{text}</a>
              )
            },
            {
              title: t('vm.status'),
              dataIndex: 'status',
              key: 'status',
              render: (status: string) => {
                const colors: Record<string, string> = {
                  running: 'green',
                  stopped: 'red',
                  suspended: 'orange',
                  pending: 'blue',
                  error: 'error'
                }
                return <Tag color={colors[status] || 'default'}>{status}</Tag>
              }
            },
            {
              title: t('table.vcpu'),
              dataIndex: 'cpuAllocated',
              key: 'cpu',
              render: (cpu: number) => `${cpu} vCPU`
            },
            {
              title: t('vm.memory'),
              dataIndex: 'memoryAllocated',
              key: 'memory',
              render: (memory: number) => `${memory} MB`
            },
            {
              title: t('common.createdAt'),
              dataIndex: 'createdAt',
              key: 'createdAt',
              render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm:ss')
            }
          ]}
          dataSource={vmList}
          rowKey="id"
          loading={vmLoading}
          pagination={{
            current: vmPage,
            pageSize: 10,
            total: vmTotal,
            showTotal: (total) => `${t('common.total')} ${total} ${t('vm.items')}`
          }}
          onChange={(p) => handleVMPageChange(p.current || 1)}
        />
      </Drawer>
    </div>
  )
}

export default Templates
