import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Table, Card, Button, Space, Tag, Select, message, Popconfirm, Input, Row, Col, Statistic } from 'antd'
import { DeleteOutlined, UploadOutlined, SearchOutlined, FileOutlined } from '@ant-design/icons'
import { isosApi, ISO } from '../../api/client'
import { useTable } from '../../hooks/useTable'
import { useAuthStore } from '../../stores/authStore'
import dayjs from 'dayjs'

const ISOList: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { user } = useAuthStore()
  const [searchKeyword, setSearchKeyword] = useState('')
  const [architectureFilter, setArchitectureFilter] = useState('')

  const { data, loading, pagination, refresh } = useTable<ISO>({
    api: (params) => isosApi.list({ 
      ...params, 
      search: searchKeyword || undefined,
      architecture: architectureFilter || undefined
    })
  })

  const archOptions = [
    { label: 'x86_64', value: 'x86_64' },
    { label: 'aarch64 (ARM 64-bit)', value: 'aarch64' }
  ]

  const handleDelete = async (id: string) => {
    try {
      await isosApi.delete(id)
      message.success(t('iso.deletedSuccessfully'))
      refresh()
    } catch (error) {
      message.error(t('iso.failedToDelete'))
    }
  }

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'green'
      case 'uploading': return 'blue'
      case 'error': return 'red'
      default: return 'default'
    }
  }

  const getArchColor = (arch: string) => {
    switch (arch) {
      case 'x86_64': return 'blue'
      case 'aarch64': return 'green'
      default: return 'default'
    }
  }

  const columns = [
    {
      title: t('iso.name'),
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => (
        <Space>
          <FileOutlined />
          {text}
        </Space>
      )
    },
    {
      title: t('iso.fileName'),
      dataIndex: 'fileName',
      key: 'fileName',
      ellipsis: true
    },
    {
      title: t('iso.size'),
      dataIndex: 'fileSize',
      key: 'fileSize',
      width: 120,
      render: (size: number) => formatSize(size)
    },
    {
      title: t('iso.osType'),
      dataIndex: 'osType',
      key: 'osType',
      width: 120,
      render: (osType: string) => osType ? <Tag>{osType}</Tag> : '-'
    },
    {
      title: t('iso.architecture'),
      dataIndex: 'architecture',
      key: 'architecture',
      width: 100,
      render: (arch: string) => (
        <Tag color={getArchColor(arch)}>{arch}</Tag>
      )
    },
    {
      title: t('iso.status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={getStatusColor(status)}>{status}</Tag>
      )
    },
    {
      title: t('iso.md5'),
      dataIndex: 'md5',
      key: 'md5',
      width: 200,
      ellipsis: true,
      render: (md5: string) => md5 || '-'
    },
    {
      title: t('common.createdAt'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm')
    },
    {
      title: t('common.actions'),
      key: 'actions',
      width: 100,
      render: (_: any, record: ISO) => (
        <Space>
          {user?.role === 'admin' && (
            <Popconfirm
              title={t('iso.confirmDelete')}
              onConfirm={() => handleDelete(record.id)}
              okText={t('common.confirm')}
              cancelText={t('common.cancel')}
            >
              <Button type="link" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          )}
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
              title={t('iso.totalISOs')} 
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
              value={searchKeyword}
              onChange={(e) => setSearchKeyword(e.target.value)}
              onPressEnter={refresh}
              style={{ width: 200 }}
            />
            <Select
              placeholder={t('iso.architecture')}
              value={architectureFilter}
              onChange={(value) => {
                setArchitectureFilter(value)
                refresh()
              }}
              allowClear
              style={{ width: 150 }}
              options={archOptions}
            />
            <Button onClick={refresh}>{t('common.refresh')}</Button>
          </Space>
          {user?.role === 'admin' && (
            <Button
              type="primary"
              icon={<UploadOutlined />}
              onClick={() => navigate('/isos/upload')}
            >
              {t('iso.uploadISO')}
            </Button>
          )}
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
            showTotal: (total) => `${t('common.total')} ${total} ${t('iso.items')}`
          }}
          onChange={(p) => {
            pagination.onChange(p.current || 1, p.pageSize || 10)
          }}
        />
      </Card>
    </div>
  )
}

export default ISOList
