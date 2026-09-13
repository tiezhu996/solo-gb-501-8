import { Empty, Button } from 'antd'
import { PlusOutlined } from '@ant-design/icons'

export function EmptyState({ title = '暂无数据', actionLabel, onAction }: { title?: string; actionLabel?: string; onAction?: () => void }) {
  return (
    <Empty description={title} image={Empty.PRESENTED_IMAGE_SIMPLE}>
      {actionLabel && onAction && <Button type="primary" icon={<PlusOutlined />} onClick={onAction}>{actionLabel}</Button>}
    </Empty>
  )
}

