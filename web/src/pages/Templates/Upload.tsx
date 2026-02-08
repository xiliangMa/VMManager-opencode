import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Form, Input, Select, InputNumber, Button, Card, Steps, message, Alert, Space, Row, Col } from 'antd'
import { ArrowLeftOutlined, UploadOutlined } from '@ant-design/icons'
import { templatesApi } from '../../api/client'

const TemplateUpload: React.FC = () => {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (values: any) => {
    setLoading(true)
    try {
      const templateData = {
        name: values.name,
        description: values.description,
        os_type: values.os_type,
        os_version: values.os_version,
        architecture: values.architecture,
        format: values.format,
        cpu_min: values.cpu_min,
        cpu_max: values.cpu_max,
        memory_min: values.memory_min,
        memory_max: values.memory_max,
        disk_min: values.disk_min,
        disk_max: values.disk_max,
        disk_size: values.disk_max,
        is_public: values.is_public
      }

      await templatesApi.create(templateData)
      message.success(t('template.upload') + ' ' + t('common.success'))
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
          {t('template.upload')}
        </Space>
      }
    >
      <Steps current={0} items={[
        { title: 'Basic Info' },
        { title: 'Upload File' },
        { title: 'Confirm' }
      ]} style={{ marginBottom: 24 }} />

      <Form
        form={form}
        layout="vertical"
        onFinish={handleSubmit}
        initialValues={{
          architecture: 'x86_64',
          format: 'qcow2',
          cpu_min: 1,
          cpu_max: 4,
          memory_min: 1024,
          memory_max: 8192,
          disk_min: 20,
          disk_max: 500,
          is_public: true
        }}
      >
        <Form.Item
          name="name"
          label={t('template.name')}
          rules={[{ required: true, message: 'Please enter template name' }]}
        >
          <Input placeholder="Enter template name" />
        </Form.Item>

        <Form.Item
          name="description"
          label="Description"
        >
          <Input.TextArea rows={3} placeholder="Optional description" />
        </Form.Item>

        <Form.Item
          name="os_type"
          label={t('template.osType')}
          rules={[{ required: true, message: 'Please select OS type' }]}
        >
          <Select placeholder="Select OS" options={osOptions} />
        </Form.Item>

        <Form.Item
          name="os_version"
          label="OS Version"
        >
          <Input placeholder="e.g., 22.04 LTS" />
        </Form.Item>

        <Form.Item
          name="architecture"
          label={t('template.architecture')}
        >
          <Select options={archOptions} />
        </Form.Item>

        <Form.Item
          name="format"
          label={t('template.format')}
        >
          <Select options={formatOptions} />
        </Form.Item>

        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="cpu_min" label="CPU Min">
              <InputNumber min={1} max={256} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="cpu_max" label="CPU Max">
              <InputNumber min={1} max={256} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="memory_min" label="Memory Min (MB)">
              <InputNumber min={512} max={524288} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="memory_max" label="Memory Max (MB)">
              <InputNumber min={512} max={524288} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="disk_min" label="Disk Min (GB)">
              <InputNumber min={10} max={10000} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="disk_max" label="Disk Max (GB)">
              <InputNumber min={10} max={10000} style={{ width: '100%' }} />
            </Form.Item>
          </Col>
        </Row>

        <Form.Item
          name="is_public"
          label={t('template.public')}
          valuePropName="checked"
        >
          <Select
            options={[
              { label: t('template.public'), value: true },
              { label: t('template.private'), value: false }
            ]}
          />
        </Form.Item>

        <Alert
          message="Note"
          description="After submitting, upload the actual template file (qcow2, vmdk, etc.) separately."
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
        />

        <Form.Item>
          <Space>
            <Button type="primary" htmlType="submit" loading={loading} icon={<UploadOutlined />}>
              {t('template.upload')}
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

export default TemplateUpload
