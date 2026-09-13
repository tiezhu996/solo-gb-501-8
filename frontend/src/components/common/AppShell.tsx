import { AuditOutlined, ExperimentOutlined, LogoutOutlined, MenuFoldOutlined, MenuUnfoldOutlined, ProductOutlined, SafetyCertificateOutlined, SettingOutlined } from '@ant-design/icons'
import { Avatar, Button, Dropdown, Layout, Menu, Space, Typography } from 'antd'
import { useState } from 'react'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '../../hooks/useAuth'
import { roleLabels } from '../../utils/permissions'

const { Header, Sider, Content } = Layout
const navigation = [
  { key: '/lines', icon: <SettingOutlined />, label: '产线总览', permission: undefined },
  { key: '/batches', icon: <ProductOutlined />, label: '批次队列', permission: undefined },
  { key: '/inspections', icon: <ExperimentOutlined />, label: '检验工作台', permission: undefined },
  { key: '/release', icon: <SafetyCertificateOutlined />, label: '放行审批', permission: undefined },
  { key: '/audit', icon: <AuditOutlined />, label: '审计记录', permission: 'audit:read' },
]

export function AppShell() {
  const [collapsed, setCollapsed] = useState(false)
  const { user, logout, can } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const visibleNavigation = navigation.filter((item) => !item.permission || can(item.permission))
  return (
    <Layout className="app-layout">
      <Sider width={224} collapsed={collapsed} collapsedWidth={0} breakpoint="lg" trigger={null} onBreakpoint={(broken) => setCollapsed(broken)} className="app-sider">
        <div className="brand-mark"><SafetyCertificateOutlined /><span>{collapsed ? 'SP' : '无菌包装放行'}</span></div>
        <Menu theme="dark" selectedKeys={[visibleNavigation.find((item) => location.pathname.startsWith(item.key))?.key || '/lines']} items={visibleNavigation} onClick={({ key }) => navigate(key)} />
      </Sider>
      <Layout>
        <Header className="app-header">
          <Button type="text" aria-label={collapsed ? '展开导航' : '折叠导航'} icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />} onClick={() => setCollapsed(!collapsed)} />
          <Dropdown menu={{ items: [{ key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: () => { logout(); navigate('/login') } }] }}>
            <Space className="user-menu"><Avatar>{user?.displayName.slice(0, 1)}</Avatar><span><Typography.Text strong>{user?.displayName}</Typography.Text><small>{user ? (roleLabels[user.role] === user.displayName ? user.username : roleLabels[user.role]) : ''}</small></span></Space>
          </Dropdown>
        </Header>
        <Content className="app-content"><Outlet /></Content>
      </Layout>
    </Layout>
  )
}
