import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Form, Input, Select, InputNumber, Button, Card, Steps, message, Alert, Space, Collapse, Divider } from 'antd'
import { ArrowLeftOutlined, CloudServerOutlined } from '@ant-design/icons'
import { vmsApi, templatesApi, Template } from '../../api/client'

const { Panel } = Collapse

const VMCreate: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [form] = Form.useForm()
  const [templates, setTemplates] = useState<Template[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedTemplate, setSelectedTemplate] = useState<Template | null>(null)

  useEffect(() => {
    templatesApi.list().then((res: any) => {
      setTemplates(res.data || [])
    })
  }, [])

  const handleTemplateSelect = (templateId: string) => {
    const template = templates.find((t) => t.id === templateId)
    if (template) {
      setSelectedTemplate(template)
      form.setFieldsValue({
        cpu: template.cpu_min,
        memory: template.memory_min,
        disk: template.disk_min
      })
    }
  }

  const handleSubmit = async (values: any) => {
    if (!selectedTemplate) {
      message.error('Please select a template')
      return
    }

    setLoading(true)
    try {
      await vmsApi.create({
        name: values.name,
        description: values.description,
        template_id: values.template_id,
        cpu: values.cpu,
        memory: values.memory,
        disk: values.disk,
        autostart: values.autostart || false,
        boot_order: values.boot_order || 'hd,cdrom,network'
      })
      message.success(t('vm.vmList') + ' ' + t('common.success'))
      navigate('/vms')
    } catch (error: any) {
      message.error(error.response?.data?.message || t('common.error'))
    } finally {
      setLoading(false)
    }
  }

  const templateOptions = templates.map((t) => ({
    label: (
      <Space>
        <span>{t.name}</span>
        <span style={{ color: '#999', fontSize: 12 }}>
          {t.os_type} | {t.architecture.toUpperCase()} | {t.format.toUpperCase()}
        </span>
      </Space>
    ),
    value: t.id,
    disabled: !t.is_active
  }))

  return (
    <Card
      title={
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/vms')} />
          {t('vm.createVM')}
        </Space>
      }
    >
      <Steps current={0} items={[
        { title: t('vm.selectTemplate') },
        { title: t('vm.configure') },
        { title: t('vm.confirm') }
      ]} style={{ marginBottom: 24 }} />

      <Form
        form={form}
        layout="vertical"
        onFinish={handleSubmit}
        initialValues={{
          boot_order: 'hd,cdrom,network',
          autostart: false
        }}
      >
        <Collapse defaultActiveKey={['basic']} bordered={false} style={{ marginBottom: 16 }}>
          <Panel header={<strong>Basic Information</strong>} key="basic">
            <Form.Item
              name="name"
              label={t('vm.name')}
              rules={[
                { required: true, message: 'Please enter VM name' },
                { min: 3, max: 50, message: 'Name must be 3-50 characters' },
                { pattern: /^[a-zA-Z0-9][a-zA-Z0-9_-]*$/, message: 'Name can only contain letters, numbers, hyphens and underscores' }
              ]}
              extra="Unique identifier for the VM"
            >
              <Input placeholder="e.g., web-server-01" prefix={<CloudServerOutlined />} />
            </Form.Item>

            <Form.Item
              name="description"
              label="Description"
              extra="Optional description for this VM"
            >
              <Input.TextArea rows={2} placeholder="Brief description of this VM's purpose" />
            </Form.Item>

            <Form.Item
              name="template_id"
              label={t('vm.template')}
              rules={[{ required: true, message: 'Please select a template' }]}
            >
              <Select
                placeholder="Select a template"
                options={templateOptions}
                onChange={handleTemplateSelect}
                showSearch
                filterOption={(input, option) =>
                  (option?.label?.toString() || '').toLowerCase().includes(input.toLowerCase())
                }
              />
            </Form.Item>

            {selectedTemplate && (
              <Alert
                message={`Selected Template: ${selectedTemplate.name}`}
                description={
                  <Space direction="vertical" size={4}>
                    <span><strong>OS:</strong> {selectedTemplate.os_type} {selectedTemplate.os_version}</span>
                    <span><strong>Architecture:</strong> {selectedTemplate.architecture.toUpperCase()}</span>
                    <span><strong>Format:</strong> {selectedTemplate.format.toUpperCase()}</span>
                    <span><strong>CPU Range:</strong> {selectedTemplate.cpu_min} - {selectedTemplate.cpu_max} cores</span>
                    <span><strong>Memory Range:</strong> {selectedTemplate.memory_min} - {selectedTemplate.memory_max} MB</span>
                    <span><strong>Disk Range:</strong> {selectedTemplate.disk_min} - {selectedTemplate.disk_max} GB</span>
                  </Space>
                }
                type="info"
                showIcon
                style={{ marginBottom: 16 }}
              />
            )}
          </Panel>
        </Collapse>

        <Collapse bordered={false} style={{ marginBottom: 16 }}>
          <Panel header={<strong>Resource Configuration</strong>} key="resources">
            <Form.Item
              name="cpu"
              label={t('vm.cpu')}
              rules={[
                { required: true, message: 'Please enter CPU count' },
                { type: 'integer', min: selectedTemplate?.cpu_min || 1, max: selectedTemplate?.cpu_max || 64, message: `CPU must be between ${selectedTemplate?.cpu_min || 1} and ${selectedTemplate?.cpu_max || 64}` }
              ]}
            >
              <InputNumber
                min={selectedTemplate?.cpu_min || 1}
                max={selectedTemplate?.cpu_max || 64}
                style={{ width: '100%' }}
                addonAfter="cores"
              />
            </Form.Item>

            <Form.Item
              name="memory"
              label={t('vm.memory')}
              rules={[
                { required: true, message: 'Please enter memory size' },
                { type: 'integer', min: selectedTemplate?.memory_min || 512, max: selectedTemplate?.memory_max || 131072, message: `Memory must be between ${selectedTemplate?.memory_min || 512} and ${selectedTemplate?.memory_max || 131072} MB` }
              ]}
            >
              <InputNumber
                min={selectedTemplate?.memory_min || 512}
                max={selectedTemplate?.memory_max || 131072}
                style={{ width: '100%' }}
                addonAfter="MB"
              />
            </Form.Item>

            <Form.Item
              name="disk"
              label={t('vm.disk')}
              rules={[
                { required: true, message: 'Please enter disk size' },
                { type: 'integer', min: selectedTemplate?.disk_min || 10, max: selectedTemplate?.disk_max || 1000, message: `Disk must be between ${selectedTemplate?.disk_min || 10} and ${selectedTemplate?.disk_max || 1000} GB` }
              ]}
            >
              <InputNumber
                min={selectedTemplate?.disk_min || 10}
                max={selectedTemplate?.disk_max || 1000}
                style={{ width: '100%' }}
                addonAfter="GB"
              />
            </Form.Item>
          </Panel>
        </Collapse>

        <Collapse bordered={false} style={{ marginBottom: 16 }}>
          <Panel header={<strong>Advanced Options</strong>} key="advanced">
            <Form.Item
              name="boot_order"
              label="Boot Order"
              extra="Order of boot devices"
            >
              <Select
                options={[
                  { label: 'Hard Disk → CD-ROM → Network', value: 'hd,cdrom,network' },
                  { label: 'CD-ROM → Hard Disk → Network', value: 'cdrom,hd,network' },
                  { label: 'Network → Hard Disk → CD-ROM', value: 'network,hd,cdrom' },
                  { label: 'Hard Disk Only', value: 'hd' },
                  { label: 'Network Only (PXE)', value: 'network' }
                ]}
              />
            </Form.Item>

            <Form.Item
              name="autostart"
              label="Auto Start"
              valuePropName="checked"
              extra="Automatically start this VM when the host system boots"
            >
              <Select
                options={[
                  { label: 'Disabled', value: false },
                  { label: 'Enabled', value: true }
                ]}
              />
            </Form.Item>
          </Panel>
        </Collapse>

        <Divider />

        <Form.Item>
          <Space>
            <Button type="primary" htmlType="submit" loading={loading} icon={<CloudServerOutlined />}>
              {t('common.create')}
            </Button>
            <Button onClick={() => navigate('/vms')}>
              {t('common.cancel')}
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  )
}

export default VMCreate
