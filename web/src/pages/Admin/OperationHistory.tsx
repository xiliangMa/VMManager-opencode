import React, { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Table, Card, Tag, Space, Select, Button, Row, Col, Statistic, DatePicker, Drawer, Descriptions, Tabs, message, Tooltip } from 'antd'
import { LoginOutlined, SwapOutlined, PlayCircleOutlined, ClockCircleOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons'
import { operationHistoryApi, LoginHistory, ResourceChangeHistory, VMOperationHistory } from '../../api/client'
import dayjs from 'dayjs'

const { RangePicker } = DatePicker

const OperationHistory: React.FC = () => {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState('login')
  const [loading, setLoading] = useState(false)

  const [loginHistories, setLoginHistories] = useState<LoginHistory[]>([])
  const [loginPagination, setLoginPagination] = useState({ current: 1, pageSize: 20, total: 0 })
  const [loginFilters, setLoginFilters] = useState({ status: '', userId: '', dateRange: null as [dayjs.Dayjs | null, dayjs.Dayjs | null] | null })

  const [resourceChanges, setResourceChanges] = useState<ResourceChangeHistory[]>([])
  const [resourcePagination, setResourcePagination] = useState({ current: 1, pageSize: 20, total: 0 })
  const [resourceFilters, setResourceFilters] = useState({ resourceType: '', action: '' })

  const [vmOperations, setVMOperations] = useState<VMOperationHistory[]>([])
  const [vmOpPagination, setVMOpPagination] = useState({ current: 1, pageSize: 20, total: 0 })
  const [vmOpFilters, setVMOpFilters] = useState({ operation: '', status: '' })

  const [detailDrawer, setDetailDrawer] = useState<{ visible: boolean; type: string; data: any }>({
    visible: false,
    type: '',
    data: null
  })

  const fetchLoginHistories = async (page = 1, pageSize = 20) => {
    setLoading(true)
    try {
      const params: any = { page, page_size: pageSize }
      if (loginFilters.status) params.status = loginFilters.status
      if (loginFilters.userId) params.user_id = loginFilters.userId
      if (loginFilters.dateRange && loginFilters.dateRange[0] && loginFilters.dateRange[1]) {
        params.start_date = loginFilters.dateRange[0].format('YYYY-MM-DD')
        params.end_date = loginFilters.dateRange[1].format('YYYY-MM-DD')
      }
      const response = await operationHistoryApi.getLoginHistories(params)
      setLoginHistories(response.data?.list || [])
      setLoginPagination(prev => ({ ...prev, current: page, pageSize, total: response.data?.meta?.total || 0 }))
    } catch (_error) {
      message.error(t('common.error'))
    } finally {
      setLoading(false)
    }
  }

  const fetchResourceChanges = async (page = 1, pageSize = 20) => {
    setLoading(true)
    try {
      const params: any = { page, page_size: pageSize }
      if (resourceFilters.resourceType) params.resource_type = resourceFilters.resourceType
      if (resourceFilters.action) params.action = resourceFilters.action
      const response = await operationHistoryApi.getResourceChanges(params)
      setResourceChanges(response.data?.list || [])
      setResourcePagination(prev => ({ ...prev, current: page, pageSize, total: response.data?.meta?.total || 0 }))
    } catch (_error) {
      message.error(t('common.error'))
    } finally {
      setLoading(false)
    }
  }

  const fetchVMOperations = async (page = 1, pageSize = 20) => {
    setLoading(true)
    try {
      const params: any = { page, page_size: pageSize }
      if (vmOpFilters.operation) params.operation = vmOpFilters.operation
      if (vmOpFilters.status) params.status = vmOpFilters.status
      const response = await operationHistoryApi.getVMOperations(params)
      setVMOperations(response.data?.list || [])
      setVMOpPagination(prev => ({ ...prev, current: page, pageSize, total: response.data?.meta?.total || 0 }))
    } catch (_error) {
      message.error(t('common.error'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (activeTab === 'login') fetchLoginHistories()
    else if (activeTab === 'resource') fetchResourceChanges()
    else if (activeTab === 'vm') fetchVMOperations()
  }, [activeTab, loginFilters, resourceFilters, vmOpFilters])

  const loginColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 100, render: (id: string) => id.substring(0, 8) },
    { title: t('table.user'), dataIndex: 'username', key: 'username', width: 120 },
    { title: t('table.email'), dataIndex: 'email', key: 'email', width: 180 },
    {
      title: t('history.loginType'),
      dataIndex: 'loginType',
      key: 'loginType',
      width: 100,
      render: (type: string) => <Tag color="blue">{type}</Tag>
    },
    { title: t('table.ipAddress'), dataIndex: 'ipAddress', key: 'ipAddress', width: 140 },
    {
      title: t('table.status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={status === 'success' ? 'green' : 'red'}>
          {status === 'success' ? t('status.success') : t('status.failed')}
        </Tag>
      )
    },
    {
      title: t('table.created'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm:ss')
    },
    {
      title: t('table.action'),
      key: 'action',
      width: 80,
      render: (_: any, record: LoginHistory) => (
        <Button type="link" size="small" onClick={() => setDetailDrawer({ visible: true, type: 'login', data: record })}>
          {t('common.view')}
        </Button>
      )
    }
  ]

  const resourceColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 100, render: (id: string) => id.substring(0, 8) },
    {
      title: t('history.resourceType'),
      dataIndex: 'resourceType',
      key: 'resourceType',
      width: 120,
      render: (type: string) => {
        const colors: Record<string, string> = { vm: 'blue', template: 'green', iso: 'orange', network: 'purple', storage: 'cyan', user: 'magenta' }
        return <Tag color={colors[type] || 'default'}>{type.toUpperCase()}</Tag>
      }
    },
    { title: t('history.resourceName'), dataIndex: 'resourceName', key: 'resourceName', width: 150 },
    {
      title: t('table.action'),
      dataIndex: 'action',
      key: 'action',
      width: 100,
      render: (action: string) => {
        const colors: Record<string, string> = { create: 'green', update: 'blue', delete: 'red', start: 'cyan', stop: 'orange', restart: 'purple' }
        return <Tag color={colors[action] || 'default'}>{action}</Tag>
      }
    },
    { title: t('table.user'), dataIndex: 'username', key: 'username', width: 120 },
    { title: t('table.ipAddress'), dataIndex: 'ipAddress', key: 'ipAddress', width: 140 },
    {
      title: t('table.created'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm:ss')
    },
    {
      title: t('table.action'),
      key: 'action_btn',
      width: 80,
      render: (_: any, record: ResourceChangeHistory) => (
        <Button type="link" size="small" onClick={() => setDetailDrawer({ visible: true, type: 'resource', data: record })}>
          {t('common.view')}
        </Button>
      )
    }
  ]

  const vmOpColumns = [
    { title: 'ID', dataIndex: 'id', key: 'id', width: 100, render: (id: string) => id.substring(0, 8) },
    { title: t('history.vmName'), dataIndex: 'vmName', key: 'vmName', width: 150 },
    {
      title: t('history.operation'),
      dataIndex: 'operation',
      key: 'operation',
      width: 120,
      render: (op: string) => {
        const colors: Record<string, string> = { create: 'green', start: 'cyan', stop: 'orange', restart: 'purple', delete: 'red', suspend: 'gold', resume: 'lime', clone: 'geekblue', snapshot: 'blue', backup: 'magenta' }
        return <Tag color={colors[op] || 'default'}>{op}</Tag>
      }
    },
    {
      title: t('table.status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const colors: Record<string, string> = { pending: 'default', running: 'processing', success: 'green', failed: 'red' }
        const icons: Record<string, React.ReactNode> = { pending: <ClockCircleOutlined />, running: <PlayCircleOutlined />, success: <CheckCircleOutlined />, failed: <CloseCircleOutlined /> }
        return <Tag color={colors[status]} icon={icons[status]}>{status}</Tag>
      }
    },
    {
      title: t('history.duration'),
      dataIndex: 'duration',
      key: 'duration',
      width: 100,
      render: (d: number) => d ? `${d}ms` : '-'
    },
    { title: t('table.user'), dataIndex: 'username', key: 'username', width: 120 },
    {
      title: t('history.startedAt'),
      dataIndex: 'startedAt',
      key: 'startedAt',
      width: 180,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm:ss')
    },
    {
      title: t('table.action'),
      key: 'action_btn',
      width: 80,
      render: (_: any, record: VMOperationHistory) => (
        <Button type="link" size="small" onClick={() => setDetailDrawer({ visible: true, type: 'vm', data: record })}>
          {t('common.view')}
        </Button>
      )
    }
  ]

  const renderLoginFilters = () => (
    <Space wrap style={{ marginBottom: 16 }}>
      <Select
        placeholder={t('table.status')}
        allowClear
        style={{ width: 120 }}
        value={loginFilters.status || undefined}
        onChange={(value) => setLoginFilters(prev => ({ ...prev, status: value || '' }))}
        options={[
          { label: t('status.success'), value: 'success' },
          { label: t('status.failed'), value: 'failed' }
        ]}
      />
      <RangePicker
        value={loginFilters.dateRange}
        onChange={(dates) => setLoginFilters(prev => ({ ...prev, dateRange: dates }))}
        style={{ width: 260 }}
      />
      <Button onClick={() => setLoginFilters({ status: '', userId: '', dateRange: null })}>{t('common.reset')}</Button>
      <Button onClick={() => fetchLoginHistories(loginPagination.current, loginPagination.pageSize)}>{t('common.refresh')}</Button>
    </Space>
  )

  const renderResourceFilters = () => (
    <Space wrap style={{ marginBottom: 16 }}>
      <Select
        placeholder={t('history.resourceType')}
        allowClear
        style={{ width: 140 }}
        value={resourceFilters.resourceType || undefined}
        onChange={(value) => setResourceFilters(prev => ({ ...prev, resourceType: value || '' }))}
        options={[
          { label: 'VM', value: 'vm' },
          { label: 'Template', value: 'template' },
          { label: 'ISO', value: 'iso' },
          { label: 'Network', value: 'network' },
          { label: 'Storage', value: 'storage' },
          { label: 'User', value: 'user' }
        ]}
      />
      <Select
        placeholder={t('table.action')}
        allowClear
        style={{ width: 120 }}
        value={resourceFilters.action || undefined}
        onChange={(value) => setResourceFilters(prev => ({ ...prev, action: value || '' }))}
        options={[
          { label: t('action.create'), value: 'create' },
          { label: t('action.update'), value: 'update' },
          { label: t('action.delete'), value: 'delete' },
          { label: t('action.start'), value: 'start' },
          { label: t('action.stop'), value: 'stop' },
          { label: t('action.restart'), value: 'restart' }
        ]}
      />
      <Button onClick={() => setResourceFilters({ resourceType: '', action: '' })}>{t('common.reset')}</Button>
      <Button onClick={() => fetchResourceChanges(resourcePagination.current, resourcePagination.pageSize)}>{t('common.refresh')}</Button>
    </Space>
  )

  const renderVMOpFilters = () => (
    <Space wrap style={{ marginBottom: 16 }}>
      <Select
        placeholder={t('history.operation')}
        allowClear
        style={{ width: 140 }}
        value={vmOpFilters.operation || undefined}
        onChange={(value) => setVMOpFilters(prev => ({ ...prev, operation: value || '' }))}
        options={[
          { label: t('action.create'), value: 'create' },
          { label: t('action.start'), value: 'start' },
          { label: t('action.stop'), value: 'stop' },
          { label: t('action.restart'), value: 'restart' },
          { label: t('action.delete'), value: 'delete' },
          { label: t('action.suspend'), value: 'suspend' },
          { label: t('action.resume'), value: 'resume' },
          { label: t('action.clone'), value: 'clone' },
          { label: t('action.snapshot'), value: 'snapshot' },
          { label: t('action.backup'), value: 'backup' }
        ]}
      />
      <Select
        placeholder={t('table.status')}
        allowClear
        style={{ width: 120 }}
        value={vmOpFilters.status || undefined}
        onChange={(value) => setVMOpFilters(prev => ({ ...prev, status: value || '' }))}
        options={[
          { label: t('status.pending'), value: 'pending' },
          { label: t('status.running'), value: 'running' },
          { label: t('status.success'), value: 'success' },
          { label: t('status.failed'), value: 'failed' }
        ]}
      />
      <Button onClick={() => setVMOpFilters({ operation: '', status: '' })}>{t('common.reset')}</Button>
      <Button onClick={() => fetchVMOperations(vmOpPagination.current, vmOpPagination.pageSize)}>{t('common.refresh')}</Button>
    </Space>
  )

  const renderDetailDrawer = () => {
    if (!detailDrawer.data) return null

    if (detailDrawer.type === 'login') {
      const data = detailDrawer.data as LoginHistory
      return (
        <Descriptions column={1} bordered size="small">
          <Descriptions.Item label="ID">{data.id}</Descriptions.Item>
          <Descriptions.Item label={t('table.user')}>{data.username} ({data.email})</Descriptions.Item>
          <Descriptions.Item label={t('history.loginType')}><Tag color="blue">{data.loginType}</Tag></Descriptions.Item>
          <Descriptions.Item label={t('table.ipAddress')}>{data.ipAddress}</Descriptions.Item>
          <Descriptions.Item label={t('table.status')}>
            <Tag color={data.status === 'success' ? 'green' : 'red'}>{data.status}</Tag>
          </Descriptions.Item>
          {data.failureReason && <Descriptions.Item label={t('history.failureReason')}><span style={{ color: 'red' }}>{data.failureReason}</span></Descriptions.Item>}
          {data.location && <Descriptions.Item label={t('history.location')}>{data.location}</Descriptions.Item>}
          <Descriptions.Item label={t('table.created')}>{dayjs(data.createdAt).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
          {data.logoutAt && <Descriptions.Item label={t('history.logoutAt')}>{dayjs(data.logoutAt).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>}
          {data.sessionDuration > 0 && <Descriptions.Item label={t('history.sessionDuration')}>{Math.floor(data.sessionDuration / 60)}m {data.sessionDuration % 60}s</Descriptions.Item>}
          <Descriptions.Item label={t('history.userAgent')}>
            <Tooltip title={data.userAgent}>
              <span style={{ maxWidth: 300, display: 'inline-block', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {data.userAgent || '-'}
              </span>
            </Tooltip>
          </Descriptions.Item>
        </Descriptions>
      )
    }

    if (detailDrawer.type === 'resource') {
      const data = detailDrawer.data as ResourceChangeHistory
      return (
        <Descriptions column={1} bordered size="small">
          <Descriptions.Item label="ID">{data.id}</Descriptions.Item>
          <Descriptions.Item label={t('history.resourceType')}><Tag color="blue">{data.resourceType}</Tag></Descriptions.Item>
          <Descriptions.Item label={t('history.resourceId')}>{data.resourceId}</Descriptions.Item>
          <Descriptions.Item label={t('history.resourceName')}>{data.resourceName || '-'}</Descriptions.Item>
          <Descriptions.Item label={t('table.action')}><Tag color="green">{data.action}</Tag></Descriptions.Item>
          {data.oldValue && (
            <Descriptions.Item label={t('history.oldValue')}>
              <pre style={{ margin: 0, maxHeight: 150, overflow: 'auto' }}>{JSON.stringify(JSON.parse(data.oldValue), null, 2)}</pre>
            </Descriptions.Item>
          )}
          {data.newValue && (
            <Descriptions.Item label={t('history.newValue')}>
              <pre style={{ margin: 0, maxHeight: 150, overflow: 'auto' }}>{JSON.stringify(JSON.parse(data.newValue), null, 2)}</pre>
            </Descriptions.Item>
          )}
          <Descriptions.Item label={t('table.user')}>{data.username || '-'}</Descriptions.Item>
          <Descriptions.Item label={t('table.ipAddress')}>{data.ipAddress}</Descriptions.Item>
          <Descriptions.Item label={t('table.created')}>{dayjs(data.createdAt).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
        </Descriptions>
      )
    }

    if (detailDrawer.type === 'vm') {
      const data = detailDrawer.data as VMOperationHistory
      return (
        <Descriptions column={1} bordered size="small">
          <Descriptions.Item label="ID">{data.id}</Descriptions.Item>
          <Descriptions.Item label={t('history.vmName')}>{data.vmName || '-'}</Descriptions.Item>
          <Descriptions.Item label={t('history.operation')}><Tag color="blue">{data.operation}</Tag></Descriptions.Item>
          <Descriptions.Item label={t('table.status')}>
            <Tag color={data.status === 'success' ? 'green' : data.status === 'failed' ? 'red' : 'default'}>{data.status}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('history.startedAt')}>{dayjs(data.startedAt).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>
          {data.completedAt && <Descriptions.Item label={t('history.completedAt')}>{dayjs(data.completedAt).format('YYYY-MM-DD HH:mm:ss')}</Descriptions.Item>}
          {data.duration > 0 && <Descriptions.Item label={t('history.duration')}>{data.duration}ms</Descriptions.Item>}
          <Descriptions.Item label={t('table.user')}>{data.username || '-'}</Descriptions.Item>
          <Descriptions.Item label={t('table.ipAddress')}>{data.ipAddress}</Descriptions.Item>
          {data.requestParams && (
            <Descriptions.Item label={t('history.requestParams')}>
              <pre style={{ margin: 0, maxHeight: 150, overflow: 'auto' }}>{JSON.stringify(JSON.parse(data.requestParams), null, 2)}</pre>
            </Descriptions.Item>
          )}
          {data.responseData && (
            <Descriptions.Item label={t('history.responseData')}>
              <pre style={{ margin: 0, maxHeight: 150, overflow: 'auto' }}>{JSON.stringify(JSON.parse(data.responseData), null, 2)}</pre>
            </Descriptions.Item>
          )}
          {data.errorMessage && <Descriptions.Item label={t('history.errorMessage')}><span style={{ color: 'red' }}>{data.errorMessage}</span></Descriptions.Item>}
        </Descriptions>
      )
    }

    return null
  }

  const tabItems = [
    {
      key: 'login',
      label: <span><LoginOutlined /> {t('history.loginHistory')}</span>,
      children: (
        <>
          {renderLoginFilters()}
          <Table
            columns={loginColumns}
            dataSource={loginHistories}
            rowKey="id"
            loading={loading}
            pagination={{
              current: loginPagination.current,
              pageSize: loginPagination.pageSize,
              total: loginPagination.total,
              showSizeChanger: true,
              showTotal: (total) => `${t('common.total')} ${total} ${t('history.items')}`
            }}
            onChange={(p) => fetchLoginHistories(p.current || 1, p.pageSize || 20)}
          />
        </>
      )
    },
    {
      key: 'resource',
      label: <span><SwapOutlined /> {t('history.resourceChanges')}</span>,
      children: (
        <>
          {renderResourceFilters()}
          <Table
            columns={resourceColumns}
            dataSource={resourceChanges}
            rowKey="id"
            loading={loading}
            pagination={{
              current: resourcePagination.current,
              pageSize: resourcePagination.pageSize,
              total: resourcePagination.total,
              showSizeChanger: true,
              showTotal: (total) => `${t('common.total')} ${total} ${t('history.items')}`
            }}
            onChange={(p) => fetchResourceChanges(p.current || 1, p.pageSize || 20)}
          />
        </>
      )
    },
    {
      key: 'vm',
      label: <span><PlayCircleOutlined /> {t('history.vmOperations')}</span>,
      children: (
        <>
          {renderVMOpFilters()}
          <Table
            columns={vmOpColumns}
            dataSource={vmOperations}
            rowKey="id"
            loading={loading}
            pagination={{
              current: vmOpPagination.current,
              pageSize: vmOpPagination.pageSize,
              total: vmOpPagination.total,
              showSizeChanger: true,
              showTotal: (total) => `${t('common.total')} ${total} ${t('history.items')}`
            }}
            onChange={(p) => fetchVMOperations(p.current || 1, p.pageSize || 20)}
          />
        </>
      )
    }
  ]

  return (
    <div>
      <Card>
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={8}>
            <Statistic title={t('history.totalLoginHistories')} value={loginPagination.total} prefix={<LoginOutlined />} />
          </Col>
          <Col span={8}>
            <Statistic title={t('history.totalResourceChanges')} value={resourcePagination.total} prefix={<SwapOutlined />} />
          </Col>
          <Col span={8}>
            <Statistic title={t('history.totalVMOperations')} value={vmOpPagination.total} prefix={<PlayCircleOutlined />} />
          </Col>
        </Row>

        <Tabs activeKey={activeTab} onChange={setActiveTab} items={tabItems} />
      </Card>

      <Drawer
        title={t('history.detail')}
        placement="right"
        width={500}
        onClose={() => setDetailDrawer({ visible: false, type: '', data: null })}
        open={detailDrawer.visible}
      >
        {renderDetailDrawer()}
      </Drawer>
    </div>
  )
}

export default OperationHistory
