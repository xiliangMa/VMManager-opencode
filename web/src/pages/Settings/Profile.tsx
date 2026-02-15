import React, { useState, useEffect, useRef } from 'react'
import { Card, Form, Input, Button, message, Tabs, Select, Avatar } from 'antd'
import { UserOutlined, LockOutlined, GlobalOutlined, UploadOutlined, SaveOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useAuthStore } from '../../stores/authStore'
import { authApi, client } from '../../api/client'

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
      } catch (_error) {
        message.error(t('message.failedToLoad') + ' profile')
      }
    }
    fetchProfile()
  }, [updateUser, t])

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

    const formData = new FormData()
    formData.append('avatar', file)
    try {
      await client.post('/auth/profile/avatar', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      })
      
      message.success(t('message.updatedSuccessfully') + ' avatar')
      const updatedProfile = await authApi.getProfile()
      
      const profileData = updatedProfile.data
      
      if (profileData.avatar) {
        profileData.avatar = profileData.avatar.includes('?') 
          ? `${profileData.avatar}&t=${Date.now()}`
          : `${profileData.avatar}?t=${Date.now()}`
      }
      
      updateUser(profileData)
    } catch (error: any) {
      message.error((error.response?.data?.message || t('message.failedToUpdate')) + ' avatar')
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
              <Avatar 
                size={100} 
                src={user?.avatar} 
                icon={<UserOutlined />}
              />
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
              <Input.Password prefix={<LockOutlined />} />
            </Form.Item>

            <Form.Item
              name="newPassword"
              label={t('form.newPassword')}
              rules={[{ required: true, message: t('validation.pleaseEnterNewPassword') }]}
            >
              <Input.Password prefix={<LockOutlined />} />
            </Form.Item>

            <Form.Item
              name="confirmPassword"
              label={t('form.confirmPassword')}
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
              <Input.Password prefix={<LockOutlined />} />
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
                {t('button.saveChanges')}
              </Button>
            </Form.Item>
          </Form>
        </Card>
      )
    }
  ]

  return (
    <div>
      <Tabs items={tabItems} />
    </div>
  )
}

export default Profile