import React, { useState, useEffect, useCallback } from 'react'
import { Card, Table, Button, Tag, Space, Modal, Form, Input, InputNumber, Select, Switch, message, Popconfirm } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, BellOutlined, ReloadOutlined } from '@ant-design/icons'
import { alertRulesApi, AlertRule } from '../../api/client'

const AlertRules: React.FC = () => {
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
      message.error('Failed to load alert rules')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchRules()
  }, [fetchRules])

  const metricOptions = [
    { label: 'CPU Usage (%)', value: 'cpu_usage' },
    { label: 'Memory Usage (%)', value: 'memory_usage' },
    { label: 'Disk Usage (%)', value: 'disk_usage' },
    { label: 'Network In (MB/s)', value: 'network_in' },
    { label: 'Network Out (MB/s)', value: 'network_out' },
    { label: 'VM Status', value: 'vm_status' }
  ]

  const conditionOptions = [
    { label: 'Greater than (>)', value: '>' },
    { label: 'Less than (<)', value: '<' },
    { label: 'Equal to (=)', value: '=' },
    { label: 'Not equal to (!=)', value: '!=' }
  ]

  const severityOptions = [
    { label: 'Critical', value: 'critical', color: 'red' },
    { label: 'Warning', value: 'warning', color: 'orange' },
    { label: 'Info', value: 'info', color: 'blue' }
  ]

  const channelOptions = [
    { label: 'Email', value: 'email' },
    { label: 'DingTalk', value: 'dingtalk' },
    { label: 'Webhook', value: 'webhook' }
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
      message.success('Alert rule deleted')
      fetchRules(pagination.current, pagination.pageSize)
    } catch (error) {
      message.error('Failed to delete alert rule')
    }
  }

  const handleSubmit = async (values: any) => {
    try {
      if (editingRule) {
        await alertRulesApi.update(editingRule.id, values)
        message.success('Alert rule updated')
      } else {
        await alertRulesApi.create(values)
        message.success('Alert rule created')
      }
      setIsModalOpen(false)
      fetchRules(pagination.current, pagination.pageSize)
    } catch (error) {
      message.error('Failed to save alert rule')
    }
  }

  const handleToggle = async (id: string) => {
    try {
      await alertRulesApi.toggle(id)
      message.success('Alert rule status updated')
      fetchRules(pagination.current, pagination.pageSize)
    } catch (error) {
      message.error('Failed to update alert rule')
    }
  }

  const handleTableChange = (paginationInfo: any) => {
    fetchRules(paginationInfo.current, paginationInfo.pageSize)
  }

  const columns = [
    {
      title: 'Name',
      dataIndex: 'name',
      key: 'name'
    },
    {
      title: 'Metric',
      dataIndex: 'metric',
      key: 'metric',
      render: (metric: string) => {
        const option = metricOptions.find(m => m.value === metric)
        return option?.label || metric
      }
    },
    {
      title: 'Condition',
      key: 'condition',
      render: (_: any, record: AlertRule) => (
        <span>{record.metric} {record.condition} {record.threshold}</span>
      )
    },
    {
      title: 'Duration',
      dataIndex: 'duration',
      key: 'duration',
      render: (duration: number) => `${duration} min`
    },
    {
      title: 'Severity',
      dataIndex: 'severity',
      key: 'severity',
      render: (severity: string) => {
        const option = severityOptions.find(s => s.value === severity)
        return <Tag color={option?.color}>{option?.label}</Tag>
      }
    },
    {
      title: 'Channels',
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
      title: 'Status',
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled: boolean, record: AlertRule) => (
        <Switch
          checked={enabled}
          onChange={() => handleToggle(record.id)}
          checkedChildren="On"
          unCheckedChildren="Off"
        />
      )
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: any, record: AlertRule) => (
        <Space>
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          />
          <Popconfirm
            title="Delete this alert rule?"
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
            Alert Rules
          </Space>
        }
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => fetchRules()}>
              Refresh
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
              Add Rule
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
        title={editingRule ? 'Edit Alert Rule' : 'Add Alert Rule'}
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
            label="Rule Name"
            rules={[{ required: true, message: 'Please enter rule name' }]}
          >
            <Input placeholder="e.g., High CPU Usage" />
          </Form.Item>

          <Form.Item
            name="description"
            label="Description"
          >
            <Input.TextArea placeholder="Optional description" rows={2} />
          </Form.Item>

          <Space style={{ width: '100%' }} size={16}>
            <Form.Item
              name="metric"
              label="Metric"
              rules={[{ required: true, message: 'Please select metric' }]}
              style={{ flex: 1 }}
            >
              <Select placeholder="Select metric" options={metricOptions} />
            </Form.Item>

            <Form.Item
              name="condition"
              label="Condition"
              rules={[{ required: true, message: 'Please select condition' }]}
              style={{ flex: 1 }}
            >
              <Select placeholder="Select condition" options={conditionOptions} />
            </Form.Item>

            <Form.Item
              name="threshold"
              label="Threshold"
              rules={[{ required: true, message: 'Please enter threshold' }]}
              style={{ flex: 1 }}
            >
              <InputNumber placeholder="Value" style={{ width: '100%' }} />
            </Form.Item>
          </Space>

          <Form.Item
            name="duration"
            label="Duration (minutes)"
            rules={[{ required: true, message: 'Please enter duration' }]}
          >
            <InputNumber min={1} max={60} style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="severity"
            label="Severity"
            rules={[{ required: true, message: 'Please select severity' }]}
          >
            <Select placeholder="Select severity" options={severityOptions} />
          </Form.Item>

          <Form.Item
            name="notifyChannels"
            label="Notification Channels"
            rules={[{ required: true, message: 'Please select channels' }]}
          >
            <Select
              mode="multiple"
              placeholder="Select channels"
              options={channelOptions}
            />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {editingRule ? 'Update' : 'Create'}
              </Button>
              <Button onClick={() => setIsModalOpen(false)}>
                Cancel
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default AlertRules
