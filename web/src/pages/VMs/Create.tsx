import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Form, Input, Select, InputNumber, Button, Card, Steps, message, Alert, Space } from 'antd'
import { ArrowLeftOutlined } from '@ant-design/icons'
import { vmsApi, templatesApi, Template } from '../../api/client'

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
    setLoading(true)
    try {
      await vmsApi.create({
        name: values.name,
        description: values.description,
        template_id: values.template_id,
        cpu_allocated: values.cpu,
        memory_allocated: values.memory,
        disk_allocated: values.disk
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
    label: `${t.name} (${t.os_type})`,
    value: t.id
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
        { title: 'Select Template' },
        { title: 'Configure' },
        { title: 'Confirm' }
      ]} style={{ marginBottom: 24 }} />

      <Form
        form={form}
        layout="vertical"
        onFinish={handleSubmit}
      >
        <Form.Item
          name="name"
          label={t('vm.name')}
          rules={[{ required: true, message: 'Please enter VM name' }]}
        >
          <Input placeholder="Enter VM name" />
        </Form.Item>

        <Form.Item
          name="description"
          label="Description"
        >
          <Input.TextArea rows={3} placeholder="Optional description" />
        </Form.Item>

        <Form.Item
          name="template_id"
          label="Template"
          rules={[{ required: true, message: 'Please select a template' }]}
        >
          <Select
            placeholder="Select a template"
            options={templateOptions}
            onChange={handleTemplateSelect}
          />
        </Form.Item>

        {selectedTemplate && (
          <Alert
            message={`Selected: ${selectedTemplate.name}`}
            description={`OS: ${selectedTemplate.os_type} | Architecture: ${selectedTemplate.architecture}`}
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
          />
        )}

        <Form.Item
          name="cpu"
          label={t('vm.cpu')}
          rules={[{ required: true, message: 'Please enter CPU count' }]}
        >
          <InputNumber
            min={selectedTemplate?.cpu_min || 1}
            max={selectedTemplate?.cpu_max || 64}
            style={{ width: '100%' }}
          />
        </Form.Item>

        <Form.Item
          name="memory"
          label={t('vm.memory')}
          rules={[{ required: true, message: 'Please enter memory size' }]}
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
          rules={[{ required: true, message: 'Please enter disk size' }]}
        >
          <InputNumber
            min={selectedTemplate?.disk_min || 10}
            max={selectedTemplate?.disk_max || 1000}
            style={{ width: '100%' }}
            addonAfter="GB"
          />
        </Form.Item>

        <Form.Item>
          <Space>
            <Button type="primary" htmlType="submit" loading={loading}>
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
