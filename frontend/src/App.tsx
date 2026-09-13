import { App as AntApp, ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import { RouterProvider } from 'react-router-dom'
import { router } from './router'

export default function App() {
  return (
    <ConfigProvider locale={zhCN} theme={{ token: { colorPrimary: '#0f6b58', borderRadius: 6, colorInfo: '#1677a6', fontFamily: 'Inter, "PingFang SC", "Microsoft YaHei", sans-serif' }, components: { Layout: { siderBg: '#17322f', headerBg: '#ffffff' }, Menu: { darkItemBg: '#17322f', darkItemSelectedBg: '#0f6b58' } } }}>
      <AntApp><RouterProvider router={router} /></AntApp>
    </ConfigProvider>
  )
}

