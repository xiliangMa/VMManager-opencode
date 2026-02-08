import React, { useState, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Form, Input, Select, InputNumber, Button, Card, Steps, message, Space, Row, Col, Progress } from 'antd'
import { ArrowLeftOutlined, UploadOutlined, CheckOutlined } from '@ant-design/icons'
import { templatesApi } from '../../api/client'

const CHUNK_SIZE = 100 * 1024 * 1024 // 100MB per chunk

const TemplateUpload: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [form] = Form.useForm()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [currentStep, setCurrentStep] = useState(0)
  const [loading, setLoading] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState(0)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [uploadId, setUploadId] = useState<string | null>(null)

  const osOptions = [
    { label: 'Ubuntu', value: 'Ubuntu' },
    { label: 'CentOS', value: 'CentOS' },
    { label: 'Debian', value: 'Debian' },
    { label: 'Windows', value: 'Windows' },
    { label: 'Rocky Linux', value: 'RockyLinux' }
  ]

  const archOptions = [
    { label: 'x86_64', value: 'x86_64' },
    { label: 'aarch64', value: 'aarch64' },
    { label: 'arm64', value: 'arm64' }
  ]

  const formatOptions = [
    { label: 'qcow2', value: 'qcow2' },
    { label: 'vmdk', value: 'vmdk' },
    { label: 'raw', value: 'raw' },
    { label: 'ova', value: 'ova' }
  ]

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      setSelectedFile(file)
      message.success(`Selected: ${file.name} (${(file.size / 1024 / 1024).toFixed(2)} MB)`)
    }
  }

  const handleStep1Submit = async (values: any) => {
    if (!selectedFile) {
      message.error('Please select a template file first')
      return
    }

    setLoading(true)
    try {
      const templateData = {
        name: values.name,
        description: values.description,
        os_type: values.os_type,
        os_version: values.os_version,
        architecture: values.architecture,
        format: values.format,
        cpu_min: values.cpu_min,
        cpu_max: values.cpu_max,
        memory_min: values.memory_min,
        memory_max: values.memory_max,
        disk_min: values.disk_min,
        disk_max: values.disk_max,
        disk_size: values.disk_max,
        is_public: values.is_public
      }

      // Step 1: Create template basic info
      await templatesApi.create(templateData)
      
      // Step 2: Initialize file upload
      const initResponse = await templatesApi.initUpload({
        name: values.name,
        description: values.description,
        file_name: selectedFile.name,
        file_size: selectedFile.size,
        format: values.format,
        architecture: values.architecture,
        chunk_size: CHUNK_SIZE
      })

      setUploadId(initResponse.data.upload_id)
      setCurrentStep(1)
      message.success('Template info saved, ready to upload file')
    } catch (error: any) {
      message.error(error.response?.data?.message || 'Failed to initialize upload')
    } finally {
      setLoading(false)
    }
  }

  const handleFileUpload = async () => {
    if (!selectedFile || !uploadId) return

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

      // Complete upload
      await templatesApi.completeUpload(uploadId, {
        total_chunks: totalChunks
      })

      setCurrentStep(2)
      message.success('Template uploaded successfully!')
      
      setTimeout(() => {
        navigate('/templates')
      }, 2000)
    } catch (error: any) {
      message.error(error.response?.data?.message || 'Failed to upload file')
    } finally {
      setUploading(false)
    }
  }

  const steps = [
    { title: 'Basic Info', description: 'Template details' },
    { title: 'Upload File', description: 'Upload template' },
    { title: 'Complete', description: 'Finished' }
  ]

  return (
    <Card
      title={
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/templates')} />
          {t('template.upload')}
        </Space>
      }
    >
      <Steps current={currentStep} items={steps} style={{ marginBottom: 32 }} />

      {currentStep === 0 && (
        <Form
          form={form}
          layout="vertical"
          onFinish={handleStep1Submit}
          initialValues={{
            architecture: 'x86_64',
            format: 'qcow2',
            cpu_min: 1,
            cpu_max: 4,
            memory_min: 1024,
            memory_max: 8192,
            disk_min: 20,
            disk_max: 500,
            is_public: true
          }}
        >
          <Form.Item
            name="name"
            label={t('template.name')}
            rules={[{ required: true, message: 'Please enter template name' }]}
          >
            <Input placeholder="Enter template name" />
          </Form.Item>

          <Form.Item name="description" label="Description">
            <Input.TextArea rows={3} placeholder="Optional description" />
          </Form.Item>

          <Form.Item
            name="os_type"
            label={t('template.osType')}
            rules={[{ required: true, message: 'Please select OS type' }]}
          >
            <Select placeholder="Select OS" options={osOptions} />
          </Form.Item>

          <Form.Item name="os_version" label="OS Version">
            <Input placeholder="e.g., 22.04 LTS" />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="architecture" label={t('template.architecture')}>
                <Select options={archOptions} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="format" label={t('template.format')}>
                <Select options={formatOptions} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="cpu_min" label="CPU Min">
                <InputNumber min={1} max={256} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="cpu_max" label="CPU Max">
                <InputNumber min={1} max={256} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="memory_min" label="Memory Min (MB)">
                <InputNumber min={512} max={524288} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="memory_max" label="Memory Max (MB)">
                <InputNumber min={512} max={524288} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="disk_min" label="Disk Min (GB)">
                <InputNumber min={10} max={10000} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="disk_max" label="Disk Max (GB)">
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

          <Form.Item label="Template File" required>
            <input
              type="file"
              ref={fileInputRef}
              style={{ display: 'none' }}
              onChange={handleFileSelect}
            />
            <Button
              type="dashed"
              icon={<UploadOutlined />}
              onClick={() => fileInputRef.current?.click()}
              style={{ width: '100%', height: 80 }}
            >
              {selectedFile 
                ? `${selectedFile.name} (${(selectedFile.size / 1024 / 1024).toFixed(2)} MB)`
                : 'Click to select template file (qcow2, vmdk, raw, ova, iso, etc.)'
              }
            </Button>
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={loading} icon={<CheckOutlined />}>
                Next: Upload File
              </Button>
              <Button onClick={() => navigate('/templates')}>
                {t('common.cancel')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      )}

      {currentStep === 1 && (
        <div style={{ textAlign: 'center', padding: '40px 0' }}>
          <h3>Uploading: {selectedFile?.name}</h3>
          <Progress percent={uploadProgress} style={{ margin: '20px auto', maxWidth: 400 }} />
          
          <p style={{ color: '#666' }}>
            {uploadProgress < 100 
              ? `Uploading chunk ${Math.ceil(uploadProgress * (selectedFile?.size || 0) / 100 / CHUNK_SIZE)}...`
              : 'Completing upload...'
            }
          </p>

          <Button 
            onClick={handleFileUpload} 
            loading={uploading}
            type="primary"
            size="large"
            icon={<UploadOutlined />}
          >
            {uploading ? 'Uploading...' : 'Start Upload'}
          </Button>
        </div>
      )}

      {currentStep === 2 && (
        <div style={{ textAlign: 'center', padding: '40px 0' }}>
          <CheckOutlined style={{ fontSize: 64, color: '#52c41a', marginBottom: 16 }} />
          <h2>Template Upload Complete!</h2>
          <p>Your template has been uploaded and is ready to use.</p>
          <p>Redirecting to templates list...</p>
        </div>
      )}
    </Card>
  )
}

export default TemplateUpload
