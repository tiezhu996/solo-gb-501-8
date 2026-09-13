import { Alert, Button, Descriptions, Form, Input, Modal, Segmented, Space, Typography } from 'antd'
import { CheckCircleOutlined, StopOutlined, ToolOutlined } from '@ant-design/icons'
import { useState } from 'react'
import type { DecisionType, ProductionBatch } from '../../types/domain'
import { BatchStatusBadge } from './BatchStatusBadge'

interface Props {
  batch: ProductionBatch
  canDecide: boolean
  loading?: boolean
  onDecide: (decision: DecisionType, reason: string) => Promise<void>
}

export function DecisionPanel({ batch, canDecide, loading, onDecide }: Props) {
  const [open, setOpen] = useState(false)
  const [decision, setDecision] = useState<DecisionType>('release')
  const [form] = Form.useForm<{ reason: string }>()
  const pending = batch.inspections?.filter((item) => item.result === 'pending' || item.retestStatus === 'requested').length || 0
  const failed = batch.inspections?.filter((item) => item.result === 'fail').length || 0
  const submit = async () => {
    const values = await form.validateFields()
    await onDecide(decision, values.reason)
    setOpen(false)
    form.resetFields()
  }
  return (
    <section className="decision-panel">
      <div className="panel-heading"><div><Typography.Title level={4}>放行判定</Typography.Title><Typography.Text type="secondary">{batch.batchNo}</Typography.Text></div><BatchStatusBadge status={batch.status} /></div>
      <Descriptions column={2} size="small">
        <Descriptions.Item label="检验总数">{batch.inspections?.length || 0}</Descriptions.Item>
        <Descriptions.Item label="待处理">{pending}</Descriptions.Item>
        <Descriptions.Item label="不合格">{failed}</Descriptions.Item>
        <Descriptions.Item label="责任班组">{batch.responsibleTeam}</Descriptions.Item>
      </Descriptions>
      {(pending > 0 || failed > 0) && <Alert className="panel-alert" type="warning" showIcon message={`当前有 ${pending} 项待处理、${failed} 项不合格，不能直接放行`} />}
      <Button type="primary" disabled={!canDecide || batch.status === 'released'} onClick={() => setOpen(true)}>提交决定</Button>
      <Modal title={`审批批次 ${batch.batchNo}`} open={open} confirmLoading={loading} onOk={() => void submit()} onCancel={() => setOpen(false)} okText="确认提交" cancelText="取消">
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <Segmented block value={decision} onChange={(value) => setDecision(value as DecisionType)} options={[
            { label: '放行', value: 'release', icon: <CheckCircleOutlined /> },
            { label: '隔离', value: 'quarantine', icon: <StopOutlined /> },
            { label: '返工', value: 'rework', icon: <ToolOutlined /> },
          ]} />
          <Form form={form} layout="vertical"><Form.Item label="审批理由" name="reason" rules={[{ required: true, min: 5, message: '请填写至少 5 个字的审批理由' }]}><Input.TextArea rows={4} maxLength={1000} showCount /></Form.Item></Form>
        </Space>
      </Modal>
    </section>
  )
}

