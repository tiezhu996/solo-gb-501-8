import { HistoryOutlined, SafetyCertificateOutlined } from '@ant-design/icons'
import { Button, Col, Row, Segmented, Space, Typography, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useEffect, useState } from 'react'
import { batchAPI, releaseAPI } from '../api'
import { BatchStatusBadge } from '../components/common/BatchStatusBadge'
import { DecisionPanel } from '../components/common/DecisionPanel'
import { EntityTable } from '../components/common/EntityTable'
import { StatusBadge } from '../components/common/StatusBadge'
import { useAuth } from '../hooks/useAuth'
import { useReleaseStore } from '../stores/releaseStore'
import type { DecisionType, ProductionBatch, ReleaseDecision } from '../types/domain'
import { formatDateTime } from '../utils/format'

export function ReleasePage() {
  const { can } = useAuth()
  const history = useReleaseStore()
  const [mode, setMode] = useState<'queue' | 'history'>('queue')
  const [batches, setBatches] = useState<ProductionBatch[]>([])
  const [selected, setSelected] = useState<ProductionBatch | null>(null)
  const [loading, setLoading] = useState(false)
  const loadQueue = async () => {
    setLoading(true)
    try {
      const response = await batchAPI.list({ page: 1, pageSize: 100 })
      const pending = response.items.filter((batch) => ['running', 'hold', 'rework'].includes(batch.status))
      setBatches(pending)
      setSelected((current) => pending.find((batch) => batch.id === current?.id) || pending[0] || null)
    } finally { setLoading(false) }
  }
  useEffect(() => { void loadQueue(); void history.load({ page: 1, pageSize: 100 }) }, [])
  const decide = async (decision: DecisionType, reason: string) => { if (!selected) return; await releaseAPI.decide(selected.id, decision, reason); message.success('放行决定已记录'); await Promise.all([loadQueue(), history.load({ page: 1, pageSize: 100 })]) }
  const batchColumns: ColumnsType<ProductionBatch> = [
    { title: '批次', dataIndex: 'batchNo', render: (value, row) => <Button className="table-link" type="link" onClick={() => setSelected(row)}>{value}</Button> },
    { title: '规格', dataIndex: 'specification' },
    { title: '状态', dataIndex: 'status', render: (value) => <BatchStatusBadge status={value} /> },
    { title: '检验进度', render: (_, row) => `${row.inspections?.filter((sample) => sample.result !== 'pending').length || 0}/${row.inspections?.length || 0}` },
    { title: '不合格', render: (_, row) => row.inspections?.filter((sample) => sample.result === 'fail').length || 0 },
  ]
  const historyColumns: ColumnsType<ReleaseDecision> = [
    { title: '批次', render: (_, row) => row.productionBatch?.batchNo || row.productionBatchId },
    { title: '决定', dataIndex: 'decision', render: (value) => <StatusBadge value={value} /> },
    { title: '审批人', dataIndex: 'approverName' },
    { title: '理由', dataIndex: 'reason', ellipsis: true },
    { title: '检验摘要', dataIndex: 'inspectionSummary' },
    { title: '生效时间', dataIndex: 'effectiveAt', render: formatDateTime },
  ]
  return (
    <div className="page-stack">
      <header className="page-header"><div><Typography.Title level={2}>放行审批</Typography.Title><Typography.Text type="secondary">综合生产和检验事实做出放行、隔离或返工决定</Typography.Text></div><Segmented value={mode} onChange={(value) => setMode(value as typeof mode)} options={[{ value: 'queue', label: '待审批', icon: <SafetyCertificateOutlined /> }, { value: 'history', label: '决定记录', icon: <HistoryOutlined /> }]} /></header>
      {mode === 'queue' ? <Row gutter={20}><Col xs={24} xl={14}><EntityTable columns={batchColumns} dataSource={batches} loading={loading} pagination={false} emptyTitle="当前没有待审批批次" rowClassName={(row) => row.id === selected?.id ? 'selected-row' : ''} onRow={(row) => ({ onClick: () => setSelected(row) })} /></Col><Col xs={24} xl={10}>{selected ? <DecisionPanel batch={selected} canDecide={can('release:write')} onDecide={decide} /> : <div className="detail-section"><Typography.Text type="secondary">选择一个批次查看判定依据</Typography.Text></div>}</Col></Row> : <EntityTable columns={historyColumns} dataSource={history.data.items} loading={history.loading} pagination={false} emptyTitle="暂无放行决定" />}
    </div>
  )
}

