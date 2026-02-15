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

  // 监听用户状态变化，输出日志
  useEffect(() => {
    console.log('User state changed:', user)
    console.log('User avatar:', user?.avatar)
  }, [user])

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
      // 使用client实例发送请求，自动处理认证
      const response = await client.post('/auth/profile/avatar', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      })
      
      console.log('Avatar upload response:', response)
      
      message.success(t('message.updatedSuccessfully') + ' avatar')
      const updatedProfile = await authApi.getProfile()
      console.log('Updated profile:', updatedProfile)
      console.log('Updated profile structure:', typeof updatedProfile)
      
      // authApi.getProfile() 已经返回了 res.data，所以 updatedProfile 就是用户数据对象
      const profileData = updatedProfile
      console.log('Final profile data:', profileData)
      
      // 添加时间戳参数，强制浏览器重新加载头像
      if (profileData.avatar) {
        profileData.avatar = profileData.avatar.includes('?') 
          ? `${profileData.avatar}&t=${Date.now()}`
          : `${profileData.avatar}?t=${Date.now()}`
        console.log('Updated avatar URL:', profileData.avatar)
      }
      
      updateUser(profileData)
      console.log('User updated in store')
      
      // 立即获取更新后的用户状态，验证是否更新成功
      setTimeout(() => {
        const { user } = useAuthStore.getState()
        console.log('User after update:', user)
        console.log('Avatar after update:', user?.avatar)
      }, 100)
    } catch (error: any) {
      console.error('Upload error:', error)
      console.error('Error response:', error.response)
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