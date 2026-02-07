import React, { useState } from 'react'
import { Card, Table, Button, Tag, Space, Modal, Form, Input, InputNumber, Select, Switch, message, Popconfirm } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, BellOutlined } from '@ant-design/icons'

interface AlertRule {
  id: string
  name: string
  metric: string
  condition: string
  threshold: number
  duration: number
  severity: string
  enabled: boolean
  notifyChannels: string[]
  createdAt: string
}

const initialRules: AlertRule[] = [
  {
    id: '1',
    name: 'High CPU Usage',
    metric: 'cpu_usage',
    condition: '>',
    threshold: 90,
    duration: 5,
    severity: 'critical',
    enabled: true,
    notifyChannels: ['email', 'dingtalk'],
    createdAt: '2024-01-01 00:00:00'
  },
  {
    id: '2',
    name: 'High Memory Usage',
    metric: 'memory_usage',
    condition: '>',
    threshold: 85,
    duration: 5,
    severity: 'warning',
    enabled: true,
    notifyChannels: ['email'],
    createdAt: '2024-01-01 00:00:00'
  },
  {
    id: '3',
    name: 'Disk Usage High',
    metric: 'disk_usage',
    condition: '>',
    threshold: 80,
    duration: 10,
    severity: 'warning',
    enabled: true,
    notifyChannels: ['email'],
    createdAt: '2024-01-01 00:00:00'
  },
  {
    id: '4',
    name: 'VM Down',
    metric: 'vm_status',
    condition: '=',
    threshold: 0,
    duration: 1,
    severity: 'critical',
    enabled: true,
    notifyChannels: ['email', 'dingtalk'],
    createdAt: '2024-01-01 00:00:00'
  }
]

const AlertRules: React.FC = () => {
  const [rules, setRules] = useState<AlertRule[]>(initialRules)
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<AlertRule | null>(null)
  const [form] = Form.useForm()

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
    form.setFieldsValue(rule)
    setIsModalOpen(true)
  }

  const handleDelete = (id: string) => {
    setRules(rules.filter(r => r.id !== id))
    message.success('Alert rule deleted')
  }

  const handleSubmit = async (values: any) => {
    if (editingRule) {
      setRules(rules.map(r => r.id === editingRule.id ? { ...r, ...values } : r))
      message.success('Alert rule updated')
    } else {
      const newRule: AlertRule = {
        id: Date.now().toString(),
        ...values,
        createdAt: new Date().toLocaleString()
      }
      setRules([...rules, newRule])
      message.success('Alert rule created')
    }
    setIsModalOpen(false)
  }

  const handleToggle = (id: string) => {
    setRules(rules.map(r => r.id === id ? { ...r, enabled: !r.enabled } : r))
    const rule = rules.find(r => r.id === id)
    message.success(`Alert rule ${rule?.enabled ? 'disabled' : 'enabled'}`)
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
          {channels.map(c => (
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
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            Add Rule
          </Button>
        }
      >
        <Table
          columns={columns}
          dataSource={rules}
          rowKey="id"
          pagination={{ pageSize: 10 }}
        />
      </Card>

      <Modal
        title={editingRule ? 'Edit Alert Rule' : 'Add Alert Rule'}
        open={isModalOpen}
        onCancel={() => setIsModalOpen(false)}
        footer={null}
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
            name="metric"
            label="Metric"
            rules={[{ required: true, message: 'Please select metric' }]}
          >
            <Select placeholder="Select metric" options={metricOptions} />
          </Form.Item>

          <Space style={{ width: '100%' }} size={16}>
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
