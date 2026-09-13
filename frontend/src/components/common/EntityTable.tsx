import { Table } from 'antd'
import type { TableProps } from 'antd'
import { EmptyState } from './EmptyState'

interface Props<T extends object> extends TableProps<T> {
  emptyTitle?: string
  emptyActionLabel?: string
  onEmptyAction?: () => void
}

export function EntityTable<T extends object>({ emptyTitle, emptyActionLabel, onEmptyAction, ...props }: Props<T>) {
  return (
    <Table<T>
      rowKey={(record) => String((record as { id: number }).id)}
      size="middle"
      locale={{ emptyText: <EmptyState title={emptyTitle} actionLabel={emptyActionLabel} onAction={onEmptyAction} /> }}
      scroll={{ x: 'max-content' }}
      {...props}
    />
  )
}

