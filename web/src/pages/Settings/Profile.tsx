import React, { useState, useEffect, useRef } from 'react'
import { Card, Form, Input, Button, message, Tabs, Select, Avatar } from 'antd'
import { UserOutlined, LockOutlined, GlobalOutlined, UploadOutlined, SaveOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '../../stores/authStore'
import { authApi } from '../../api/client'

const Profile: React.FC = () => {
  const { t, i18n } = useTranslation()
  const { user, updateUser } = useAuthStore()
  const [profileForm] = Form.useForm()
  const [passwordForm] = Form.useForm()
  const [preferencesForm] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    const fetchProfile = async () => {
      try {
        const response = await authApi.getProfile()
        updateUser(response.data)
      } catch (error) {
        console.error(t('message.failedToLoad') + ' profile:', error)
      }
    }
    fetchProfile()
  }, [updateUser])

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
      message.success(t('message.updatedSuccessfully') + ' profile')
      const updatedProfile = await authApi.getProfile()
      updateUser(updatedProfile.data)
    } catch (error) {
      message.error(t('message.failedToUpdate') + ' profile')
    } finally {
      setLoading(false)
    }
  }

  const handlePasswordChange = async (values: any) => {
    setLoading(true)
    try {
      await authApi.updateProfile({ password: values.newPassword })
      message.success(t('message.updatedSuccessfully') + ' password')
      passwordForm.resetFields()
    } catch (error) {
      message.error(t('message.failedToUpdate') + ' password')
    } finally {
      setLoading(false)
    }
  }

  const handlePreferencesUpdate = async (values: any) => {
    setLoading(true)
    try {
      await authApi.updateProfile(values)
      message.success(t('message.updatedSuccessfully') + ' preferences')
      i18n.changeLanguage(values.language)
      const updatedProfile = await authApi.getProfile()
      updateUser(updatedProfile.data)
    } catch (error) {
      message.error(t('message.failedToUpdate') + ' preferences')
    } finally {
      setLoading(false)
    }
  }

  const handleAvatarClick = () => {
    fileInputRef.current?.click()
  }

  const handleAvatarChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    console.log('Avatar upload started:', file.name)
    const formData = new FormData()
    formData.append('avatar', file)
    try {
      const token = localStorage.getItem('auth-storage')
        ? JSON.parse(localStorage.getItem('auth-storage') || '{}')?.state?.token
        : ''
      
      const response = await fetch('/api/v1/auth/profile/avatar', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`
        },
        body: formData
      })
      
      console.log('Response status:', response.status)
      
      if (response.ok) {
        message.success(t('message.createdSuccessfully') + ' avatar')
        const updatedProfile = await authApi.getProfile()
        // 添加时间戳参数，强制浏览器重新加载头像
        if (updatedProfile.data.avatar) {
          updatedProfile.data.avatar = updatedProfile.data.avatar.includes('?') 
            ? `${updatedProfile.data.avatar}&t=${Date.now()}`
            : `${updatedProfile.data.avatar}?t=${Date.now()}`
        }
        updateUser(updatedProfile.data)
      } else {
        message.error(t('message.failedToCreate') + ' avatar')
      }
    } catch (error) {
      console.error('Upload error:', error)
      message.error(t('message.failedToCreate') + ' avatar')
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
          {t('form.profile')}
        </span>
      ),
      children: (
        <Card>
          <div style={{ marginBottom: 24, textAlign: 'center' }}>
            <input
              type="file"
              ref={fileInputRef}
              style={{ display: 'none' }}
              accept="image/*"
              onChange={handleAvatarChange}
            />
            <div style={{ cursor: 'pointer' }} onClick={handleAvatarClick}>
              <Avatar size={100} src={user?.avatar} icon={<UserOutlined />} />
              <div style={{ marginTop: 8 }}>
                <Button icon={<UploadOutlined />}>{t('button.changeAvatar')}</Button>
              </div>
            </div>
          </div>

          <Form
            form={profileForm}
            layout="vertical"
            onFinish={handleProfileUpdate}
          >
            <Form.Item
              name="username"
              label={t('auth.username')}
              rules={[{ required: true, message: t('validation.pleaseEnterName') }]}
            >
              <Input disabled prefix={<UserOutlined />} />
            </Form.Item>

            <Form.Item
              name="email"
              label={t('auth.email')}
              rules={[{ required: true, type: 'email', message: t('validation.pleaseEnterValidEmail') }]}
            >
              <Input />
            </Form.Item>

            <Form.Item>
              <Button type="primary" htmlType="submit" loading={loading} icon={<SaveOutlined />}>
                {t('button.saveChanges')}
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
          {t('form.password')}
        </span>
      ),
      children: (
        <Card>
          <Form
            form={passwordForm}
            layout="vertical"
            onFinish={handlePasswordChange}
          >
            <Form.Item
              name="currentPassword"
              label={t('form.currentPassword')}
              rules={[{ required: true, message: t('validation.pleaseEnterCurrentPassword') }]}
            >
              <Input.Password />
            </Form.Item>

            <Form.Item
              name="newPassword"
              label={t('form.newPassword')}
              rules={[{ required: true, message: t('validation.pleaseEnterNewPassword') }]}
            >
              <Input.Password />
            </Form.Item>

            <Form.Item
              name="confirmPassword"
              label={t('form.confirmPassword')}
              dependencies={['newPassword']}
              rules={[
                { required: true, message: t('validation.pleaseConfirmPassword') },
                ({ getFieldValue }) => ({
                  validator(_, value) {
                    if (!value || getFieldValue('newPassword') === value) {
                      return Promise.resolve()
                    }
                    return Promise.reject(new Error(t('validation.passwordsDoNotMatch')))
                  }
                })
              ]}
            >
              <Input.Password />
            </Form.Item>

            <Form.Item>
              <Button type="primary" htmlType="submit" loading={loading} icon={<SaveOutlined />}>
                {t('button.changePassword')}
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
          {t('form.preferences')}
        </span>
      ),
      children: (
        <Card>
          <Form
            form={preferencesForm}
            layout="vertical"
            onFinish={handlePreferencesUpdate}
          >
            <Form.Item
              name="language"
              label={t('form.language')}
              rules={[{ required: true, message: t('validation.pleaseSelectLanguage') }]}
            >
              <Select options={languageOptions} />
            </Form.Item>

            <Form.Item
              name="timezone"
              label={t('form.timezone')}
              rules={[{ required: true, message: t('validation.pleaseSelectTimezone') }]}
            >
              <Select options={timezoneOptions} />
            </Form.Item>

            <Form.Item>
              <Button type="primary" htmlType="submit" loading={loading} icon={<SaveOutlined />}>
                {t('button.savePreferences')}
              </Button>
            </Form.Item>
          </Form>
        </Card>
      )
    }
  ]

  return (
    <Card title={t('common.profile')}>
      <Tabs defaultActiveKey="profile" items={tabItems} />
    </Card>
  )
}

export default Profile
