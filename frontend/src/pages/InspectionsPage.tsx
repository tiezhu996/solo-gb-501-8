import { CheckOutlined, PlusOutlined, SearchOutlined } from '@ant-design/icons'
import { Button, Col, Form, Input, Modal, Radio, Row, Select, Space, Typography, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useEffect, useState } from 'react'
import { batchAPI, inspectionAPI } from '../api'
import { BatchStatusBadge } from '../components/common/BatchStatusBadge'
import { EntityTable } from '../components/common/EntityTable'
import { StatusBadge } from '../components/common/StatusBadge'
import { useAuth } from '../hooks/useAuth'
import { usePagination } from '../hooks/usePagination'
import { useInspectionStore } from '../stores/inspectionStore'
import type { InspectionSample, ProductionBatch } from '../types/domain'
import { formatDateTime } from '../utils/format'

export function InspectionsPage() {
  const { data, loading, load } = useInspectionStore()
  const pagination = usePagination()
  const { can } = useAuth()
  const [batches, setBatches] = useState<ProductionBatch[]>([])
  const [search, setSearch] = useState('')
  const [result, setResult] = useState<string>()
  const [createOpen, setCreateOpen] = useState(false)
  const [completeTarget, setCompleteTarget] = useState<InspectionSample | null>(null)
  const [saving, setSaving] = useState(false)
  const [createForm] = Form.useForm()
  const [completeForm] = Form.useForm()
  const refresh = () => load({ page: pagination.page, pageSize: pagination.pageSize, search, result })
  useEffect(() => { void refresh() }, [pagination.page, pagination.pageSize, result])
  useEffect(() => { void batchAPI.list({ page: 1, pageSize: 100 }).then((value) => setBatches(value.items.filter((batch) => ['running', 'hold', 'rework'].includes(batch.status)))) }, [])
  const create = async () => {
    const values = await createForm.validateFields()
    setSaving(true)
    try { await inspectionAPI.create({ ...values, notes: values.notes || '' }); message.success('检验样本已登记'); setCreateOpen(false); createForm.resetFields(); await refresh() } finally { setSaving(false) }
  }
  const complete = async () => {
    if (!completeTarget) return
    const values = await completeForm.validateFields()
    setSaving(true)
    try { await inspectionAPI.complete(completeTarget.id, values); message.success('检验结果已提交'); setCompleteTarget(null); completeForm.resetFields(); await refresh() } finally { setSaving(false) }
  }
  const columns: ColumnsType<InspectionSample> = [
    { title: '样本编号', dataIndex: 'sampleCode', fixed: 'left' },
    { title: '生产批次', render: (_, row) => <Space>{row.productionBatch?.batchNo || row.productionBatchId}{row.productionBatch && <BatchStatusBadge status={row.productionBatch.status} />}</Space> },
    { title: '抽样位置', dataIndex: 'samplingPosition' },
    { title: '检验项', dataIndex: 'inspectionItem' },
    { title: '接受范围', dataIndex: 'acceptanceRange' },
    { title: '测量值', dataIndex: 'measuredValue', render: (value) => value || '-' },
    { title: '结果', dataIndex: 'result', render: (value) => <StatusBadge value={value} /> },
    { title: '复测状态', dataIndex: 'retestStatus', render: (value) => <StatusBadge value={value} /> },
    { title: '检验员/时间', render: (_, row) => <div>{row.inspectorName || '-'}<small className="cell-subtitle">{formatDateTime(row.inspectedAt)}</small></div> },
    { title: '操作', fixed: 'right', render: (_, row) => (row.result === 'pending' || row.retestStatus === 'requested') && <Button size="small" type="primary" icon={<CheckOutlined />} disabled={!can('inspection:write')} onClick={() => setCompleteTarget(row)}>录入结果</Button> },
  ]
  return (
    <div className="page-stack">
      <header className="page-header"><div><Typography.Title level={2}>检验工作台</Typography.Title><Typography.Text type="secondary">登记抽样、录入检验结果并管理复测</Typography.Text></div><Button type="primary" icon={<PlusOutlined />} disabled={!can('inspection:write')} onClick={() => setCreateOpen(true)}>登记样本</Button></header>
      <div className="table-toolbar"><Input allowClear prefix={<SearchOutlined />} placeholder="搜索样本、检验项或位置" value={search} onChange={(event) => setSearch(event.target.value)} onPressEnter={() => void refresh()} /><Select allowClear placeholder="全部结果" value={result} onChange={setResult} options={[{ value: 'pending', label: '待检验' }, { value: 'pass', label: '合格' }, { value: 'fail', label: '不合格' }]} /><Button onClick={() => void refresh()}>查询</Button></div>
      <EntityTable columns={columns} dataSource={data.items} loading={loading} emptyTitle="暂无检验样本" pagination={{ current: pagination.page, pageSize: pagination.pageSize, total: data.total, onChange: pagination.update, showSizeChanger: true }} />
      <Modal title="登记检验样本" width={640} open={createOpen} confirmLoading={saving} onOk={() => void create()} onCancel={() => setCreateOpen(false)} okText="登记" cancelText="取消">
        <Form form={createForm} layout="vertical"><Form.Item name="productionBatchId" label="生产批次" rules={[{ required: true }]}><Select showSearch optionFilterProp="label" options={batches.map((batch) => ({ value: batch.id, label: `${batch.batchNo} · ${batch.specification}` }))} /></Form.Item><Row gutter={16}><Col span={12}><Form.Item name="sampleCode" label="样本编号" rules={[{ required: true, min: 3 }]}><Input /></Form.Item></Col><Col span={12}><Form.Item name="samplingPosition" label="抽样位置" rules={[{ required: true }]}><Input placeholder="起始/中段/末段" /></Form.Item></Col></Row><Form.Item name="inspectionItem" label="检验项目" rules={[{ required: true }]}><Input placeholder="密封强度、染色渗透等" /></Form.Item><Form.Item name="acceptanceRange" label="接受范围" rules={[{ required: true }]}><Input placeholder="≥ 1.5 N/15mm" /></Form.Item><Form.Item name="notes" label="备注"><Input.TextArea rows={2} /></Form.Item></Form>
      </Modal>
      <Modal title={`录入结果 · ${completeTarget?.sampleCode || ''}`} open={Boolean(completeTarget)} confirmLoading={saving} onOk={() => void complete()} onCancel={() => setCompleteTarget(null)} okText="提交" cancelText="取消">
        <Form form={completeForm} layout="vertical" initialValues={{ result: 'pass' }}><Form.Item name="result" label="检验结果" rules={[{ required: true }]}><Radio.Group optionType="button" buttonStyle="solid" options={[{ label: '合格', value: 'pass' }, { label: '不合格', value: 'fail' }]} /></Form.Item><Form.Item name="measuredValue" label="测量值/结论" rules={[{ required: true }]}><Input /></Form.Item><Form.Item name="notes" label="检验说明"><Input.TextArea rows={3} /></Form.Item></Form>
      </Modal>
    </div>
  )
}

