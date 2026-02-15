import React, { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Form, Input, Select, InputNumber, Button, Card, Steps, message, Alert, Space, Collapse, Divider, Radio } from 'antd'
import { ArrowLeftOutlined, CloudServerOutlined, AppstoreOutlined, PlayCircleOutlined } from '@ant-design/icons'
import { vmsApi, templatesApi, isosApi, Template, ISO } from '../../api/client'

const { Panel } = Collapse

const VMCreate: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [form] = Form.useForm()
  const [templates, setTemplates] = useState<Template[]>([])
  const [isos, setISOs] = useState<ISO[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedTemplate, setSelectedTemplate] = useState<Template | null>(null)
  const [selectedISO, setSelectedISO] = useState<ISO | null>(null)
  const [installationMode, setInstallationMode] = useState<'template' | 'iso'>('template')

  useEffect(() => {
    templatesApi.list().then((res: any) => {
      setTemplates(res.data || [])
    })
    isosApi.list({ page: 1, page_size: 100 }).then((res: any) => {
      setISOs(res.data?.list || res.data || [])
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

  const handleISOSelect = (isoId: string) => {
    const iso = isos.find((i) => i.id === isoId)
    if (iso) {
      setSelectedISO(iso)
      form.setFieldsValue({
        cpu: 2,
        memory: 4096,
        disk: 50
      })
    }
  }

  const handleInstallationModeChange = (mode: 'template' | 'iso') => {
    setInstallationMode(mode)
    setSelectedTemplate(null)
    setSelectedISO(null)
    form.setFieldsValue({
      template_id: undefined,
      iso_id: undefined,
      cpu: mode === 'iso' ? 2 : undefined,
      memory: mode === 'iso' ? 4096 : undefined,
      disk: mode === 'iso' ? 50 : undefined
    })
  }

  const handleSubmit = async (values: any) => {
    if (installationMode === 'template' && !selectedTemplate) {
      message.error(t('validation.pleaseSelectTemplate'))
      return
    }

    if (installationMode === 'iso' && !selectedISO) {
      message.error(t('validation.pleaseSelectISO'))
      return
    }

    setLoading(true)
    try {
      const payload: any = {
        name: values.name,
        description: values.description,
        cpu_allocated: values.cpu,
        memory_allocated: values.memory,
        disk_allocated: values.disk,
        autostart: values.autostart || false,
        installation_mode: installationMode
      }

      if (installationMode === 'template') {
        payload.template_id = values.template_id
        payload.boot_order = values.boot_order || 'hd,cdrom,network'
      } else {
        payload.iso_id = values.iso_id
        payload.boot_order = 'cdrom,hd,network'
      }

      await vmsApi.create(payload)
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
          {t.osType} | {t.architecture?.toUpperCase() || 'X86_64'} | {t.format?.toUpperCase() || 'QCOW2'}
        </span>
      </Space>
    ),
    value: t.id,
    disabled: !t.isActive
  }))

  const isoOptions = isos.map((i) => ({
    label: (
      <Space>
        <span>{i.name}</span>
        <span style={{ color: '#999', fontSize: 12 }}>
          {i.osType || 'Unknown'} | {i.architecture?.toUpperCase() || 'X86_64'} | {(i.fileSize / 1024 / 1024 / 1024).toFixed(2)} GB
        </span>
      </Space>
    ),
    value: i.id
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
        { title: t('vm.selectInstallationMode') },
        { title: installationMode === 'template' ? t('vm.selectTemplate') : t('vm.selectISO') },
        { title: t('vm.configureResources') },
        { title: t('vm.confirmCreate') }
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
              label={t('vm.selectInstallationMode')}
              required
            >
              <Radio.Group
                value={installationMode}
                onChange={(e) => handleInstallationModeChange(e.target.value)}
                style={{ width: '100%' }}
              >
                <Space direction="vertical" style={{ width: '100%' }}>
                  <Radio.Button value="template" style={{ width: '100%', height: 'auto', padding: '12px 16px' }}>
                    <Space>
                      <AppstoreOutlined style={{ fontSize: 20 }} />
                      <div>
                        <div style={{ fontWeight: 500 }}>{t('vm.templateMode')}</div>
                        <div style={{ fontSize: 12, color: '#666' }}>{t('vm.templateModeDesc')}</div>
                      </div>
                    </Space>
                  </Radio.Button>
                  <Radio.Button value="iso" style={{ width: '100%', height: 'auto', padding: '12px 16px' }}>
                    <Space>
                      <PlayCircleOutlined style={{ fontSize: 20 }} />
                      <div>
                        <div style={{ fontWeight: 500 }}>{t('vm.isoMode')}</div>
                        <div style={{ fontSize: 12, color: '#666' }}>{t('vm.isoModeDesc')}</div>
                      </div>
                    </Space>
                  </Radio.Button>
                </Space>
              </Radio.Group>
            </Form.Item>

            {installationMode === 'template' && (
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
            )}

            {installationMode === 'iso' && (
              <Form.Item
                name="iso_id"
                label={t('vm.selectISO')}
                rules={[{ required: true, message: t('validation.pleaseSelectISO') }]}
              >
                <Select
                  placeholder="Select an ISO image"
                  options={isoOptions}
                  onChange={handleISOSelect}
                  showSearch
                  filterOption={(input, option) =>
                    (option?.label?.toString() || '').toLowerCase().includes(input.toLowerCase())
                  }
                />
              </Form.Item>
            )}

            {selectedTemplate && installationMode === 'template' && (
              <Alert
                message={`${t('vm.template')}: ${selectedTemplate.name}`}
                description={
                  <Space direction="vertical" size={4}>
                    <span><strong>OS:</strong> {selectedTemplate.osType} {selectedTemplate.osVersion}</span>
                    <span><strong>Architecture:</strong> {selectedTemplate.architecture?.toUpperCase() || 'X86_64'}</span>
                    <span><strong>Format:</strong> {selectedTemplate.format?.toUpperCase() || 'QCOW2'}</span>
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

            {selectedISO && installationMode === 'iso' && (
              <Alert
                message={`${t('vm.selectISO')}: ${selectedISO.name}`}
                description={
                  <Space direction="vertical" size={4}>
                    <span><strong>OS:</strong> {selectedISO.osType || 'Unknown'} {selectedISO.osVersion || ''}</span>
                    <span><strong>Architecture:</strong> {selectedISO.architecture?.toUpperCase() || 'X86_64'}</span>
                    <span><strong>Size:</strong> {(selectedISO.fileSize / 1024 / 1024 / 1024).toFixed(2)} GB</span>
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
                { type: 'integer', min: 1, max: 64, message: 'CPU must be between 1 and 64' }
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
                { type: 'integer', min: 512, max: 131072, message: 'Memory must be between 512 and 131072 MB' }
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
                { type: 'integer', min: 10, max: 1000, message: 'Disk must be between 10 and 1000 GB' }
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

        {installationMode === 'template' && (
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
        )}

        {installationMode === 'iso' && (
          <Collapse bordered={false} style={{ marginBottom: 16 }}>
            <Panel header={<strong>{t('form.advancedOptions')}</strong>} key="advanced">
              <Form.Item
                name="autostart"
                label={t('form.autoStart')}
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
        )}

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
