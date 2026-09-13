import { ArrowLeftOutlined } from '@ant-design/icons'
import { Button, Descriptions, Divider, Spin, Typography, message } from 'antd'
import { useEffect, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { batchAPI, releaseAPI } from '../api'
import { BatchStatusBadge } from '../components/common/BatchStatusBadge'
import { DecisionPanel } from '../components/common/DecisionPanel'
import { EntityTable } from '../components/common/EntityTable'
import { StatusBadge } from '../components/common/StatusBadge'
import { useAuth } from '../hooks/useAuth'
import type { DecisionType, InspectionSample, ProductionBatch } from '../types/domain'
import { formatDateTime, formatNumber } from '../utils/format'

export function BatchDetailPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { can } = useAuth()
  const [batch, setBatch] = useState<ProductionBatch | null>(null)
  const [loading, setLoading] = useState(true)
  const load = async () => { setLoading(true); try { setBatch(await batchAPI.get(Number(id))) } finally { setLoading(false) } }
  useEffect(() => { void load() }, [id])
  if (loading || !batch) return <Spin fullscreen />
  const decide = async (decision: DecisionType, reason: string) => { await releaseAPI.decide(batch.id, decision, reason); message.success('审批决定已提交'); await load() }
  return (
    <div className="page-stack">
      <header className="page-header"><div><Button type="text" icon={<ArrowLeftOutlined />} onClick={() => navigate('/batches')}>返回批次队列</Button><Typography.Title level={2}>{batch.batchNo}</Typography.Title></div><BatchStatusBadge status={batch.status} /></header>
      <section className="detail-section"><Typography.Title level={4}>批次信息</Typography.Title><Descriptions column={{ xs: 1, sm: 2, lg: 3 }} bordered size="small"><Descriptions.Item label="规格">{batch.specification}</Descriptions.Item><Descriptions.Item label="责任班组">{batch.responsibleTeam}</Descriptions.Item><Descriptions.Item label="包装产线">{batch.packagingLine?.name || batch.packagingLineId}</Descriptions.Item><Descriptions.Item label="生产数量">{formatNumber(batch.producedQuantity)} / {formatNumber(batch.plannedQuantity)}</Descriptions.Item><Descriptions.Item label="开始时间">{formatDateTime(batch.startedAt)}</Descriptions.Item><Descriptions.Item label="完成时间">{formatDateTime(batch.completedAt)}</Descriptions.Item>{batch.holdReason && <Descriptions.Item label="暂停/处置原因" span={3}>{batch.holdReason}</Descriptions.Item>}</Descriptions></section>
      <section className="detail-section"><Typography.Title level={4}>检验明细</Typography.Title><EntityTable<InspectionSample> size="small" pagination={false} dataSource={batch.inspections || []} columns={[{ title: '样本', dataIndex: 'sampleCode' }, { title: '抽样位置', dataIndex: 'samplingPosition' }, { title: '检验项', dataIndex: 'inspectionItem' }, { title: '结果', dataIndex: 'result', render: (value) => <StatusBadge value={value} /> }, { title: '测量值', dataIndex: 'measuredValue' }, { title: '接受范围', dataIndex: 'acceptanceRange' }, { title: '复测', dataIndex: 'retestStatus', render: (value) => <StatusBadge value={value} /> }]} /></section>
      <Divider />
      <DecisionPanel batch={batch} canDecide={can('release:write')} onDecide={decide} />
    </div>
  )
}

