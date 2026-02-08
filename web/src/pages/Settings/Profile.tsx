import React, { useState, useEffect } from 'react'
import { Card, Form, Input, Button, Space, message, Tabs, Switch, Select, Divider, Avatar, Upload } from 'antd'
import { UserOutlined, LockOutlined, BellOutlined, GlobalOutlined, UploadOutlined, SaveOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '../../stores/authStore'
import { authApi } from '../../api/client'

const Profile: React.FC = () => {
  const { t, i18n } = useTranslation()
  const { user, setUser, token } = useAuthStore()
  const [profileForm] = Form.useForm()
  const [passwordForm] = Form.useForm()
  const [preferencesForm] = Form.useForm()
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    if (user) {
      profileForm.setFieldsValue({
        username: user.username,
        email: user.email
      })
      preferencesForm.setFieldsValue({
        language: user.language || 'zh-CN',
        timezone: user.timezone || 'Asia/Shanghai'
      })
    }
  }, [user, profileForm, preferencesForm])

  const handleProfileUpdate = async (values: any) => {
    setLoading(true)
    try {
      await authApi.updateProfile(values)
      message.success('Profile updated successfully')
    } catch (error) {
      message.error('Failed to update profile')
    } finally {
      setLoading(false)
    }
  }

  const handlePasswordChange = async (values: any) => {
    setLoading(true)
    try {
      await authApi.updateProfile({ password: values.newPassword })
      message.success('Password changed successfully')
      passwordForm.resetFields()
    } catch (error) {
      message.error('Failed to change password')
    } finally {
      setLoading(false)
    }
  }

  const handlePreferencesUpdate = async (values: any) => {
    setLoading(true)
    try {
      await authApi.updateProfile(values)
      message.success('Preferences updated successfully')
      i18n.changeLanguage(values.language)
    } catch (error) {
      message.error('Failed to update preferences')
    } finally {
      setLoading(false)
    }
  }

  const languageOptions = [
    { label: '中文 (简体)', value: 'zh-CN' },
    { label: 'English', value: 'en-US' }
  ]

  const timezoneOptions = [
    { label: 'Asia/Shanghai (UTC+8)', value: 'Asia/Shanghai' },
    { label: 'America/New_York (UTC-5)', value: 'America/New_York' },
    { label: 'Europe/London (UTC+0)', value: 'Europe/London' },
    { label: 'Asia/Tokyo (UTC+9)', value: 'Asia/Tokyo' }
  ]

  const tabItems = [
    {
      key: 'profile',
      label: (
        <span>
          <UserOutlined />
          Profile
        </span>
      ),
      children: (
        <Card>
          <div style={{ marginBottom: 24, textAlign: 'center' }}>
            <Upload showUploadList={false}>
              <div style={{ cursor: 'pointer' }}>
                <Avatar size={100} icon={<UserOutlined />} />
                <div style={{ marginTop: 8 }}>
                  <Button icon={<UploadOutlined />}>Change Avatar</Button>
                </div>
              </div>
            </Upload>
          </div>

          <Form
            form={profileForm}
            layout="vertical"
            onFinish={handleProfileUpdate}
          >
            <Form.Item
              name="username"
              label="Username"
              rules={[{ required: true, message: 'Please enter username' }]}
            >
              <Input disabled prefix={<UserOutlined />} />
            </Form.Item>

            <Form.Item
              name="email"
              label="Email"
              rules={[
                { required: true, message: 'Please enter email' },
                { type: 'email', message: 'Please enter a valid email' }
              ]}
            >
              <Input prefix="@" />
            </Form.Item>

            <Form.Item>
              <Button type="primary" htmlType="submit" loading={loading} icon={<SaveOutlined />}>
                Save Changes
              </Button>
            </Form.Item>
          </Form>
        </Card>
      )
    },
    {
      key: 'password',
      label: (
        <span>
          <LockOutlined />
          Security
        </span>
      ),
      children: (
        <Card title="Change Password">
          <Form
            form={passwordForm}
            layout="vertical"
            onFinish={handlePasswordChange}
          >
            <Form.Item
              name="currentPassword"
              label="Current Password"
              rules={[{ required: true, message: 'Please enter current password' }]}
            >
              <Input.Password prefix={<LockOutlined />} />
            </Form.Item>

            <Form.Item
              name="newPassword"
              label="New Password"
              rules={[
                { required: true, message: 'Please enter new password' },
                { min: 6, message: 'Password must be at least 6 characters' }
              ]}
            >
              <Input.Password prefix={<LockOutlined />} />
            </Form.Item>

            <Form.Item
              name="confirmPassword"
              label="Confirm New Password"
              dependencies={['newPassword']}
              rules={[
                { required: true, message: 'Please confirm password' },
                ({ getFieldValue }) => ({
                  validator(_, value) {
                    if (!value || getFieldValue('newPassword') === value) {
                      return Promise.resolve()
                    }
                    return Promise.reject(new Error('Passwords do not match'))
                  }
                })
              ]}
            >
              <Input.Password prefix={<LockOutlined />} />
            </Form.Item>

            <Form.Item>
              <Button type="primary" htmlType="submit" loading={loading}>
                Change Password
              </Button>
            </Form.Item>
          </Form>
        </Card>
      )
    },
    {
      key: 'preferences',
      label: (
        <span>
          <GlobalOutlined />
          Preferences
        </span>
      ),
      children: (
        <Card title="Display Preferences">
          <Form
            form={preferencesForm}
            layout="vertical"
            onFinish={handlePreferencesUpdate}
          >
            <Form.Item
              name="language"
              label="Language"
              rules={[{ required: true, message: 'Please select language' }]}
            >
              <Select options={languageOptions} />
            </Form.Item>

            <Form.Item
              name="timezone"
              label="Timezone"
              rules={[{ required: true, message: 'Please select timezone' }]}
            >
              <Select options={timezoneOptions} />
            </Form.Item>

            <Divider />

            <h4>Notifications</h4>

            <Form.Item name="emailNotifications" label="Email Notifications" valuePropName="checked">
              <Switch />
            </Form.Item>

            <Form.Item name="vmAlerts" label="VM Status Alerts" valuePropName="checked">
              <Switch />
            </Form.Item>

            <Form.Item name="securityAlerts" label="Security Alerts" valuePropName="checked">
              <Switch />
            </Form.Item>

            <Form.Item>
              <Button type="primary" htmlType="submit" loading={loading} icon={<SaveOutlined />}>
                Save Preferences
              </Button>
            </Form.Item>
          </Form>
        </Card>
      )
    }
  ]

  return (
    <div>
      <Card title="Settings">
        <Tabs items={tabItems} />
      </Card>
    </div>
  )
}

export default Profile
