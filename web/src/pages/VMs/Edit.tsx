import React, { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Form, Input, InputNumber, Button, Card, Space, Row, Col, message, Select } from 'antd'
import { ArrowLeftOutlined, SaveOutlined } from '@ant-design/icons'
import { vmsApi } from '../../api/client'

const VMEdit: React.FC = () => {
  const { id } = useParams()
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (id) {
      vmsApi.get(id).then((res: any) => {
        const vm = res.data || res
        form.setFieldsValue({
          name: vm.name,
          cpu_allocated: vm.cpuAllocated,
          memory_allocated: vm.memoryAllocated,
          disk_allocated: vm.diskAllocated,
          boot_order: vm.bootOrder,
          autostart: vm.autostart
        })
      }).catch(() => {
        message.error(t('message.failedToLoad') + ' VM')
      })
    }
  }, [id, form])

  const handleSubmit = async (values: any) => {
    if (!id) return

    setLoading(true)
    try {
      await vmsApi.update(id, {
        name: values.name,
        boot_order: values.boot_order,
        autostart: values.autostart
      })
      message.success(t('common.success'))
      navigate(`/vms/${id}`)
    } catch (error: any) {
      message.error(error.response?.data?.message || t('common.error'))
    } finally {
      setLoading(false)
    }
  }

  const bootOrderOptions = [
    { label: 'Hard Disk → CD-ROM → Network', value: 'hd,cdrom,network' },
    { label: 'CD-ROM → Hard Disk → Network', value: 'cdrom,hd,network' },
    { label: 'Network → Hard Disk → CD-ROM', value: 'network,hd,cdrom' },
    { label: 'Hard Disk Only', value: 'hd' },
    { label: 'Network Only (PXE)', value: 'network' }
  ]

  return (
    <Card
      title={
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(`/vms/${id}`)} />
          {t('vm.edit')}
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
          label={t('vm.name')}
          rules={[{ required: true, message: t('placeholder.enterName') }]}
        >
          <Input placeholder={t('placeholder.enterName')} />
        </Form.Item>

        <Row gutter={16}>
          <Col span={8}>
            <Form.Item name="cpu_allocated" label={t('table.vcpu')}>
              <InputNumber min={1} max={256} style={{ width: '100%' }} disabled />
            </Form.Item>
          </Col>
          <Col span={8}>
            <Form.Item name="memory_allocated" label={t('vm.memory')}>
              <InputNumber min={512} max={524288} style={{ width: '100%' }} disabled />
            </Form.Item>
          </Col>
          <Col span={8}>
            <Form.Item name="disk_allocated" label={t('vm.disk')}>
              <InputNumber min={10} max={10000} style={{ width: '100%' }} disabled />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="boot_order" label="Boot Order">
              <Select options={bootOrderOptions} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="autostart" label="Auto Start">
              <Select
                options={[
                  { label: 'Enabled', value: true },
                  { label: 'Disabled', value: false }
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
            <Button onClick={() => navigate(`/vms/${id}`)}>
              {t('common.cancel')}
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  )
}

export default VMEdit
