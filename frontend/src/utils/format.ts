import dayjs from 'dayjs'

export const formatDateTime = (value?: string) => value ? dayjs(value).format('YYYY-MM-DD HH:mm') : '-'
export const formatNumber = (value?: number) => new Intl.NumberFormat('zh-CN').format(value ?? 0)

export const equipmentLabels: Record<string, string> = {
  ready: '待机', running: '运行中', maintenance: '维护中', fault: '故障',
}

export const actionLabels: Record<string, string> = {
  'line.created': '创建产线', 'line.updated': '更新产线',
  'batch.created': '创建批次', 'batch.updated': '更新批次', 'batch.transitioned': '变更批次状态',
  'inspection.created': '创建检验', 'inspection.completed': '完成检验',
  'inspection.retest_requested': '申请复测', 'release.decided': '提交放行决定',
}

