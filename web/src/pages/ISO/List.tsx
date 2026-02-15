import React, { useState, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Table, Card, Button, Space, Tag, Modal, Form, Input, Select, message, Popconfirm, Progress, Row, Col, Statistic } from 'antd'
import { DeleteOutlined, UploadOutlined, SearchOutlined, FileOutlined, CloudUploadOutlined, CheckCircleOutlined } from '@ant-design/icons'
import { isosApi, ISO } from '../../api/client'
import { useTable } from '../../hooks/useTable'
import { useAuthStore } from '../../stores/authStore'
import dayjs from 'dayjs'

const CHUNK_SIZE = 100 * 1024 * 1024

const ISOList: React.FC = () => {
  const { t } = useTranslation()
  const { user } = useAuthStore()
  const [isUploadModalOpen, setIsUploadModalOpen] = useState(false)
  const [uploadStep, setUploadStep] = useState(0)
  const [uploadForm] = Form.useForm()
  const [uploading, setUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState(0)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [uploadId, setUploadId] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [searchKeyword, setSearchKeyword] = useState('')
  const [architectureFilter, setArchitectureFilter] = useState('')

  const { data, loading, pagination, refresh } = useTable<ISO>({
    api: (params) => isosApi.list({ 
      ...params, 
      search: searchKeyword || undefined,
      architecture: architectureFilter || undefined
    })
  })

  const osTypeOptions = [
    { label: 'Ubuntu', value: 'Ubuntu' },
    { label: 'CentOS', value: 'CentOS' },
    { label: 'Debian', value: 'Debian' },
    { label: 'Windows', value: 'Windows' },
    { label: 'Rocky Linux', value: 'RockyLinux' },
    { label: 'AlmaLinux', value: 'AlmaLinux' },
    { label: 'Other', value: 'Other' }
  ]

  const archOptions = [
    { label: 'x86_64', value: 'x86_64' },
    { label: 'aarch64 (ARM 64-bit)', value: 'aarch64' }
  ]

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      const ext = file.name.toLowerCase().split('.').pop()
      if (ext !== 'iso') {
        message.error(t('iso.onlyIsoAllowed'))
        return
      }
      setSelectedFile(file)
      message.success(`${t('iso.fileSelected')}: ${file.name} (${formatSize(file.size)})`)
    }
  }

  const handleStartUpload = async () => {
    const values = uploadForm.getFieldsValue()
    
    if (!values.name) {
      message.error(t('validation.pleaseEnterName'))
      return
    }
    
    if (!selectedFile) {
      message.error(t('iso.pleaseSelectFile'))
      return
    }

    setUploading(true)
    try {
      const initResponse = await isosApi.initUpload({
        name: values.name,
        description: values.description || '',
        file_name: selectedFile.name,
        file_size: selectedFile.size,
        architecture: values.architecture || 'x86_64',
        os_type: values.os_type,
        os_version: values.os_version,
        chunk_size: CHUNK_SIZE
      })

      setUploadId(initResponse.data.upload_id)
      setUploadStep(1)
      message.success(t('iso.uploadInitialized'))
    } catch (error: any) {
      message.error(error.response?.data?.message || t('iso.failedToInitUpload'))
    } finally {
      setUploading(false)
    }
  }

  const handleFileUpload = async () => {
    if (!selectedFile || !uploadId) return

    const values = uploadForm.getFieldsValue()

    setUploading(true)
    setUploadProgress(0)

    try {
      const totalChunks = Math.ceil(selectedFile.size / CHUNK_SIZE)

      for (let i = 0; i < totalChunks; i++) {
        const start = i * CHUNK_SIZE
        const end = Math.min(start + CHUNK_SIZE, selectedFile.size)
        const chunk = selectedFile.slice(start, end)

        const formData = new FormData()
        formData.append('file', chunk)

        await isosApi.uploadPart(uploadId, i, totalChunks, formData)

        const progress = Math.round(((i + 1) / totalChunks) * 100)
        setUploadProgress(progress)
      }

      await isosApi.completeUpload(uploadId, {
        total_chunks: totalChunks,
        name: values.name,
        description: values.description,
        os_type: values.os_type,
        os_version: values.os_version
      })

      message.success(t('iso.uploadCompleted'))
      setUploadStep(2)
      refresh()
    } catch (error: any) {
      message.error(error.response?.data?.message || t('iso.failedToUpload'))
    } finally {
      setUploading(false)
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await isosApi.delete(id)
      message.success(t('iso.deletedSuccessfully'))
      refresh()
    } catch (error) {
      message.error(t('iso.failedToDelete'))
    }
  }

  const handleUploadModalClose = () => {
    setIsUploadModalOpen(false)
    setUploadStep(0)
    setSelectedFile(null)
    setUploadId(null)
    setUploadProgress(0)
    uploadForm.resetFields()
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
              onClick={() => setIsUploadModalOpen(true)}
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

      <Modal
        title={t('iso.uploadISO')}
        open={isUploadModalOpen}
        onCancel={handleUploadModalClose}
        footer={null}
        width={600}
      >
        {uploadStep === 0 && (
          <>
            <Form form={uploadForm} layout="vertical">
              <Form.Item
                name="name"
                label={t('iso.name')}
                rules={[{ required: true, message: t('validation.pleaseEnterName') }]}
              >
                <Input placeholder={t('iso.namePlaceholder')} />
              </Form.Item>

              <Form.Item name="description" label={t('common.description')}>
                <Input.TextArea rows={2} placeholder={t('iso.descriptionPlaceholder')} />
              </Form.Item>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item name="os_type" label={t('iso.osType')}>
                    <Select placeholder={t('iso.selectOSType')} options={osTypeOptions} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="os_version" label={t('iso.osVersion')}>
                    <Input placeholder="e.g., 22.04, 9" />
                  </Form.Item>
                </Col>
              </Row>

              <Form.Item name="architecture" label={t('iso.architecture')} initialValue="x86_64">
                <Select options={archOptions} />
              </Form.Item>

              <Form.Item label={t('iso.file')}>
                <input
                  type="file"
                  ref={fileInputRef}
                  style={{ display: 'none' }}
                  accept=".iso"
                  onChange={handleFileSelect}
                />
                <Button icon={<UploadOutlined />} onClick={() => fileInputRef.current?.click()}>
                  {t('iso.selectFile')}
                </Button>
                {selectedFile && (
                  <span style={{ marginLeft: 8 }}>
                    {selectedFile.name} ({formatSize(selectedFile.size)})
                  </span>
                )}
              </Form.Item>
            </Form>

            <div style={{ textAlign: 'right', marginTop: 16 }}>
              <Space>
                <Button onClick={handleUploadModalClose}>{t('common.cancel')}</Button>
                <Button type="primary" loading={uploading} onClick={handleStartUpload}>
                  {t('common.next')}
                </Button>
              </Space>
            </div>
          </>
        )}

        {uploadStep === 1 && (
          <div style={{ textAlign: 'center', padding: '24px 0' }}>
            <CloudUploadOutlined style={{ fontSize: 48, color: '#1890ff', marginBottom: 16 }} />
            <div style={{ marginBottom: 16 }}>
              {t('iso.uploading')}...
            </div>
            <Progress percent={uploadProgress} status="active" />
            <div style={{ marginTop: 16, color: '#888' }}>
              {selectedFile?.name}
            </div>
            <div style={{ marginTop: 24 }}>
              <Button type="primary" loading={uploading} onClick={handleFileUpload}>
                {t('iso.startUpload')}
              </Button>
            </div>
          </div>
        )}

        {uploadStep === 2 && (
          <div style={{ textAlign: 'center', padding: '24px 0' }}>
            <CheckCircleOutlined style={{ fontSize: 48, color: '#52c41a', marginBottom: 16 }} />
            <div style={{ marginBottom: 16, fontSize: 18 }}>
              {t('iso.uploadSuccess')}
            </div>
            <Button type="primary" onClick={handleUploadModalClose}>
              {t('common.close')}
            </Button>
          </div>
        )}
      </Modal>
    </div>
  )
}

export default ISOList
