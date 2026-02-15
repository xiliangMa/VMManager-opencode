import React, { useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { Card, Table, Button, Tag, Space, Modal, Form, Input, Select, Switch, message, Popconfirm, Row, Col, Statistic } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, PlayCircleOutlined, PoweroffOutlined, ReloadOutlined, WifiOutlined } from '@ant-design/icons'
import { networksApi, VirtualNetwork } from '../../api/client'

const VirtualNetworks: React.FC = () => {
  const { t } = useTranslation()
  const [networks, setNetworks] = useState<VirtualNetwork[]>([])
  const [loading, setLoading] = useState(false)
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [editingNetwork, setEditingNetwork] = useState<VirtualNetwork | null>(null)
  const [pagination, setPagination] = useState({ current: 1, pageSize: 10, total: 0 })
  const [form] = Form.useForm()

  const fetchNetworks = useCallback(async (page = 1, pageSize = 10) => {
    setLoading(true)
    try {
      const response = await networksApi.list({ page, page_size: pageSize })
      if (response.code === 0) {
        setNetworks(response.data || [])
        setPagination({
          current: page,
          pageSize,
          total: response.meta?.total || 0
        })
      }
    } catch (error) {
      message.error(t('network.failedToList'))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    fetchNetworks()
  }, [fetchNetworks])

  const networkTypeOptions = [
    { label: t('network.nat'), value: 'nat' },
    { label: t('network.bridge'), value: 'bridge' },
    { label: t('network.isolated'), value: 'isolated' }
  ]

  const handleAdd = () => {
    setEditingNetwork(null)
    form.resetFields()
    form.setFieldsValue({
      networkType: 'nat',
      dhcpEnabled: true,
      autostart: true
    })
    setIsModalOpen(true)
  }

  const handleEdit = (network: VirtualNetwork) => {
    setEditingNetwork(network)
    form.setFieldsValue(network)
    setIsModalOpen(true)
  }

  const handleDelete = async (id: string) => {
    try {
      await networksApi.delete(id)
      message.success(t('network.deleteSuccess'))
      fetchNetworks(pagination.current, pagination.pageSize)
    } catch (error) {
      message.error(t('network.failedToDelete'))
    }
  }

  const handleSubmit = async (values: any) => {
    try {
      if (editingNetwork) {
        await networksApi.update(editingNetwork.id, values)
        message.success(t('network.updateSuccess'))
      } else {
        await networksApi.create(values)
        message.success(t('network.createSuccess'))
      }
      setIsModalOpen(false)
      fetchNetworks(pagination.current, pagination.pageSize)
    } catch (error) {
      message.error(t('network.failedToSave'))
    }
  }

  const handleStart = async (id: string) => {
    try {
      await networksApi.start(id)
      message.success(t('network.startSuccess'))
      fetchNetworks(pagination.current, pagination.pageSize)
    } catch (error) {
      message.error(t('network.failedToStart'))
    }
  }

  const handleStop = async (id: string) => {
    try {
      await networksApi.stop(id)
      message.success(t('network.stopSuccess'))
      fetchNetworks(pagination.current, pagination.pageSize)
    } catch (error) {
      message.error(t('network.failedToStop'))
    }
  }

  const handleTableChange = (paginationInfo: any) => {
    fetchNetworks(paginationInfo.current, paginationInfo.pageSize)
  }

  const columns = [
    {
      title: t('network.name'),
      dataIndex: 'name',
      key: 'name'
    },
    {
      title: t('network.type'),
      dataIndex: 'networkType',
      key: 'networkType',
      render: (type: string) => {
        const option = networkTypeOptions.find(t => t.value === type)
        return <Tag>{option?.label || type}</Tag>
      }
    },
    {
      title: t('network.subnet'),
      dataIndex: 'subnet',
      key: 'subnet'
    },
    {
      title: t('network.gateway'),
      dataIndex: 'gateway',
      key: 'gateway'
    },
    {
      title: t('network.dhcp'),
      key: 'dhcp',
      render: (_: any, record: VirtualNetwork) => (
        record.dhcpEnabled ? (
          <span>{record.dhcpStart} - {record.dhcpEnd}</span>
        ) : (
          <Tag>{t('network.disabled')}</Tag>
        )
      )
    },
    {
      title: t('network.status'),
      dataIndex: 'active',
      key: 'active',
      render: (active: boolean) => (
        <Tag color={active ? 'green' : 'default'}>
          {active ? t('network.active') : t('network.inactive')}
        </Tag>
      )
    },
    {
      title: t('network.autostart'),
      dataIndex: 'autostart',
      key: 'autostart',
      render: (autostart: boolean) => (
        <Tag color={autostart ? 'blue' : 'default'}>
          {autostart ? t('option.on') : t('option.off')}
        </Tag>
      )
    },
    {
      title: t('table.action'),
      key: 'actions',
      render: (_: any, record: VirtualNetwork) => (
        <Space>
          {record.active ? (
            <Button
              type="text"
              danger
              icon={<PoweroffOutlined />}
              onClick={() => handleStop(record.id)}
            />
          ) : (
            <Button
              type="text"
              icon={<PlayCircleOutlined />}
              onClick={() => handleStart(record.id)}
            />
          )}
          <Button
            type="text"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          />
          <Popconfirm
            title={t('popconfirm.deleteNetwork')}
            onConfirm={() => handleDelete(record.id)}
          >
            <Button type="text" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      )
    }
  ]

  const activeCount = networks.filter(n => n.active).length
  const inactiveCount = networks.filter(n => !n.active).length

  return (
    <div>
      <Card>
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={6}>
            <Statistic 
              title={t('network.totalNetworks')} 
              value={pagination.total} 
              prefix={<WifiOutlined />} 
            />
          </Col>
          <Col span={6}>
            <Statistic 
              title={t('network.activeNetworks')} 
              value={activeCount} 
              valueStyle={{ color: '#3f8600' }}
            />
          </Col>
          <Col span={6}>
            <Statistic 
              title={t('network.inactiveNetworks')} 
              value={inactiveCount} 
              valueStyle={{ color: '#cf1322' }}
            />
          </Col>
        </Row>

        <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
          <Button onClick={() => fetchNetworks(pagination.current, pagination.pageSize)}>
            <ReloadOutlined /> {t('common.refresh')}
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            {t('network.createNetwork')}
          </Button>
        </div>

        <Table
          columns={columns}
          dataSource={networks}
          rowKey="id"
          loading={loading}
          pagination={{
            current: pagination.current,
            pageSize: pagination.pageSize,
            total: pagination.total,
            showSizeChanger: true,
            showTotal: (total) => `${t('common.total')} ${total} ${t('network.items')}`
          }}
          onChange={handleTableChange}
        />
      </Card>

      <Modal
        title={editingNetwork ? t('network.editNetwork') : t('network.createNetwork')}
        open={isModalOpen}
        onCancel={() => setIsModalOpen(false)}
        onOk={() => form.submit()}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          onFinish={handleSubmit}
        >
          <Form.Item
            name="name"
            label={t('network.name')}
            rules={[{ required: true, message: t('network.nameRequired') }]}
          >
            <Input placeholder={t('network.namePlaceholder')} />
          </Form.Item>

          <Form.Item
            name="description"
            label={t('network.description')}
          >
            <Input.TextArea rows={2} placeholder={t('network.descriptionPlaceholder')} />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="networkType"
                label={t('network.type')}
                rules={[{ required: true }]}
              >
                <Select options={networkTypeOptions} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="bridgeName"
                label={t('network.bridgeName')}
              >
                <Input placeholder={t('network.bridgeNamePlaceholder')} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="subnet"
                label={t('network.subnet')}
                rules={[{ required: true, message: t('network.subnetRequired') }]}
              >
                <Input placeholder="255.255.255.0" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="gateway"
                label={t('network.gateway')}
                rules={[{ required: true, message: t('network.gatewayRequired') }]}
              >
                <Input placeholder="192.168.1.1" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="dhcpEnabled"
            label={t('network.dhcpEnabled')}
            valuePropName="checked"
          >
            <Switch checkedChildren={t('option.on')} unCheckedChildren={t('option.off')} />
          </Form.Item>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="dhcpStart"
                label={t('network.dhcpStart')}
              >
                <Input placeholder="192.168.1.100" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="dhcpEnd"
                label={t('network.dhcpEnd')}
              >
                <Input placeholder="192.168.1.200" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item
            name="autostart"
            label={t('network.autostart')}
            valuePropName="checked"
          >
            <Switch checkedChildren={t('option.on')} unCheckedChildren={t('option.off')} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default VirtualNetworks
