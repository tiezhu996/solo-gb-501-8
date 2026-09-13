import { Timeline, Typography } from 'antd'
import type { InspectionRetest, InspectionSample } from '../../types/domain'
import { formatDateTime } from '../../utils/format'
import { EmptyState } from './EmptyState'
import { StatusBadge } from './StatusBadge'

function roundTitle(record: InspectionRetest) {
  return record.round === 'initial' ? '首检' : `第 ${record.sequence - 1} 次复测`
}

// RetestHistory 按发生顺序展示样本的首检和历次复测：测量值、结论、检验人、
// 时间和备注。样本上的同名字段始终是最新一次结论，历史来自 inspection_retests。
export function RetestHistory({ sample }: { sample: InspectionSample }) {
  const records = [...(sample.retests || [])].sort((a, b) => a.sequence - b.sequence)
  if (records.length === 0) {
    if (sample.result === 'pending') return <EmptyState title="暂无检验记录" />
    // 兜底：历史数据尚未生成轮次记录时，用样本当前字段合成一条首检
    records.push({
      id: 0, createdAt: sample.createdAt, updatedAt: sample.updatedAt,
      inspectionSampleId: sample.id, round: 'initial', sequence: 1,
      result: sample.result as 'pass' | 'fail', measuredValue: sample.measuredValue || '',
      inspectorId: sample.inspectorId, inspectorName: sample.inspectorName,
      inspectedAt: sample.inspectedAt, notes: sample.notes,
    })
  }
  return (
    <Timeline
      items={records.map((record) => ({
        key: record.id || record.sequence,
        color: record.result === 'pass' ? 'green' : 'red',
        children: (
          <div>
            <div>
              <Typography.Text strong>{roundTitle(record)}</Typography.Text>{' '}
              <StatusBadge value={record.result} />
            </div>
            <div>测量值：{record.measuredValue || '-'}</div>
            <div>检验人：{record.inspectorName || '-'} · {formatDateTime(record.inspectedAt)}</div>
            {record.notes && <Typography.Paragraph type="secondary" style={{ marginBottom: 0, whiteSpace: 'pre-wrap' }}>{record.notes}</Typography.Paragraph>}
          </div>
        ),
      }))}
    />
  )
}
