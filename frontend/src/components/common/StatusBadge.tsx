import { Badge, Tag } from 'antd'

const colors: Record<string, string> = {
  ready: 'success', running: 'processing', maintenance: 'warning', fault: 'error',
  pending: 'default', pass: 'success', fail: 'error',
  none: 'default', requested: 'warning', completed: 'success',
  release: 'success', quarantine: 'error', rework: 'warning',
}

const labels: Record<string, string> = {
  ready: '待机', running: '运行中', maintenance: '维护中', fault: '故障',
  pending: '待检验', pass: '合格', fail: '不合格', none: '无需复测', requested: '待复测', completed: '复测完成',
  release: '放行', quarantine: '隔离', rework: '返工',
}

export function StatusBadge({ value, dot = false }: { value: string; dot?: boolean }) {
  if (dot) return <Badge status={colors[value] as 'success' | 'processing' | 'warning' | 'error' | 'default'} text={labels[value] || value} />
  return <Tag color={colors[value]}>{labels[value] || value}</Tag>
}

