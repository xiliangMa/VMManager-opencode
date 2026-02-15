import React, { useState, useRef, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Form, Input, Select, Button, Card, message, Space, Row, Col, Progress, Modal, Statistic } from 'antd'
import { ArrowLeftOutlined, UploadOutlined, CloudUploadOutlined, CheckCircleOutlined, ExclamationCircleOutlined, PauseCircleOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons'
import { isosApi } from '../../api/client'

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
  totalChunks: number
  uploadedChunks: number[]
  progress: number
  timestamp: number
}

const ISOUpload: React.FC = () => {
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
    localStorage.setItem('iso_upload_state', JSON.stringify(state))
  }, [])

  const clearUploadState = useCallback(() => {
    localStorage.removeItem('iso_upload_state')
  }, [])

  useEffect(() => {
    const savedState = localStorage.getItem('iso_upload_state')
    if (savedState) {
      try {
        const state: UploadState = JSON.parse(savedState)
        const now = Date.now()
        if (now - state.timestamp < 24 * 60 * 60 * 1000) {
          setPendingUpload(state)
        } else {
          localStorage.removeItem('iso_upload_state')
        }
      } catch {
        localStorage.removeItem('iso_upload_state')
      }
    }
  }, [])

  const checkPendingUpload = async () => {
    if (!pendingUpload) return
    
    try {
      const response = await isosApi.getUploadStatus(pendingUpload.uploadId)
      if (response.data && response.data.status === 'uploading') {
        Modal.confirm({
          title: t('iso.resumeUpload'),
          icon: <ExclamationCircleOutlined />,
          content: t('iso.foundPendingUpload', { fileName: pendingUpload.fileName, progress: pendingUpload.progress }),
          okText: t('iso.continueUpload'),
          cancelText: t('iso.startNewUpload'),
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
    
    form.setFieldsValue({
      name: state.name,
      description: state.description,
      os_type: state.osType,
      os_version: state.osVersion,
      architecture: state.architecture
    })

    const fileInput = document.createElement('input')
    fileInput.type = 'file'
    fileInput.accept = '.iso'
    fileInput.onchange = async (e) => {
      const file = (e.target as HTMLInputElement).files?.[0]
      if (file && file.name === state.fileName && file.size === state.fileSize) {
        setSelectedFile(file)
        message.success(t('iso.fileReselected'))
      } else {
        message.error(t('iso.fileMismatch'))
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
    currentRetry: number = 0
  ): Promise<void> => {
    if (uploadAbortRef.current) {
      throw new Error('Upload paused')
    }

    const formData = new FormData()
    formData.append('file', chunk)

    try {
      await isosApi.uploadPart(uploadId!, chunkIndex, totalChunks, formData)
    } catch (error: any) {
      if (currentRetry < MAX_RETRY_COUNT) {
        setRetryCount(currentRetry + 1)
        message.warning(t('iso.retrying', { count: currentRetry + 1 }))
        await new Promise(resolve => setTimeout(resolve, RETRY_DELAY))
        return uploadChunkWithRetry(chunkIndex, totalChunks, chunk, currentRetry + 1)
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
        message.info(t('iso.allChunksUploaded'))
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

          await uploadChunkWithRetry(i, totalChunks, chunk)
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
            architecture: values.architecture || 'x86_64',
            totalChunks,
            uploadedChunks: newUploadedChunks,
            progress,
            timestamp: Date.now()
          })
        }
      }

      await isosApi.completeUpload(uploadId, {
        total_chunks: totalChunks,
        name: values.name,
        description: values.description,
        os_type: values.os_type,
        os_version: values.os_version
      })

      clearUploadState()
      message.success(t('iso.uploadCompleted'))
      setUploadStep(2)
    } catch (error: any) {
      if (error.message === 'Upload paused') {
        message.info(t('iso.uploadPaused'))
      } else {
        message.error(error.response?.data?.message || t('iso.failedToUpload'))
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
            <CloudUploadOutlined style={{ fontSize: 48, color: paused ? '#faad14' : '#1890ff', marginBottom: 16 }} />
            <div style={{ marginBottom: 16 }}>
              {paused ? t('iso.uploadPaused') : t('iso.uploading')}...
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
                    title={t('iso.uploadSpeed')} 
                    value={formatSpeed(uploadSpeed)} 
                    valueStyle={{ fontSize: 16 }}
                  />
                </Col>
                <Col>
                  <Statistic 
                    title={t('iso.uploaded')} 
                    value={`${formatSize(uploadedBytes)} / ${formatSize(selectedFile?.size || 0)}`} 
                    valueStyle={{ fontSize: 16 }}
                  />
                </Col>
                <Col>
                  <Statistic 
                    title={t('iso.remainingTime')} 
                    value={formatTime(remainingTime)} 
                    valueStyle={{ fontSize: 16 }}
                  />
                </Col>
              </Row>
            )}

            {retryCount > 0 && uploading && (
              <div style={{ marginTop: 16, color: '#faad14' }}>
                <ReloadOutlined spin style={{ marginRight: 8 }} />
                {t('iso.retrying', { count: retryCount })}
              </div>
            )}

            <div style={{ marginTop: 24 }}>
              <Space>
                {!uploading && !paused && (
                  <Button type="primary" onClick={handleFileUpload}>
                    {t('iso.startUpload')}
                  </Button>
                )}
                {uploading && (
                  <Button 
                    icon={<PauseCircleOutlined />} 
                    onClick={handlePauseUpload}
                    danger
                  >
                    {t('iso.pause')}
                  </Button>
                )}
                {paused && (
                  <>
                    <Button 
                      type="primary" 
                      icon={<PlayCircleOutlined />} 
                      onClick={handleResumeUpload}
                    >
                      {t('iso.continueUpload')}
                    </Button>
                    <Button onClick={() => navigate('/isos')}>
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
