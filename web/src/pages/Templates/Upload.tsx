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
      // 直接初始化上传，模板信息在完成时一起保存
      const initResponse = await templatesApi.initUpload({
        name: values.name,
        description: values.description || '',
        file_name: selectedFile.name,
        file_size: selectedFile.size,
        format: values.format || 'qcow2',
        architecture: values.architecture || 'x86_64',
        chunk_size: CHUNK_SIZE
      })

      setUploadId(initResponse.data.upload_id)
      setCurrentStep(1)
      message.success(t('message.uploadInitialized'))
    } catch (error: any) {
      message.error(error.response?.data?.message || t('alert.failedToLoad'))
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

      // Complete upload with full template info
      await templatesApi.completeUpload(uploadId, {
        total_chunks: totalChunks,
        name: values.name,
        description: values.description || '',
        os_type: values.os_type || 'Linux',
        os_version: values.os_version || '',
        architecture: values.architecture || 'x86_64',
        format: values.format || 'qcow2',
        cpu_min: values.cpu_min || 1,
        cpu_max: values.cpu_max || 4,
        memory_min: values.memory_min || 1024,
        memory_max: values.memory_max || 8192,
        disk_min: values.disk_min || 20,
        disk_max: values.disk_max || 500,
        is_public: values.is_public !== false
      })

      setCurrentStep(2)
      message.success(t('alert.uploadComplete'))
      
      setTimeout(() => {
        navigate('/templates')
      }, 2000)
    } catch (error: any) {
      message.error(error.response?.data?.message || t('message.failedToCreate'))
    } finally {
      setUploading(false)
    }
  }

  const steps = [
    { title: t('step.basicInfo'), description: t('step.templateDetails') },
    { title: t('step.uploadFile'), description: t('step.uploadTemplate') },
    { title: t('step.complete'), description: t('step.finished') }
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
            rules={[{ required: true, message: t('validation.pleaseEnterName') }]}
          >
            <Input placeholder={t('placeholder.enterName')} />
          </Form.Item>

          <Form.Item name="description" label={t('template.description')}>
            <Input.TextArea rows={3} placeholder={t('placeholder.optionalDescription')} />
          </Form.Item>

          <Form.Item
            name="os_type"
            label={t('template.osType')}
            rules={[{ required: true, message: t('validation.pleaseSelectTemplate') }]}
          >
            <Select placeholder={t('placeholder.selectOs')} options={osOptions} />
          </Form.Item>

          <Form.Item name="os_version" label={t('form.osVersion')}>
            <Input placeholder={t('placeholder.enterVersion')} />
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

          <Form.Item label={t('form.templateFile')} required>
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
                : t('message.selectTemplateFile')
              }
            </Button>
          </Form.Item>

          <Form.Item>
            <Space>
              <Button 
                type="primary" 
                onClick={handleStartUpload} 
                loading={loading} 
                icon={<UploadOutlined />}
                disabled={!selectedFile}
              >
                {t('button.next')}
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
          <h3>{t('message.uploading')}: {selectedFile?.name}</h3>
          <Progress percent={uploadProgress} style={{ margin: '20px auto', maxWidth: 400 }} />
          
          <p style={{ color: '#666' }}>
            {uploadProgress < 100 
              ? `${t('message.uploadingChunk')} ${Math.ceil(uploadProgress * (selectedFile?.size || 0) / 100 / CHUNK_SIZE)}...`
              : t('message.completingUpload')
            }
          </p>

          <Button 
            onClick={handleFileUpload} 
            loading={uploading}
            type="primary"
            size="large"
            icon={<UploadOutlined />}
          >
            {uploading ? t('message.uploadingChunk') : t('message.startUpload')}
          </Button>
        </div>
      )}

      {currentStep === 2 && (
        <div style={{ textAlign: 'center', padding: '40px 0' }}>
          <CheckOutlined style={{ fontSize: 64, color: '#52c41a', marginBottom: 16 }} />
          <h2>{t('message.uploadComplete')}</h2>
          <p>{t('message.uploadRedirecting')}</p>
          <p>{t('message.uploadRedirecting')}</p>
        </div>
      )}
    </Card>
  )
}

export default TemplateUpload
