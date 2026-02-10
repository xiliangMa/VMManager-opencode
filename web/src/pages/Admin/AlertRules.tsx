import React, { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Card, Table, Button, Tag, Space, Modal, Form, Input, InputNumber, Select, Switch, message, Popconfirm } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, BellOutlined, ReloadOutlined } from '@ant-design/icons'
import { alertRulesApi, AlertRule } from '../../api/client'

const AlertRules: React.FC = () => {
  const { t } = useTranslation()
  const [rules, setRules] = useState<AlertRule[]>([])
  const [loading, setLoading] = useState(false)
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<AlertRule | null>(null)
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 })
  const [form] = Form.useForm()

  const fetchRules = useCallback(async (page = 1, pageSize = 10) => {
    setLoading(true)
    try {
      const response = await alertRulesApi.list({ page, page_size: pageSize })
      if (response.code === 0) {
        setRules(response.data || [])
        setPagination({
          current: page,
          pageSize,
          total: response.meta?.total || 0
        })
      }
    } catch (error) {
      message.error(t('alert.loadingRules'))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    fetchRules()
  }, [fetchRules])

  const metricOptions = [
    { label: t('metric.cpuUsage'), value: 'cpu_usage' },
    { label: t('metric.memoryUsage'), value: 'memory_usage' },
    { label: t('metric.diskUsage'), value: 'disk_usage' },
    { label: t('metric.networkIn'), value: 'network_in' },
    { label: t('metric.networkOut'), value: 'network_out' },
    { label: t('metric.vmStatus'), value: 'vm_status' }
  ]

  const conditionOptions = [
    { label: t('condition.greaterThan'), value: '>' },
    { label: t('condition.lessThan'), value: '<' },
    { label: t('condition.equalTo'), value: '=' },
    { label: t('condition.notEqualTo'), value: '!=' }
  ]

  const severityOptions = [
    { label: t('severity.critical'), value: 'critical', color: 'red' },
    { label: t('severity.warning'), value: 'warning', color: 'orange' },
    { label: t('severity.info'), value: 'info', color: 'blue' }
  ]

  const channelOptions = [
    { label: t('channel.email'), value: 'email' },
    { label: t('channel.dingTalk'), value: 'dingtalk' },
    { label: t('channel.webhook'), value: 'webhook' }
  ]

  const handleAdd = () => {
    setEditingRule(null)
    form.resetFields()
    setIsModalOpen(true)
  }

  const handleEdit = (rule: AlertRule) => {
    setEditingRule(rule)
    form.setFieldsValue({
      ...rule,
      notifyChannels: rule.notifyChannels || []
    })
    setIsModalOpen(true)
  }

  const handleDelete = async (id: string) => {
    try {
      await alertRulesApi.delete(id)
      message.success(t('alert.ruleDeleted'))
      fetchRules(pagination.current, pagination.pageSize)
    } catch (error) {
      message.error(t('alert.failedToDeleteRule'))
    }
  }

  const handleSubmit = async (values: any) => {
    try {
      if (editingRule) {
        await alertRulesApi.update(editingRule.id, values)
        message.success(t('alert.ruleUpdated'))
      } else {
        await alertRulesApi.create(values)
        message.success(t('alert.ruleCreated'))
      }
      setIsModalOpen(false)
      fetchRules(pagination.current, pagination.pageSize)
    } catch (error) {
      message.error(t('alert.failedToSaveRule'))
    }
  }

  const handleToggle = async (id: string) => {
    try {
      await alertRulesApi.toggle(id)
      message.success(t('alert.ruleStatusUpdated'))
      fetchRules(pagination.current, pagination.pageSize)
    } catch (error) {
      message.error(t('alert.failedToUpdateRuleStatus'))
    }
  }

  const handleTableChange = (paginationInfo: any) => {
    fetchRules(paginationInfo.current, paginationInfo.pageSize)
  }

  const columns = [
    {
      title: t('alerts.ruleName'),
      dataIndex: 'name',
      key: 'name'
    },
    {
      title: t('alerts.metric'),
      dataIndex: 'metric',
      key: 'metric',
      render: (metric: string) => {
        const option = metricOptions.find(m => m.value === metric)
        return option?.label || metric
      }
    },
    {
      title: t('alerts.condition'),
      key: 'condition',
      render: (_: any, record: AlertRule) => (
        <span>{record.metric} {record.condition} {record.threshold}</span>
      )
    },
    {
      title: t('alerts.duration'),
      dataIndex: 'duration',
      key: 'duration',
      render: (duration: number) => `${duration} ${t('unit.minutes')}`
    },
    {
      title: t('alerts.severity'),
      dataIndex: 'severity',
      key: 'severity',
      render: (severity: string) => {
        const option = severityOptions.find(s => s.value === severity)
        return <Tag color={option?.color}>{option?.label}</Tag>
      }
    },
    {
      title: t('alerts.notifyChannels'),
      dataIndex: 'notifyChannels',
      key: 'notifyChannels',
      render: (channels: string[]) => (
        <Space>
          {channels?.map(c => (
            <Tag key={c}>{c}</Tag>
          ))}
        </Space>
      )
    },
    {
      title: t('alerts.status'),
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled: boolean, record: AlertRule) => (
        <Switch
          checked={enabled}
          onChange={() => handleToggle(record.id)}
          checkedChildren={t('option.on')}
          unCheckedChildren={t('option.off')}
        />
      )
    },
    {
      title: t('table.action'),
      key: 'actions',
      render: (_: any, record: AlertRule) => (
        <Space>
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          />
          <Popconfirm
            title={t('popconfirm.deleteRule')}
            onConfirm={() => handleDelete(record.id)}
          >
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      )
    }
  ]

  return (
    <div>
      <Card
        title={
          <Space>
            <BellOutlined />
            {t('alerts.alertRules')}
          </Space>
        }
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => fetchRules()}>
              {t('common.refresh')}
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
              {t('alerts.createAlertRule')}
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={rules}
          rowKey="id"
          loading={loading}
          pagination={pagination}
          onChange={handleTableChange}
        />
      </Card>

      <Modal
        title={editingRule ? t('alerts.editAlertRule') : t('alerts.createAlertRule')}
        open={isModalOpen}
        onCancel={() => setIsModalOpen(false)}
        footer={null}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
        >
          <Form.Item
            name="name"
            label={t('alerts.ruleName')}
            rules={[{ required: true, message: t('validation.pleaseEnterRuleName') }]}
          >
            <Input placeholder={t('placeholder.ruleName')} />
          </Form.Item>

          <Form.Item
            name="description"
            label={t('form.description')}
          >
            <Input.TextArea placeholder={t('placeholder.optionalDescription')} rows={2} />
          </Form.Item>

          <Space style={{ width: '100%' }} size={16}>
            <Form.Item
              name="metric"
              label={t('alerts.metric')}
              rules={[{ required: true, message: t('validation.pleaseSelectMetric') }]}
              style={{ flex: 1 }}
            >
              <Select placeholder={t('placeholder.selectMetric')} options={metricOptions} />
            </Form.Item>

            <Form.Item
              name="condition"
              label={t('alerts.condition')}
              rules={[{ required: true, message: t('validation.pleaseSelectCondition') }]}
              style={{ flex: 1 }}
            >
              <Select placeholder={t('placeholder.selectCondition')} options={conditionOptions} />
            </Form.Item>

            <Form.Item
              name="threshold"
              label={t('alerts.threshold')}
              rules={[{ required: true, message: t('validation.pleaseEnterThreshold') }]}
              style={{ flex: 1 }}
            >
              <InputNumber placeholder="Value" style={{ width: '100%' }} />
            </Form.Item>
          </Space>

          <Form.Item
            name="duration"
            label={`${t('alerts.duration')} (${t('unit.minutes')})`}
            rules={[{ required: true, message: t('validation.pleaseEnterDuration') }]}
          >
            <InputNumber min={1} max={60} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="severity"
            label={t('alerts.severity')}
            rules={[{ required: true, message: t('validation.pleaseSelectSeverity') }]}
          >
            <Select placeholder={t('placeholder.selectSeverity')} options={severityOptions} />
          </Form.Item>

          <Form.Item
            name="notifyChannels"
            label={t('alerts.notifyChannels')}
            rules={[{ required: true, message: t('validation.pleaseSelectChannels') }]}
          >
            <Select
              mode="multiple"
              placeholder={t('placeholder.selectChannels')}
              options={channelOptions}
            />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {editingRule ? t('button.update') : t('button.create')}
              </Button>
              <Button onClick={() => setIsModalOpen(false)}>
                {t('common.cancel')}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default AlertRules
