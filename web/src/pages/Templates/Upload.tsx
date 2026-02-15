import React, { useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Form, Input, Select, InputNumber, Button, Card, message, Space, Row, Col, Progress } from 'antd'
import { ArrowLeftOutlined, UploadOutlined, CloudUploadOutlined, CheckCircleOutlined } from '@ant-design/icons'
import { templatesApi } from '../../api/client'

const CHUNK_SIZE = 100 * 1024 * 1024

const TemplateUpload: React.FC = () => {
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
  const [selectedArch, setSelectedArch] = useState<string>('x86_64')
  const [selectedFormat, setSelectedFormat] = useState<string>('qcow2')

  const osOptions = [
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

  const formatOptions = [
    { label: 'qcow2', value: 'qcow2' },
    { label: 'vmdk', value: 'vmdk' },
    { label: 'raw', value: 'raw' },
    { label: 'ova', value: 'ova' }
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
      setSelectedFile(file)
      message.success(`${t('template.fileSelected')}: ${file.name} (${formatSize(file.size)})`)
    }
  }

  const handleStartUpload = async () => {
    const values = form.getFieldsValue()
    
    if (!values.name) {
      message.error(t('validation.pleaseEnterName'))
      return
    }
    
    if (!selectedFile) {
      message.error(t('validation.pleaseSelectTemplate'))
      return
    }

    setLoading(true)
    try {
      const initResponse = await templatesApi.initUpload({
        name: values.name,
        description: values.description || '',
        file_name: selectedFile.name,
        file_size: selectedFile.size,
        format: selectedFormat,
        architecture: selectedArch,
        chunk_size: CHUNK_SIZE
      })

      setUploadId(initResponse.data.upload_id)
      setUploadStep(1)
      message.success(t('template.uploadInitialized'))
    } catch (error: any) {
      message.error(error.response?.data?.message || t('template.failedToInitUpload'))
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
        formData.append('file', chunk, selectedFile.name)

        await templatesApi.uploadPart(uploadId, i, totalChunks, formData)
        
        const progress = Math.round(((i + 1) / totalChunks) * 100)
        setUploadProgress(progress)
      }

      await templatesApi.completeUpload(uploadId, {
        total_chunks: totalChunks,
        name: values.name,
        description: values.description || '',
        os_type: values.os_type || 'Linux',
        os_version: values.os_version || '',
        architecture: selectedArch,
        format: selectedFormat,
        cpu_min: values.cpu_min || 1,
        cpu_max: values.cpu_max || 4,
        memory_min: values.memory_min || 1024,
        memory_max: values.memory_max || 8192,
        disk_min: values.disk_min || 20,
        disk_max: values.disk_max || 500,
        is_public: values.is_public !== false
      })

      message.success(t('template.uploadCompleted'))
      setUploadStep(2)
    } catch (error: any) {
      message.error(error.response?.data?.message || t('template.failedToUpload'))
    } finally {
      setUploading(false)
    }
  }

  return (
    <div>
      <Card>
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col>
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/templates')}>
              {t('common.back')}
            </Button>
          </Col>
        </Row>

        {uploadStep === 0 && (
          <>
            <Form form={form} layout="vertical" initialValues={{
              architecture: 'x86_64',
              format: 'qcow2',
              cpu_min: 1,
              cpu_max: 4,
              memory_min: 1024,
              memory_max: 8192,
              disk_min: 20,
              disk_max: 500,
              is_public: true
            }}>
              <Form.Item
                name="name"
                label={t('template.name')}
                rules={[{ required: true, message: t('validation.pleaseEnterName') }]}
              >
                <Input placeholder={t('template.namePlaceholder')} />
              </Form.Item>

              <Form.Item name="description" label={t('common.description')}>
                <Input.TextArea rows={2} placeholder={t('template.descriptionPlaceholder')} />
              </Form.Item>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item name="os_type" label={t('template.osType')}>
                    <Select placeholder={t('template.selectOSType')} options={osOptions} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="os_version" label={t('template.osVersion')}>
                    <Input placeholder="e.g., 22.04, 9" />
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item label={t('template.architecture')}>
                    <Select 
                      options={archOptions}
                      value={selectedArch}
                      onChange={(value) => setSelectedArch(value)}
                    />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item label={t('template.format')}>
                    <Select 
                      options={formatOptions}
                      value={selectedFormat}
                      onChange={(value) => setSelectedFormat(value)}
                    />
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item name="cpu_min" label={t('template.minCPU')}>
                    <InputNumber min={1} max={256} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="cpu_max" label={t('template.maxCPU')}>
                    <InputNumber min={1} max={256} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item name="memory_min" label={t('template.minMemory')}>
                    <InputNumber min={512} max={524288} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="memory_max" label={t('template.maxMemory')}>
                    <InputNumber min={512} max={524288} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
              </Row>

              <Row gutter={16}>
                <Col span={12}>
                  <Form.Item name="disk_min" label={t('template.minDisk')}>
                    <InputNumber min={10} max={10000} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
                <Col span={12}>
                  <Form.Item name="disk_max" label={t('template.maxDisk')}>
                    <InputNumber min={10} max={10000} style={{ width: '100%' }} />
                  </Form.Item>
                </Col>
              </Row>

              <Form.Item
                name="is_public"
                label={t('template.public')}
                valuePropName="checked"
              >
                <Select
                  options={[
                    { label: t('template.public'), value: true },
                    { label: t('template.private'), value: false }
                  ]}
                />
              </Form.Item>

              <Form.Item label={t('template.file')}>
                <input
                  type="file"
                  ref={fileInputRef}
                  style={{ display: 'none' }}
                  onChange={handleFileSelect}
                />
                <Button icon={<UploadOutlined />} onClick={() => fileInputRef.current?.click()}>
                  {t('template.selectFile')}
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
                <Button onClick={() => navigate('/templates')}>{t('common.cancel')}</Button>
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
              {t('template.uploading')}...
            </div>
            <Progress percent={uploadProgress} status="active" />
            <div style={{ marginTop: 16, color: '#888' }}>
              {selectedFile?.name}
            </div>
            <div style={{ marginTop: 24 }}>
              <Button type="primary" loading={uploading} onClick={handleFileUpload}>
                {t('template.startUpload')}
              </Button>
            </div>
          </div>
        )}

        {uploadStep === 2 && (
          <div style={{ textAlign: 'center', padding: '24px 0' }}>
            <CheckCircleOutlined style={{ fontSize: 48, color: '#52c41a', marginBottom: 16 }} />
            <div style={{ marginBottom: 16, fontSize: 18 }}>
              {t('template.uploadSuccess')}
            </div>
            <Button type="primary" onClick={() => navigate('/templates')}>
              {t('common.close')}
            </Button>
          </div>
        )}
      </Card>
    </div>
  )
}

export default TemplateUpload
