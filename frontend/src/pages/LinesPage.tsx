import { PlusOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons'
import { Button, Col, Form, Input, Modal, Row, Select, Space, Statistic, Typography, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useEffect, useState } from 'react'
import { lineAPI } from '../api'
import { EntityTable } from '../components/common/EntityTable'
import { StatusBadge } from '../components/common/StatusBadge'
import { useAuth } from '../hooks/useAuth'
import { usePagination } from '../hooks/usePagination'
import { useLineStore } from '../stores/lineStore'
import type { PackagingLine } from '../types/domain'
import { formatDateTime } from '../utils/format'

export function LinesPage() {
  const { data, loading, load } = useLineStore()
  const pagination = usePagination()
  const { can } = useAuth()
  const [search, setSearch] = useState('')
  const [open, setOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm()
  const refresh = () => load({ page: pagination.page, pageSize: pagination.pageSize, search })
  useEffect(() => { void refresh() }, [pagination.page, pagination.pageSize])
  const create = async () => {
    const values = await form.validateFields()
    setSaving(true)
    try { await lineAPI.create(values); message.success('产线创建成功'); setOpen(false); form.resetFields(); await refresh() } finally { setSaving(false) }
  }
  const columns: ColumnsType<PackagingLine> = [
    { title: '产线', key: 'line', render: (_, row) => <div><strong>{row.code}</strong><small className="cell-subtitle">{row.name}</small></div> },
    { title: '班组', dataIndex: 'team' },
    { title: '设备状态', dataIndex: 'equipmentStatus', render: (value) => <StatusBadge value={value} dot /> },
    { title: '位置', dataIndex: 'location' },
    { title: '当前批次', render: (_, row) => row.batches?.filter((item) => item.status !== 'released').length || 0 },
    { title: '启用状态', dataIndex: 'active', render: (value) => value ? '启用' : '停用' },
    { title: '最近更新', dataIndex: 'updatedAt', render: formatDateTime },
  ]
  return (
    <div className="page-stack">
      <header className="page-header"><div><Typography.Title level={2}>产线总览</Typography.Title><Typography.Text type="secondary">跟踪包装线、班组及正在执行的生产批次</Typography.Text></div><Space><Button icon={<ReloadOutlined />} onClick={() => void refresh()} /><Button type="primary" icon={<PlusOutlined />} disabled={!can('line:write')} onClick={() => setOpen(true)}>新增产线</Button></Space></header>
      <Row gutter={16}><Col span={8}><div className="metric"><Statistic title="产线总数" value={data.total} /></div></Col><Col span={8}><div className="metric"><Statistic title="运行/待机" value={data.items.filter((line) => ['ready', 'running'].includes(line.equipmentStatus)).length} /></div></Col><Col span={8}><div className="metric"><Statistic title="需关注" value={data.items.filter((line) => ['maintenance', 'fault'].includes(line.equipmentStatus)).length} /></div></Col></Row>
      <div className="table-toolbar"><Input allowClear prefix={<SearchOutlined />} placeholder="搜索编码、名称或班组" value={search} onChange={(event) => setSearch(event.target.value)} onPressEnter={() => { pagination.reset(); void refresh() }} /><Button onClick={() => void refresh()}>查询</Button></div>
      <EntityTable columns={columns} dataSource={data.items} loading={loading} emptyTitle="尚未配置包装产线" emptyActionLabel={can('line:write') ? '新增产线' : undefined} onEmptyAction={() => setOpen(true)} pagination={{ current: pagination.page, pageSize: pagination.pageSize, total: data.total, showSizeChanger: true, onChange: pagination.update }} />
      <Modal title="新增包装产线" open={open} confirmLoading={saving} onOk={() => void create()} onCancel={() => setOpen(false)} okText="创建" cancelText="取消">
        <Form form={form} layout="vertical" initialValues={{ equipmentStatus: 'ready' }}>
          <Row gutter={12}><Col span={12}><Form.Item name="code" label="产线编码" rules={[{ required: true, min: 2 }]}><Input placeholder="PKG-01" /></Form.Item></Col><Col span={12}><Form.Item name="name" label="产线名称" rules={[{ required: true }]}><Input /></Form.Item></Col></Row>
          <Form.Item name="team" label="责任班组" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item name="equipmentStatus" label="设备状态" rules={[{ required: true }]}><Select options={[{ value: 'ready', label: '待机' }, { value: 'running', label: '运行中' }, { value: 'maintenance', label: '维护中' }, { value: 'fault', label: '故障' }]} /></Form.Item>
          <Form.Item name="location" label="车间位置"><Input /></Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

