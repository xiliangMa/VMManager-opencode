import React, { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Form, Input, Select, InputNumber, Button, Card, Space, Row, Col, message } from 'antd'
import { ArrowLeftOutlined, SaveOutlined } from '@ant-design/icons'
import { templatesApi } from '../../api/client'

const TemplateEdit: React.FC = () => {
  const { id } = useParams()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)

  useEffect(() => {
      if (id) {
        templatesApi.get(id).then((res: any) => {
          const template = res.data
          form.setFieldsValue({
          name: template.name,
          description: template.description,
          osType: template.osType,
          osVersion: template.osVersion,
          architecture: template.architecture,
          format: template.format,
          cpuMin: template.cpuMin,
          cpuMax: template.cpuMax,
          memoryMin: template.memoryMin,
          memoryMax: template.memoryMax,
          diskMin: template.diskMin,
          diskMax: template.diskMax,
          isPublic: template.isPublic,
          isActive: template.isActive
        })
      }).catch(() => {
        message.error('Failed to load template')
      })
    }
  }, [id, form])

  const handleSubmit = async (values: any) => {
    if (!id) return

    setLoading(true)
    try {
      await templatesApi.update(id, {
        name: values.name,
        description: values.description,
        isPublic: values.isPublic,
        isActive: values.isActive
      })
      message.success(t('common.success'))
      navigate('/templates')
    } catch (error: any) {
      message.error(error.response?.data?.message || t('common.error'))
    } finally {
      setLoading(false)
    }
  }

  const osOptions = [
    { label: 'Ubuntu', value: 'Ubuntu' },
    { label: 'CentOS', value: 'CentOS' },
    { label: 'Debian', value: 'Debian' },
    { label: 'Windows', value: 'Windows' },
    { label: 'Rocky Linux', value: 'RockyLinux' }
  ]

  const archOptions = [
    { label: 'x86_64', value: 'x86_64' },
    { label: 'aarch64', value: 'aarch64' },
    { label: 'arm64', value: 'arm64' }
  ]

  const formatOptions = [
    { label: 'qcow2', value: 'qcow2' },
    { label: 'vmdk', value: 'vmdk' },
    { label: 'raw', value: 'raw' },
    { label: 'ova', value: 'ova' }
  ]

  return (
    <Card
      title={
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/templates')} />
          {t('template.edit')}
        </Space>
      }
    >
      <Form
        form={form}
        layout="vertical"
        onFinish={handleSubmit}
      >
        <Form.Item
          name="name"
          label={t('template.name')}
          rules={[{ required: true, message: 'Please enter template name' }]}
        >
          <Input placeholder="Enter template name" />
        </Form.Item>

        <Form.Item name="description" label="Description">
          <Input.TextArea rows={3} placeholder="Optional description" />
        </Form.Item>

        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="osType" label={t('template.osType')}>
              <Select placeholder="Select OS" options={osOptions} disabled />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="osVersion" label="OS Version">
              <Input placeholder="e.g., 22.04 LTS" disabled />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={8}>
            <Form.Item name="architecture" label={t('template.architecture')}>
              <Select options={archOptions} disabled />
            </Form.Item>
          </Col>
          <Col span={8}>
            <Form.Item name="format" label={t('template.format')}>
              <Select options={formatOptions} disabled />
            </Form.Item>
          </Col>
          <Col span={8}>
            <Form.Item name="isActive" label="Active">
              <Select
                options={[
                  { label: 'Active', value: true },
                  { label: 'Inactive', value: false }
                ]}
              />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={6}>
            <Form.Item name="cpuMin" label="CPU Min">
              <InputNumber min={1} max={256} style={{ width: '100%' }} disabled />
            </Form.Item>
          </Col>
          <Col span={6}>
            <Form.Item name="cpuMax" label="CPU Max">
              <InputNumber min={1} max={256} style={{ width: '100%' }} disabled />
            </Form.Item>
          </Col>
          <Col span={6}>
            <Form.Item name="memoryMin" label="Memory Min (MB)">
              <InputNumber min={512} max={524288} style={{ width: '100%' }} disabled />
            </Form.Item>
          </Col>
          <Col span={6}>
            <Form.Item name="memoryMax" label="Memory Max (MB)">
              <InputNumber min={512} max={524288} style={{ width: '100%' }} disabled />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={6}>
            <Form.Item name="diskMin" label="Disk Min (GB)">
              <InputNumber min={10} max={10000} style={{ width: '100%' }} disabled />
            </Form.Item>
          </Col>
          <Col span={6}>
            <Form.Item name="diskMax" label="Disk Max (GB)">
              <InputNumber min={10} max={10000} style={{ width: '100%' }} disabled />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="isPublic" label={t('template.public')}>
              <Select
                options={[
                  { label: t('template.public'), value: true },
                  { label: t('template.private'), value: false }
                ]}
              />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item>
          <Space>
            <Button type="primary" htmlType="submit" loading={loading} icon={<SaveOutlined />}>
              {t('common.save')}
            </Button>
            <Button onClick={() => navigate('/templates')}>
              {t('common.cancel')}
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  )
}

export default TemplateEdit
