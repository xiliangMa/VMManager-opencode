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
        cpu: template.cpuMin,
        memory: template.memoryMin,
        disk: template.diskMin
      })
    }
  }

  const handleSubmit = async (values: any) => {
    if (!selectedTemplate) {
      message.error(t('validation.pleaseSelectTemplate'))
      return
    }

    setLoading(true)
    try {
      await vmsApi.create({
        name: values.name,
        description: values.description,
        template_id: values.template_id,
        cpu_allocated: values.cpu,
        memory_allocated: values.memory,
        disk_allocated: values.disk,
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
          {t.osType} | {t.architecture.toUpperCase()} | {t.format.toUpperCase()}
        </span>
      </Space>
    ),
    value: t.id,
    disabled: !t.isActive
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
          <Panel header={<strong>{t('form.basicInfo')}</strong>} key="basic">
            <Form.Item
              name="name"
              label={t('vm.name')}
              rules={[
                { required: true, message: t('validation.pleaseEnterName') },
                { min: 3, max: 50, message: t('validation.nameLength') },
                { pattern: /^[a-zA-Z0-9][a-zA-Z0-9_-]*$/, message: t('validation.namePattern') }
              ]}
              extra={t('helper.uniqueIdentifier')}
            >
              <Input placeholder={t('placeholder.enterVmName')} prefix={<CloudServerOutlined />} />
            </Form.Item>

            <Form.Item
              name="description"
              label={t('form.description')}
              extra={t('helper.optionalDescription')}
            >
              <Input.TextArea rows={2} placeholder={t('placeholder.vmDescription')} />
            </Form.Item>

            <Form.Item
              name="template_id"
              label={t('vm.template')}
              rules={[{ required: true, message: t('validation.pleaseSelectTemplate') }]}
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
                    <span><strong>OS:</strong> {selectedTemplate.osType} {selectedTemplate.osVersion}</span>
                    <span><strong>Architecture:</strong> {selectedTemplate.architecture.toUpperCase()}</span>
                    <span><strong>Format:</strong> {selectedTemplate.format.toUpperCase()}</span>
                    <span><strong>CPU Range:</strong> {selectedTemplate.cpuMin} - {selectedTemplate.cpuMax} cores</span>
                    <span><strong>Memory Range:</strong> {selectedTemplate.memoryMin} - {selectedTemplate.memoryMax} MB</span>
                    <span><strong>Disk Range:</strong> {selectedTemplate.diskMin} - {selectedTemplate.diskMax} GB</span>
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
          <Panel header={<strong>{t('form.resourceConfig')}</strong>} key="resources">
            <Form.Item
              name="cpu"
              label={t('vm.cpu')}
              rules={[
                { required: true, message: t('validation.pleaseEnterCpuCount') },
                { type: 'integer', min: selectedTemplate?.cpuMin || 1, max: selectedTemplate?.cpuMax || 64, message: `CPU must be between ${selectedTemplate?.cpuMin || 1} and ${selectedTemplate?.cpuMax || 64}` }
              ]}
            >
              <InputNumber
                min={selectedTemplate?.cpuMin || 1}
                max={selectedTemplate?.cpuMax || 64}
                style={{ width: '100%' }}
                addonAfter={t('unit.cores')}
              />
            </Form.Item>

            <Form.Item
              name="memory"
              label={t('vm.memory')}
              rules={[
                { required: true, message: t('validation.pleaseEnterMemorySize') },
                { type: 'integer', min: selectedTemplate?.memoryMin || 512, max: selectedTemplate?.memoryMax || 131072, message: `Memory must be between ${selectedTemplate?.memoryMin || 512} and ${selectedTemplate?.memoryMax || 131072} MB` }
              ]}
            >
              <InputNumber
                min={selectedTemplate?.memoryMin || 512}
                max={selectedTemplate?.memoryMax || 131072}
                style={{ width: '100%' }}
                addonAfter={t('unit.mb')}
              />
            </Form.Item>

            <Form.Item
              name="disk"
              label={t('vm.disk')}
              rules={[
                { required: true, message: t('validation.pleaseEnterDiskSize') },
                { type: 'integer', min: selectedTemplate?.diskMin || 10, max: selectedTemplate?.diskMax || 1000, message: `Disk must be between ${selectedTemplate?.diskMin || 10} and ${selectedTemplate?.diskMax || 1000} GB` }
              ]}
            >
              <InputNumber
                min={selectedTemplate?.diskMin || 10}
                max={selectedTemplate?.diskMax || 1000}
                style={{ width: '100%' }}
                addonAfter={t('unit.gb')}
              />
            </Form.Item>
          </Panel>
        </Collapse>

        <Collapse bordered={false} style={{ marginBottom: 16 }}>
          <Panel header={<strong>{t('form.advancedOptions')}</strong>} key="advanced">
            <Form.Item
              name="boot_order"
              label={t('form.bootOrder')}
              extra={t('helper.bootOrderHelp')}
            >
              <Select
                options={[
                  { label: t('option.hardDiskCdromNetwork'), value: 'hd,cdrom,network' },
                  { label: t('option.cdromHardDiskNetwork'), value: 'cdrom,hd,network' },
                  { label: t('option.networkHardDiskCdrom'), value: 'network,hd,cdrom' },
                  { label: t('option.hardDiskOnly'), value: 'hd' },
                  { label: t('option.networkOnly'), value: 'network' }
                ]}
              />
            </Form.Item>

            <Form.Item
              name="autostart"
              label={t('form.autoStart')}
              valuePropName="checked"
              extra={t('helper.autoStartHelp')}
            >
              <Select
                options={[
                  { label: t('option.disabled'), value: false },
                  { label: t('option.enabled'), value: true }
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
