import React, { useState, useEffect, useCallback, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Card, Table, Button, Tag, Space, Modal, Form, Input, Select, InputNumber, Switch, message, Popconfirm, Row, Col, Statistic, Progress, Tabs, Tooltip } from 'antd'
import { PlusOutlined, DeleteOutlined, UndoOutlined, ClockCircleOutlined, ScheduleOutlined, SyncOutlined, CheckCircleOutlined, CloseCircleOutlined, LoadingOutlined } from '@ant-design/icons'
import { backupApi, VMBackup, BackupSchedule } from '../../api/client'
import dayjs from 'dayjs'

interface VMBackupsProps {
  vmId: string
}

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const VMBackups: React.FC<VMBackupsProps> = ({ vmId }) => {
  const { t } = useTranslation()
  const [backups, setBackups] = useState<VMBackup[]>([])
  const [schedules, setSchedules] = useState<BackupSchedule[]>([])
  const [loading, setLoading] = useState(false)
  const [scheduleLoading, setScheduleLoading] = useState(false)
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 })
  const [backupModalOpen, setBackupModalOpen] = useState(false)
  const [scheduleModalOpen, setScheduleModalOpen] = useState(false)
  const [editingSchedule, setEditingSchedule] = useState<BackupSchedule | null>(null)
  const [backupForm] = Form.useForm()
  const [scheduleForm] = Form.useForm()
  const progressIntervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const fetchBackups = useCallback(async (page = 1, pageSize = 10) => {
    setLoading(true)
    try {
      const response = await backupApi.listBackups(vmId, { page, page_size: pageSize })
      if (response.code === 0) {
        setBackups(response.data || [])
        setPagination({
          current: page,
          pageSize,
          total: response.meta?.total || 0
        })
      }
    } catch (error) {
      message.error(t('backup.failedToList'))
    } finally {
      setLoading(false)
    }
  }, [vmId, t])

  const fetchSchedules = useCallback(async () => {
    setScheduleLoading(true)
    try {
      const response = await backupApi.listSchedules(vmId)
      if (response.code === 0) {
        setSchedules(response.data || [])
      }
    } catch (error) {
      message.error(t('backup.failedToListSchedules'))
    } finally {
      setScheduleLoading(false)
    }
  }, [vmId, t])

  useEffect(() => {
    fetchBackups()
    fetchSchedules()
  }, [fetchBackups, fetchSchedules])

  useEffect(() => {
    const hasRunningBackup = backups.some(b => b.status === 'running')
    
    if (hasRunningBackup && !progressIntervalRef.current) {
      progressIntervalRef.current = setInterval(() => {
        fetchBackups(pagination.current, pagination.pageSize)
      }, 3000)
    } else if (!hasRunningBackup && progressIntervalRef.current) {
      clearInterval(progressIntervalRef.current)
      progressIntervalRef.current = null
    }

    return () => {
      if (progressIntervalRef.current) {
        clearInterval(progressIntervalRef.current)
        progressIntervalRef.current = null
      }
    }
  }, [backups, fetchBackups, pagination.current, pagination.pageSize])

  const handleCreateBackup = async (values: any) => {
    try {
      await backupApi.createBackup(vmId, values)
      message.success(t('backup.createBackupSuccess'))
      setBackupModalOpen(false)
      backupForm.resetFields()
      fetchBackups(pagination.current, pagination.pageSize)
    } catch (error) {
      message.error(t('backup.failedToCreate'))
    }
  }

  const handleDeleteBackup = async (backupId: string) => {
    try {
      await backupApi.deleteBackup(vmId, backupId)
      message.success(t('backup.deleteBackupSuccess'))
      fetchBackups(pagination.current, pagination.pageSize)
    } catch (error) {
      message.error(t('backup.failedToDelete'))
    }
  }

  const handleRestoreBackup = async (backupId: string) => {
    try {
      await backupApi.restoreBackup(vmId, backupId)
      message.success(t('backup.restoreBackupSuccess'))
    } catch (error) {
      message.error(t('backup.failedToRestore'))
    }
  }

  const handleCreateSchedule = async (values: any) => {
    try {
      if (editingSchedule) {
        await backupApi.updateSchedule(vmId, editingSchedule.id, values)
        message.success(t('backup.updateScheduleSuccess'))
      } else {
        await backupApi.createSchedule(vmId, values)
        message.success(t('backup.createScheduleSuccess'))
      }
      setScheduleModalOpen(false)
      scheduleForm.resetFields()
      setEditingSchedule(null)
      fetchSchedules()
    } catch (error) {
      message.error(editingSchedule ? t('backup.failedToUpdateSchedule') : t('backup.failedToCreateSchedule'))
    }
  }

  const handleDeleteSchedule = async (scheduleId: string) => {
    try {
      await backupApi.deleteSchedule(vmId, scheduleId)
      message.success(t('backup.deleteScheduleSuccess'))
      fetchSchedules()
    } catch (error) {
      message.error(t('backup.failedToDeleteSchedule'))
    }
  }

  const handleToggleSchedule = async (scheduleId: string) => {
    try {
      await backupApi.toggleSchedule(vmId, scheduleId)
      fetchSchedules()
    } catch (error) {
      message.error(t('backup.failedToUpdateSchedule'))
    }
  }

  const handleEditSchedule = (schedule: BackupSchedule) => {
    setEditingSchedule(schedule)
    scheduleForm.setFieldsValue(schedule)
    setScheduleModalOpen(true)
  }

  const handleTableChange = (paginationInfo: any) => {
    fetchBackups(paginationInfo.current, paginationInfo.pageSize)
  }

  const backupTypeOptions = [
    { label: t('backup.full'), value: 'full' },
    { label: t('backup.incremental'), value: 'incremental' }
  ]

  const backupColumns = [
    {
      title: t('backup.backupName'),
      dataIndex: 'name',
      key: 'name'
    },
    {
      title: t('backup.backupType'),
      dataIndex: 'backupType',
      key: 'backupType',
      render: (type: string) => {
        const option = backupTypeOptions.find(t => t.value === type)
        return <Tag>{option?.label || type}</Tag>
      }
    },
    {
      title: t('backup.status'),
      dataIndex: 'status',
      key: 'status',
      render: (status: string, record: VMBackup) => {
        const statusConfig: Record<string, { color: string; icon: React.ReactNode }> = {
          pending: { color: 'default', icon: <ClockCircleOutlined /> },
          running: { color: 'processing', icon: <LoadingOutlined spin /> },
          completed: { color: 'success', icon: <CheckCircleOutlined /> },
          failed: { color: 'error', icon: <CloseCircleOutlined /> }
        }
        const config = statusConfig[status] || { color: 'default', icon: null }
        
        return (
          <Space direction="vertical" size={0} style={{ width: '100%' }}>
            <Space>
              <Tag color={config.color} icon={config.icon}>
                {t(`backup.${status}`)}
              </Tag>
            </Space>
            {status === 'running' && (
              <Progress 
                percent={record.progress} 
                size="small" 
                status="active"
                style={{ width: 120 }}
              />
            )}
            {status === 'failed' && record.errorMsg && (
              <Tooltip title={record.errorMsg}>
                <span style={{ color: '#ff4d4f', fontSize: 12 }}>
                  {record.errorMsg.length > 30 ? `${record.errorMsg.substring(0, 30)}...` : record.errorMsg}
                </span>
              </Tooltip>
            )}
          </Space>
        )
      }
    },
    {
      title: t('backup.fileSize'),
      dataIndex: 'fileSize',
      key: 'fileSize',
      render: (size: number) => formatBytes(size)
    },
    {
      title: t('backup.startedAt'),
      dataIndex: 'startedAt',
      key: 'startedAt',
      render: (time: string) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-'
    },
    {
      title: t('backup.completedAt'),
      dataIndex: 'completedAt',
      key: 'completedAt',
      render: (time: string) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-'
    },
    {
      title: t('table.action'),
      key: 'actions',
      render: (_: any, record: VMBackup) => (
        <Space>
          {record.status === 'completed' && (
            <Popconfirm
              title={t('popconfirm.restoreBackup')}
              onConfirm={() => handleRestoreBackup(record.id)}
            >
              <Button type="text" icon={<UndoOutlined />} />
            </Popconfirm>
          )}
          {record.status !== 'running' && (
            <Popconfirm
              title={t('popconfirm.deleteBackup')}
              onConfirm={() => handleDeleteBackup(record.id)}
            >
              <Button type="text" danger icon={<DeleteOutlined />} />
            </Popconfirm>
          )}
        </Space>
      )
    }
  ]

  const scheduleColumns = [
    {
      title: t('backup.scheduleName'),
      dataIndex: 'name',
      key: 'name'
    },
    {
      title: t('backup.cronExpr'),
      dataIndex: 'cronExpr',
      key: 'cronExpr'
    },
    {
      title: t('backup.backupType'),
      dataIndex: 'backupType',
      key: 'backupType',
      render: (type: string) => {
        const option = backupTypeOptions.find(t => t.value === type)
        return <Tag>{option?.label || type}</Tag>
      }
    },
    {
      title: t('backup.retention'),
      dataIndex: 'retention',
      key: 'retention',
      render: (days: number) => `${days} ${t('common.days')}`
    },
    {
      title: t('backup.enabled'),
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled: boolean, record: BackupSchedule) => (
        <Switch
          checked={enabled}
          onChange={() => handleToggleSchedule(record.id)}
          checkedChildren={t('option.on')}
          unCheckedChildren={t('option.off')}
        />
      )
    },
    {
      title: t('backup.lastRunAt'),
      dataIndex: 'lastRunAt',
      key: 'lastRunAt',
      render: (time: string) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-'
    },
    {
      title: t('backup.nextRunAt'),
      dataIndex: 'nextRunAt',
      key: 'nextRunAt',
      render: (time: string) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-'
    },
    {
      title: t('table.action'),
      key: 'actions',
      render: (_: any, record: BackupSchedule) => (
        <Space>
          <Button type="text" icon={<ScheduleOutlined />} onClick={() => handleEditSchedule(record)} />
          <Popconfirm
            title={t('popconfirm.deleteSchedule')}
            onConfirm={() => handleDeleteSchedule(record.id)}
          >
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      )
    }
  ]

  const totalSize = backups.reduce((sum, b) => sum + b.fileSize, 0)

  return (
    <div>
      <Tabs
        defaultActiveKey="backups"
        items={[
          {
            key: 'backups',
            label: (
              <span>
                <ClockCircleOutlined />
                {t('backup.backups')}
              </span>
            ),
            children: (
              <Card>
                <Row gutter={16} style={{ marginBottom: 16 }}>
                  <Col span={8}>
                    <Statistic 
                      title={t('backup.totalBackups')} 
                      value={pagination.total} 
                    />
                  </Col>
                  <Col span={8}>
                    <Statistic 
                      title={t('backup.totalSize')} 
                      value={formatBytes(totalSize)} 
                    />
                  </Col>
                </Row>

                <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
                  <Button onClick={() => fetchBackups(pagination.current, pagination.pageSize)}>
                    <SyncOutlined /> {t('common.refresh')}
                  </Button>
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => setBackupModalOpen(true)}>
                    {t('backup.createBackup')}
                  </Button>
                </div>

                <Table
                  columns={backupColumns}
                  dataSource={backups}
                  rowKey="id"
                  loading={loading}
                  pagination={{
                    current: pagination.current,
                    pageSize: pagination.pageSize,
                    total: pagination.total,
                    showSizeChanger: true,
                    showTotal: (total) => `${t('common.total')} ${total} ${t('backup.backups')}`
                  }}
                  onChange={handleTableChange}
                />
              </Card>
            )
          },
          {
            key: 'schedules',
            label: (
              <span>
                <ScheduleOutlined />
                {t('backup.schedules')}
              </span>
            ),
            children: (
              <Card>
                <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
                  <Button onClick={fetchSchedules}>
                    <SyncOutlined /> {t('common.refresh')}
                  </Button>
                  <Button type="primary" icon={<PlusOutlined />} onClick={() => { setEditingSchedule(null); scheduleForm.resetFields(); setScheduleModalOpen(true); }}>
                    {t('backup.createSchedule')}
                  </Button>
                </div>

                <Table
                  columns={scheduleColumns}
                  dataSource={schedules}
                  rowKey="id"
                  loading={scheduleLoading}
                  pagination={false}
                />
              </Card>
            )
          }
        ]}
      />

      <Modal
        title={t('backup.createBackup')}
        open={backupModalOpen}
        onCancel={() => setBackupModalOpen(false)}
        onOk={() => backupForm.submit()}
      >
        <Form
          form={backupForm}
          layout="vertical"
          onFinish={handleCreateBackup}
        >
          <Form.Item
            name="name"
            label={t('backup.backupName')}
            rules={[{ required: true, message: t('backup.backupNameRequired') }]}
          >
            <Input placeholder={t('backup.backupNamePlaceholder')} />
          </Form.Item>

          <Form.Item
            name="backupType"
            label={t('backup.backupType')}
            initialValue="full"
          >
            <Select options={backupTypeOptions} />
          </Form.Item>

          <Form.Item
            name="description"
            label={t('common.description')}
          >
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingSchedule ? t('backup.editSchedule') : t('backup.createSchedule')}
        open={scheduleModalOpen}
        onCancel={() => { setScheduleModalOpen(false); setEditingSchedule(null); }}
        onOk={() => scheduleForm.submit()}
      >
        <Form
          form={scheduleForm}
          layout="vertical"
          onFinish={handleCreateSchedule}
        >
          <Form.Item
            name="name"
            label={t('backup.scheduleName')}
            rules={[{ required: true, message: t('backup.scheduleNameRequired') }]}
          >
            <Input placeholder={t('backup.scheduleNamePlaceholder')} />
          </Form.Item>

          <Form.Item
            name="cronExpr"
            label={t('backup.cronExpr')}
            rules={[{ required: true, message: t('backup.cronExprRequired') }]}
          >
            <Input placeholder={t('backup.cronExprPlaceholder')} />
          </Form.Item>

          <Form.Item
            name="backupType"
            label={t('backup.backupType')}
            initialValue="full"
          >
            <Select options={backupTypeOptions} />
          </Form.Item>

          <Form.Item
            name="retention"
            label={t('backup.retention')}
            initialValue={7}
          >
            <InputNumber min={1} max={365} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="enabled"
            label={t('backup.enabled')}
            valuePropName="checked"
            initialValue={true}
          >
            <Switch checkedChildren={t('option.on')} unCheckedChildren={t('option.off')} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default VMBackups
