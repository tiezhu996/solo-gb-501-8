import { Tag } from 'antd'
import type { BatchStatus } from '../../types/domain'

const config: Record<BatchStatus, { label: string; color: string }> = {
  draft: { label: '草稿', color: 'default' },
  running: { label: '生产中', color: 'processing' },
  hold: { label: '暂停/隔离', color: 'error' },
  rework: { label: '返工', color: 'warning' },
  released: { label: '已放行', color: 'success' },
}

export function BatchStatusBadge({ status }: { status: BatchStatus }) {
  const item = config[status]
  return <Tag color={item.color}>{item.label}</Tag>
}

