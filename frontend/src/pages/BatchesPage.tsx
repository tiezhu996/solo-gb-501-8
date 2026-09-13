import { EyeOutlined, PauseCircleOutlined, PlayCircleOutlined, PlusOutlined, SearchOutlined } from '@ant-design/icons'
import { Button, Col, Form, Input, InputNumber, Modal, Row, Select, Space, Typography, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { batchAPI, lineAPI } from '../api'
import { BatchStatusBadge } from '../components/common/BatchStatusBadge'
import { EntityTable } from '../components/common/EntityTable'
import { useAuth } from '../hooks/useAuth'
import { usePagination } from '../hooks/usePagination'
import { useBatchStore } from '../stores/batchStore'
import type { BatchStatus, PackagingLine, ProductionBatch } from '../types/domain'
import { formatDateTime, formatNumber } from '../utils/format'

export function BatchesPage() {
  const { data, loading, load } = useBatchStore()
  const pagination = usePagination()
  const { can } = useAuth()
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [status, setStatus] = useState<BatchStatus | undefined>()
  const [lines, setLines] = useState<PackagingLine[]>([])
  const [open, setOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm()
  const refresh = () => load({ page: pagination.page, pageSize: pagination.pageSize, search, status })
  useEffect(() => { void refresh() }, [pagination.page, pagination.pageSize, status])
  useEffect(() => { void lineAPI.list({ page: 1, pageSize: 100 }).then((result) => setLines(result.items.filter((line) => line.active))) }, [])
  const create = async () => {
    const values = await form.validateFields()
    setSaving(true)
    try { await batchAPI.create({ ...values, producedQuantity: values.producedQuantity || 0 }); message.success('生产批次创建成功'); setOpen(false); form.resetFields(); await refresh() } finally { setSaving(false) }
  }
  const transition = async (batch: ProductionBatch, next: BatchStatus) => {
    let reason = ''
    if (next === 'hold') reason = await new Promise<string>((resolve, reject) => {
      let input = ''
      Modal.confirm({ title: `暂停批次 ${batch.batchNo}`, content: <Input.TextArea placeholder="请输入暂停原因" onChange={(event) => { input = event.target.value }} />, onOk: () => input.trim().length >= 3 ? resolve(input) : Promise.reject(new Error('请填写暂停原因')), onCancel: () => reject(new Error('cancel')) })
    }).catch(() => '')
    if (next === 'hold' && !reason) return
    await batchAPI.transition(batch.id, next, reason)
    message.success('批次状态已更新')
    await refresh()
  }
  const columns: ColumnsType<ProductionBatch> = [
    { title: '批次号', dataIndex: 'batchNo', fixed: 'left', render: (value, row) => <Button type="link" className="table-link" onClick={() => navigate(`/batches/${row.id}`)}>{value}</Button> },
    { title: '规格', dataIndex: 'specification' },
    { title: '状态', dataIndex: 'status', render: (value) => <BatchStatusBadge status={value} /> },
    { title: '产线', render: (_, row) => row.packagingLine ? `${row.packagingLine.code} · ${row.packagingLine.name}` : row.packagingLineId },
    { title: '责任班组', dataIndex: 'responsibleTeam' },
    { title: '进度', render: (_, row) => `${formatNumber(row.producedQuantity)} / ${formatNumber(row.plannedQuantity)}` },
    { title: '检验', render: (_, row) => `${row.inspections?.filter((item) => item.result !== 'pending').length || 0}/${row.inspections?.length || 0}` },
    { title: '创建时间', dataIndex: 'createdAt', render: formatDateTime },
    { title: '操作', fixed: 'right', render: (_, row) => <Space><Button size="small" icon={<EyeOutlined />} onClick={() => navigate(`/batches/${row.id}`)}>详情</Button>{row.status === 'draft' && <Button size="small" icon={<PlayCircleOutlined />} disabled={!can('batch:write')} onClick={() => void transition(row, 'running')}>开工</Button>}{row.status === 'running' && <Button size="small" danger icon={<PauseCircleOutlined />} disabled={!can('batch:write')} onClick={() => void transition(row, 'hold')}>暂停</Button>}{['hold', 'rework'].includes(row.status) && <Button size="small" icon={<PlayCircleOutlined />} disabled={!can('batch:write')} onClick={() => void transition(row, 'running')}>恢复</Button>}</Space> },
  ]
  return (
    <div className="page-stack">
      <header className="page-header"><div><Typography.Title level={2}>批次队列</Typography.Title><Typography.Text type="secondary">安排包装生产并跟踪检验与放行进度</Typography.Text></div><Button type="primary" icon={<PlusOutlined />} disabled={!can('batch:write')} onClick={() => setOpen(true)}>新建批次</Button></header>
      <div className="table-toolbar"><Input allowClear prefix={<SearchOutlined />} placeholder="搜索批次号、规格或班组" value={search} onChange={(event) => setSearch(event.target.value)} onPressEnter={() => void refresh()} /><Select allowClear placeholder="全部状态" value={status} onChange={setStatus} options={[{ value: 'draft', label: '草稿' }, { value: 'running', label: '生产中' }, { value: 'hold', label: '暂停/隔离' }, { value: 'rework', label: '返工' }, { value: 'released', label: '已放行' }]} /><Button onClick={() => void refresh()}>查询</Button></div>
      <EntityTable columns={columns} dataSource={data.items} loading={loading} emptyTitle="暂无生产批次" emptyActionLabel={can('batch:write') ? '新建批次' : undefined} onEmptyAction={() => setOpen(true)} pagination={{ current: pagination.page, pageSize: pagination.pageSize, total: data.total, onChange: pagination.update, showSizeChanger: true }} />
      <Modal title="新建生产批次" width={640} open={open} confirmLoading={saving} onOk={() => void create()} onCancel={() => setOpen(false)} okText="创建" cancelText="取消">
        <Form form={form} layout="vertical">
          <Row gutter={16}><Col span={12}><Form.Item name="batchNo" label="批次号" rules={[{ required: true, min: 3 }]}><Input placeholder="B20260822-001" /></Form.Item></Col><Col span={12}><Form.Item name="packagingLineId" label="包装产线" rules={[{ required: true }]}><Select showSearch optionFilterProp="label" options={lines.map((line) => ({ value: line.id, label: `${line.code} · ${line.name}` }))} /></Form.Item></Col></Row>
          <Form.Item name="specification" label="包装规格" rules={[{ required: true }]}><Input placeholder="双层无菌屏障袋 120×180 mm" /></Form.Item>
          <Row gutter={16}><Col span={12}><Form.Item name="responsibleTeam" label="责任班组" rules={[{ required: true }]}><Input /></Form.Item></Col><Col span={12}><Form.Item name="plannedQuantity" label="计划数量" rules={[{ required: true }]}><InputNumber min={1} precision={0} style={{ width: '100%' }} /></Form.Item></Col></Row>
        </Form>
      </Modal>
    </div>
  )
}

