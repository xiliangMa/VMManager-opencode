import React, { useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Form, Input, Select, Button, Card, message, Space, Row, Col, Progress } from 'antd'
import { ArrowLeftOutlined, UploadOutlined, CloudUploadOutlined, CheckCircleOutlined } from '@ant-design/icons'
import { isosApi } from '../../api/client'

const CHUNK_SIZE = 100 * 1024 * 1024

const ISOUpload: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [form] = Form.useForm()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [uploadStep, setUploadStep] = useState(0)
  const [loading, setLoading] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState(0)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [uploadId, setUploadId] = useState<string | null>(null)

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

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

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
    const values = form.getFieldsValue()
    
    if (!values.name) {
      message.error(t('validation.pleaseEnterName'))
      return
    }
    
    if (!selectedFile) {
      message.error(t('iso.pleaseSelectFile'))
      return
    }

    setLoading(true)
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
      setLoading(false)
    }
  }

  const handleFileUpload = async () => {
    if (!selectedFile || !uploadId) return

    const values = form.getFieldsValue()

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
    } catch (error: any) {
      message.error(error.response?.data?.message || t('iso.failedToUpload'))
    } finally {
      setUploading(false)
    }
  }

  return (
    <div>
      <Card>
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col>
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/isos')}>
              {t('common.back')}
            </Button>
          </Col>
        </Row>

        {uploadStep === 0 && (
          <>
            <Form form={form} layout="vertical" initialValues={{ architecture: 'x86_64' }}>
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
                <Button onClick={() => navigate('/isos')}>{t('common.cancel')}</Button>
                <Button type="primary" loading={loading} onClick={handleStartUpload}>
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
            <Button type="primary" onClick={() => navigate('/isos')}>
              {t('common.close')}
            </Button>
          </div>
        )}
      </Card>
    </div>
  )
}

export default ISOUpload
