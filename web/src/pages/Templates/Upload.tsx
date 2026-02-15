import React, { useState, useRef, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Form, Input, Select, InputNumber, Button, Card, message, Space, Row, Col, Progress, Modal, Statistic } from 'antd'
import { ArrowLeftOutlined, UploadOutlined, CloudUploadOutlined, CheckCircleOutlined, ExclamationCircleOutlined, PauseCircleOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons'
import { templatesApi } from '../../api/client'

const CHUNK_SIZE = 100 * 1024 * 1024
const MAX_RETRY_COUNT = 3
const RETRY_DELAY = 2000

interface UploadState {
  uploadId: string
  fileName: string
  fileSize: number
  name: string
  description: string
  osType: string
  osVersion: string
  architecture: string
  format: string
  totalChunks: number
  uploadedChunks: number[]
  progress: number
  timestamp: number
}

const TemplateUpload: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [form] = Form.useForm()
  const fileInputRef = useRef<HTMLInputElement>(null)
  const uploadAbortRef = useRef<boolean>(false)
  const [uploadStep, setUploadStep] = useState(0)
  const [loading, setLoading] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [paused, setPaused] = useState(false)
  const [uploadProgress, setUploadProgress] = useState(0)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [uploadId, setUploadId] = useState<string | null>(null)
  const [selectedArch, setSelectedArch] = useState<string>('x86_64')
  const [selectedFormat, setSelectedFormat] = useState<string>('qcow2')
  const [uploadedChunks, setUploadedChunks] = useState<number[]>([])
  const [pendingUpload, setPendingUpload] = useState<UploadState | null>(null)
  const [uploadSpeed, setUploadSpeed] = useState(0)
  const [uploadedBytes, setUploadedBytes] = useState(0)
  const [remainingTime, setRemainingTime] = useState(0)
  const [retryCount, setRetryCount] = useState(0)
  const uploadStartTimeRef = useRef<number>(0)
  const lastUploadedBytesRef = useRef<number>(0)
  const lastSpeedUpdateTimeRef = useRef<number>(0)

  const saveUploadState = useCallback((state: UploadState) => {
    localStorage.setItem('template_upload_state', JSON.stringify(state))
  }, [])

  const clearUploadState = useCallback(() => {
    localStorage.removeItem('template_upload_state')
  }, [])

  useEffect(() => {
    const savedState = localStorage.getItem('template_upload_state')
    if (savedState) {
      try {
        const state: UploadState = JSON.parse(savedState)
        const now = Date.now()
        if (now - state.timestamp < 24 * 60 * 60 * 1000) {
          setPendingUpload(state)
        } else {
          localStorage.removeItem('template_upload_state')
        }
      } catch {
        localStorage.removeItem('template_upload_state')
      }
    }
  }, [])

  const checkPendingUpload = async () => {
    if (!pendingUpload) return
    
    try {
      const response = await templatesApi.getUploadStatus(pendingUpload.uploadId)
      if (response.data && response.data.status === 'uploading') {
        Modal.confirm({
          title: t('template.resumeUpload'),
          icon: <ExclamationCircleOutlined />,
          content: t('template.foundPendingUpload', { fileName: pendingUpload.fileName, progress: pendingUpload.progress }),
          okText: t('template.continueUpload'),
          cancelText: t('template.startNewUpload'),
          onOk: () => resumeUpload(pendingUpload),
          onCancel: () => {
            clearUploadState()
            setPendingUpload(null)
          }
        })
      } else {
        clearUploadState()
        setPendingUpload(null)
      }
    } catch {
      clearUploadState()
      setPendingUpload(null)
    }
  }

  useEffect(() => {
    if (pendingUpload) {
      checkPendingUpload()
    }
  }, [pendingUpload])

  const resumeUpload = async (state: UploadState) => {
    setUploadId(state.uploadId)
    setUploadStep(1)
    setUploadProgress(state.progress)
    setUploadedChunks(state.uploadedChunks)
    setSelectedArch(state.architecture)
    setSelectedFormat(state.format)
    
    form.setFieldsValue({
      name: state.name,
      description: state.description,
      os_type: state.osType,
      os_version: state.osVersion
    })

    const fileInput = document.createElement('input')
    fileInput.type = 'file'
    fileInput.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (file && file.name === state.fileName && file.size === state.fileSize) {
        setSelectedFile(file)
        message.success(t('template.fileReselected'))
      } else {
        message.error(t('template.fileMismatch'))
        clearUploadState()
        setPendingUpload(null)
        setUploadStep(0)
      }
    }
    fileInput.click()
  }

  useEffect(() => {
    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      if (uploading) {
        e.preventDefault()
        e.returnValue = ''
      }
    }
    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [uploading])

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

  const formatSpeed = (bytesPerSecond: number) => {
    if (bytesPerSecond === 0) return '0 B/s'
    const k = 1024
    const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s']
    const i = Math.floor(Math.log(bytesPerSecond) / Math.log(k))
    return parseFloat((bytesPerSecond / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const formatTime = (seconds: number) => {
    if (seconds <= 0) return '--:--'
    if (seconds < 60) return `${Math.round(seconds)}s`
    if (seconds < 3600) {
      const mins = Math.floor(seconds / 60)
      const secs = Math.round(seconds % 60)
      return `${mins}m ${secs}s`
    }
    const hours = Math.floor(seconds / 3600)
    const mins = Math.floor((seconds % 3600) / 60)
    return `${hours}h ${mins}m`
  }

  const updateUploadSpeed = useCallback((currentUploadedBytes: number) => {
    const now = Date.now()
    const timeDiff = (now - lastSpeedUpdateTimeRef.current) / 1000
    
    if (timeDiff >= 0.5) {
      const bytesDiff = currentUploadedBytes - lastUploadedBytesRef.current
      const speed = bytesDiff / timeDiff
      setUploadSpeed(speed)
      
      if (speed > 0 && selectedFile) {
        const remainingBytes = selectedFile.size - currentUploadedBytes
        setRemainingTime(remainingBytes / speed)
      }
      
      lastUploadedBytesRef.current = currentUploadedBytes
      lastSpeedUpdateTimeRef.current = now
    }
  }, [selectedFile])

  const uploadChunkWithRetry = async (
    chunkIndex: number,
    totalChunks: number,
    chunk: Blob,
    fileName: string,
    currentRetry: number = 0
  ): Promise<void> => {
    if (uploadAbortRef.current) {
      throw new Error('Upload paused')
    }

    const formData = new FormData()
    formData.append('file', chunk, fileName)

    try {
      await templatesApi.uploadPart(uploadId!, chunkIndex, totalChunks, formData)
    } catch (error: any) {
      if (currentRetry < MAX_RETRY_COUNT) {
        setRetryCount(currentRetry + 1)
        message.warning(t('template.retrying', { count: currentRetry + 1 }))
        await new Promise(resolve => setTimeout(resolve, RETRY_DELAY))
        return uploadChunkWithRetry(chunkIndex, totalChunks, chunk, fileName, currentRetry + 1)
      }
      throw error
    }
  }

  const handleFileUpload = async () => {
    if (!selectedFile || !uploadId) return

    const values = form.getFieldsValue()

    setUploading(true)
    setPaused(false)
    uploadAbortRef.current = false
    
    if (uploadedChunks.length === 0) {
      setUploadProgress(0)
      setUploadedBytes(0)
      uploadStartTimeRef.current = Date.now()
      lastSpeedUpdateTimeRef.current = Date.now()
      lastUploadedBytesRef.current = 0
    }

    try {
      const totalChunks = Math.ceil(selectedFile.size / CHUNK_SIZE)
      const chunksToUpload: number[] = []
      
      for (let i = 0; i < totalChunks; i++) {
        if (!uploadedChunks.includes(i)) {
          chunksToUpload.push(i)
        }
      }

      if (chunksToUpload.length === 0) {
        message.info(t('template.allChunksUploaded'))
      } else {
        let currentUploadedBytes = uploadedChunks.length * CHUNK_SIZE
        if (currentUploadedBytes > selectedFile.size) {
          currentUploadedBytes = selectedFile.size
        }
        setUploadedBytes(currentUploadedBytes)

        for (let idx = 0; idx < chunksToUpload.length; idx++) {
          if (uploadAbortRef.current) {
            setPaused(true)
            setUploading(false)
            return
          }

          const i = chunksToUpload[idx]
          const start = i * CHUNK_SIZE
          const end = Math.min(start + CHUNK_SIZE, selectedFile.size)
          const chunk = selectedFile.slice(start, end)

          await uploadChunkWithRetry(i, totalChunks, chunk, selectedFile.name)
          setRetryCount(0)

          const newUploadedChunks = [...uploadedChunks, i]
          setUploadedChunks(newUploadedChunks)
          
          currentUploadedBytes += chunk.size
          setUploadedBytes(currentUploadedBytes)
          
          const progress = Math.round((currentUploadedBytes / selectedFile.size) * 100)
          setUploadProgress(progress)

          updateUploadSpeed(currentUploadedBytes)

          saveUploadState({
            uploadId,
            fileName: selectedFile.name,
            fileSize: selectedFile.size,
            name: values.name,
            description: values.description || '',
            osType: values.os_type || '',
            osVersion: values.os_version || '',
            architecture: selectedArch,
            format: selectedFormat,
            totalChunks,
            uploadedChunks: newUploadedChunks,
            progress,
            timestamp: Date.now()
          })
        }
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

      clearUploadState()
      message.success(t('template.uploadCompleted'))
      setUploadStep(2)
    } catch (error: any) {
      if (error.message === 'Upload paused') {
        message.info(t('template.uploadPaused'))
      } else {
        message.error(error.response?.data?.message || t('template.failedToUpload'))
      }
    } finally {
      setUploading(false)
    }
  }

  const handlePauseUpload = () => {
    uploadAbortRef.current = true
  }

  const handleResumeUpload = () => {
    setPaused(false)
    handleFileUpload()
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
            <CloudUploadOutlined style={{ fontSize: 48, color: paused ? '#faad14' : '#1890ff', marginBottom: 16 }} />
            <div style={{ marginBottom: 16 }}>
              {paused ? t('template.uploadPaused') : t('template.uploading')}...
            </div>
            <Progress 
              percent={uploadProgress} 
              status={paused ? 'normal' : 'active'}
              format={(percent) => `${percent}%`}
            />
            <div style={{ marginTop: 16, color: '#888' }}>
              {selectedFile?.name}
            </div>
            
            {uploading && (
              <Row gutter={24} style={{ marginTop: 24, justifyContent: 'center' }}>
                <Col>
                  <Statistic 
                    title={t('template.uploadSpeed')} 
                    value={formatSpeed(uploadSpeed)} 
                    valueStyle={{ fontSize: 16 }}
                  />
                </Col>
                <Col>
                  <Statistic 
                    title={t('template.uploaded')} 
                    value={`${formatSize(uploadedBytes)} / ${formatSize(selectedFile?.size || 0)}`} 
                    valueStyle={{ fontSize: 16 }}
                  />
                </Col>
                <Col>
                  <Statistic 
                    title={t('template.remainingTime')} 
                    value={formatTime(remainingTime)} 
                    valueStyle={{ fontSize: 16 }}
                  />
                </Col>
              </Row>
            )}

            {retryCount > 0 && uploading && (
              <div style={{ marginTop: 16, color: '#faad14' }}>
                <ReloadOutlined spin style={{ marginRight: 8 }} />
                {t('template.retrying', { count: retryCount })}
              </div>
            )}

            <div style={{ marginTop: 24 }}>
              <Space>
                {!uploading && !paused && (
                  <Button type="primary" onClick={handleFileUpload}>
                    {t('template.startUpload')}
                  </Button>
                )}
                {uploading && (
                  <Button 
                    icon={<PauseCircleOutlined />} 
                    onClick={handlePauseUpload}
                    danger
                  >
                    {t('template.pause')}
                  </Button>
                )}
                {paused && (
                  <>
                    <Button 
                      type="primary" 
                      icon={<PlayCircleOutlined />} 
                      onClick={handleResumeUpload}
                    >
                      {t('template.continueUpload')}
                    </Button>
                    <Button onClick={() => navigate('/templates')}>
                      {t('common.cancel')}
                    </Button>
                  </>
                )}
              </Space>
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
